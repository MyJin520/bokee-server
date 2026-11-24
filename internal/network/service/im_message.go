package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"bokee/global"
	"bokee/internal/mods/basic"
	"bokee/internal/mods/request"
	"bokee/internal/mods/response"
	"bokee/pkg/imx"
	"bokee/pkg/limitx"
	"bokee/pkg/randx"
	"bokee/pkg/redisx"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// QuotaExceededError 陌生会话额度耗尽
type QuotaExceededError struct {
	Quota int
}

func (e *QuotaExceededError) Error() string {
	return fmt.Sprintf("对方回复前最多发送 %d 条消息，请等待对方回复后再试", e.Quota)
}

// IsQuotaExceeded 判断错误是否为陌生会话额度耗尽
func IsQuotaExceeded(err error) bool {
	_, ok := errors.AsType[*QuotaExceededError](err)
	return ok
}

type MessageService struct {
	conv *ConversationService
}

func NewMessageService() *MessageService {
	return &MessageService{conv: NewConversationService()}
}

// Send 发送私信
func (s *MessageService) Send(ctx context.Context, req request.MessageSendReq, senderID uint) (*response.MessageResp, error) {
	conf := global.Config.IM

	limiter := &limitx.RateLimiter{
		Scope:  "im:send",
		Window: time.Duration(conf.SendRateWindow) * time.Second,
		Max:    int64(conf.SendRateMax),
	}
	if !limiter.Allow(ctx, strconv.FormatUint(uint64(senderID), 10)) {
		return nil, fmt.Errorf("发送过于频繁，请稍后再试")
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, fmt.Errorf("消息内容不能为空")
	}
	if utf8.RuneCountInString(content) > conf.MaxContentLength {
		return nil, fmt.Errorf("消息内容不能超过 %d 个字符", conf.MaxContentLength)
	}
	msgType := req.MsgType
	if msgType == "" {
		msgType = basic.MessageTypeText
	}

	conv, err := s.conv.GetOrCreate(ctx, senderID, req.TargetID)
	if err != nil {
		return nil, err
	}

	// 互关即解锁；解锁事件随消息事件一并推送，避免重复推送
	unlockedByMutual, err := s.conv.unlockIfMutual(ctx, conv)
	if err != nil {
		return nil, err
	}

	msg := basic.NewIMMessage(conv.ID, senderID, conv.PeerID(senderID), msgType, content, req.ClientMsgID)
	if err := s.fillQuoteSnapshot(ctx, conv.ID, req.ReplyToID, msg); err != nil {
		return nil, err
	}
	unlockedByReply, err := s.persist(ctx, conv, msg, conf.StrangerQuota)
	if err != nil {
		if !errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, err
		}
		// 客户端重试：返回库中已落库的消息与当前额度，保证幂等
		exist, qErr := s.findByClientMsgID(ctx, senderID, req.ClientMsgID)
		if qErr != nil {
			return nil, qErr
		}
		s.refreshState(ctx, conv)
		return response.NewMessageResp(exist, conv.Status, QuotaRemaining(conv, senderID)), nil
	}

	if unlockedByReply {
		conv.MarkUnlocked(time.Now())
	}
	// 并发场景下以库中额度为准，避免客户端剩余条数显示错误
	if conv.Status == basic.ConversationStatusStranger {
		s.refreshState(ctx, conv)
	}
	unlocked := unlockedByMutual || unlockedByReply

	resp := response.NewMessageResp(msg, conv.Status, QuotaRemaining(conv, senderID))
	resp.Unlocked = unlocked
	s.publishAfterCommit(ctx, conv, msg, unlocked)
	return resp, nil
}

// fillQuoteSnapshot 解析被引用消息并复制快照；无引用时直接返回
func (s *MessageService) fillQuoteSnapshot(ctx context.Context, conversationID, replyToID uint, msg *basic.IMMessage) error {
	if replyToID == 0 {
		return nil
	}
	var quoted basic.IMMessage
	if err := global.DB.WithContext(ctx).
		Where("id = ? AND conversation_id = ?", replyToID, conversationID).
		First(&quoted).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Error("查询被引用消息失败",
				zap.Uint("conversation_id", conversationID),
				zap.Uint("reply_to_id", replyToID),
				zap.Error(err))
		}
		return fmt.Errorf("引用的消息不存在或已撤回")
	}
	if quoted.Recalled {
		return fmt.Errorf("引用的消息已撤回")
	}
	msg.ReplyToID = quoted.ID
	msg.ReplyToSenderID = quoted.SenderID
	msg.ReplyToMsgType = quoted.MsgType
	msg.ReplyToContent = quoted.Content
	return nil
}

// Recall 撤回本人在时间窗口内发送的消息，并通知会话双方
func (s *MessageService) Recall(ctx context.Context, req request.MessageRecallReq, uid uint) error {
	var msg basic.IMMessage
	if err := global.DB.WithContext(ctx).
		Where("id = ?", req.MessageID).
		First(&msg).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Error("查询待撤回消息失败", zap.Uint("message_id", req.MessageID), zap.Error(err))
		}
		return fmt.Errorf("消息不存在或已撤回")
	}
	if msg.SenderID != uid {
		return fmt.Errorf("只能撤回自己发送的消息")
	}
	if msg.Recalled {
		return fmt.Errorf("消息已撤回")
	}

	window := time.Duration(global.Config.IM.RecallWindowMin) * time.Minute
	if window <= 0 {
		window = 3 * time.Minute
	}
	if time.Since(msg.CreatedAt) > window {
		return fmt.Errorf("消息发送已超过 %d 分钟，无法撤回", int(window.Minutes()))
	}

	updates := map[string]any{
		"recalled":           true,
		"content":            "",
		"reply_to_id":        0,
		"reply_to_sender_id": 0,
		"reply_to_msg_type":  "",
		"reply_to_content":   "",
		"reply_recalled":     false,
	}

	err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&basic.IMMessage{}).
			Where("id = ? AND sender_id = ? AND recalled = ?", msg.ID, uid, false).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("消息不存在或已撤回")
		}
		// 引用了被撤回消息的其他消息：标记引用快照失效并清空残留内容
		return tx.Model(&basic.IMMessage{}).
			Where("reply_to_id = ? AND recalled = ?", msg.ID, false).
			Updates(map[string]any{
				"reply_recalled":    true,
				"reply_to_content":  "",
				"reply_to_msg_type": "",
			}).Error
	})
	if err != nil {
		global.Log.Error("消息撤回更新失败", zap.Uint("message_id", msg.ID), zap.Error(err))
		return fmt.Errorf("消息撤回失败，请稍后重试")
	}

	publish(ctx, []uint{msg.SenderID, msg.ReceiverID},
		imx.NewEvent(imx.EventMessageRecalled, imx.RecallPayload{
			ConversationID: msg.ConversationID,
			MessageID:      msg.ID,
			SenderID:       msg.SenderID,
		}),
		zap.Uint("message_id", msg.ID))
	return nil
}

// persist 在同一事务内完成额度校验、解锁、消息落库、未读数与会话摘要更新
func (s *MessageService) persist(ctx context.Context, conv *basic.Conversation, msg *basic.IMMessage, quota int) (bool, error) {
	unlockedByReply := false

	err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if conv.Status == basic.ConversationStatusStranger {
			if msg.SenderID == conv.InitiatorID {
				// 条件更新即原子扣减，并发不可能突破额度
				res := tx.Model(&basic.Conversation{}).
					Where("id = ? AND status = ? AND initiator_sent < ?",
						conv.ID, basic.ConversationStatusStranger, quota).
					UpdateColumn("initiator_sent", gorm.Expr("initiator_sent + 1"))
				if res.Error != nil {
					return res.Error
				}
				if res.RowsAffected == 0 {
					// 可能已在本事务前被对方回复解锁，复核一次
					var current basic.Conversation
					if err := tx.Model(&basic.Conversation{}).Select("status").
						Where("id = ?", conv.ID).First(&current).Error; err != nil {
						return err
					}
					if current.Status != basic.ConversationStatusUnlocked {
						return &QuotaExceededError{Quota: quota}
					}
					conv.Status = basic.ConversationStatusUnlocked
				}
			} else {
				// 被私信方发言 = 回复，永久解锁
				changed, err := unlockConversation(tx, conv.ID)
				if err != nil {
					return err
				}
				conv.Status = basic.ConversationStatusUnlocked
				unlockedByReply = changed
			}
		}

		if err := tx.Create(msg).Error; err != nil {
			return err
		}

		updates := map[string]any{
			"last_message_id": msg.ID,
			"last_message_at": msg.CreatedAt,
		}
		if msg.ReceiverID == conv.UserAID {
			updates["user_a_unread"] = gorm.Expr("user_a_unread + 1")
		} else {
			updates["user_b_unread"] = gorm.Expr("user_b_unread + 1")
		}
		return tx.Model(&basic.Conversation{}).Where("id = ?", conv.ID).Updates(updates).Error
	})
	if err != nil {
		// 额度耗尽与唯一键冲突属于预期结果，交由上层转换为业务响应
		if IsQuotaExceeded(err) || errors.Is(err, gorm.ErrDuplicatedKey) {
			return false, err
		}
		global.Log.Error("私信写入失败",
			zap.Uint("conversation_id", conv.ID),
			zap.Uint("sender_id", msg.SenderID),
			zap.Error(err))
		return false, fmt.Errorf("消息发送失败，请稍后重试")
	}
	return unlockedByReply, nil
}

// refreshState 以库中状态为准刷新本地会话，避免并发解锁后返回过期状态与额度
func (s *MessageService) refreshState(ctx context.Context, conv *basic.Conversation) {
	var latest basic.Conversation
	if err := global.DB.WithContext(ctx).
		Select("status", "initiator_sent", "unlocked_at").
		Where("id = ?", conv.ID).First(&latest).Error; err != nil {
		return
	}
	conv.Status = latest.Status
	conv.InitiatorSent = latest.InitiatorSent
	if latest.UnlockedAt != nil {
		conv.UnlockedAt = latest.UnlockedAt
	}
}

// findByClientMsgID 客户端重试时查询已落库消息
func (s *MessageService) findByClientMsgID(ctx context.Context, senderID uint, clientMsgID string) (*basic.IMMessage, error) {
	var msg basic.IMMessage
	if err := global.DB.WithContext(ctx).
		Where("sender_id = ? AND client_msg_id = ?", senderID, clientMsgID).
		First(&msg).Error; err != nil {
		global.Log.Error("查询幂等私信失败", zap.Uint("sender_id", senderID), zap.Error(err))
		return nil, fmt.Errorf("消息发送失败，请稍后重试")
	}
	return &msg, nil
}

// publishAfterCommit 提交成功后推送实时事件，失败只记日志
func (s *MessageService) publishAfterCommit(ctx context.Context, conv *basic.Conversation, msg *basic.IMMessage, unlocked bool) {
	publish(ctx, []uint{conv.UserAID, conv.UserBID},
		imx.NewEvent(imx.EventMessageNew, imx.NewMessagePayload{
			ID:              msg.ID,
			ConversationID:  msg.ConversationID,
			SenderID:        msg.SenderID,
			ReceiverID:      msg.ReceiverID,
			MsgType:         msg.MsgType,
			Content:         msg.Content,
			ClientMsgID:     msg.ClientMsgID,
			ReplyToID:       msg.ReplyToID,
			ReplyToSenderID: msg.ReplyToSenderID,
			ReplyToMsgType:  msg.ReplyToMsgType,
			ReplyToContent:  msg.ReplyToContent,
			ReplyRecalled:   msg.ReplyRecalled,
			Recalled:        msg.Recalled,
			CreatedAt:       msg.CreatedAt,
			Status:          conv.Status,
			Unlocked:        unlocked,
		}),
		zap.Uint("message_id", msg.ID))
	if unlocked {
		publishUnlocked(ctx, conv)
	}
}

// List 会话内历史消息（游标分页，倒序返回）
func (s *MessageService) List(ctx context.Context, req request.MessageListReq, uid uint) (*response.MessageListResp, error) {
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var conv basic.Conversation
	if err := global.DB.WithContext(ctx).
		Select("id", "user_a_id", "user_b_id").
		Where("id = ?", req.ConversationID).
		First(&conv).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Error("查询私信会话失败", zap.Uint("conversation_id", req.ConversationID), zap.Error(err))
		}
		return nil, fmt.Errorf("会话不存在或无权访问")
	}
	if !conv.IsParticipant(uid) {
		return nil, fmt.Errorf("会话不存在或无权访问")
	}

	query := global.DB.WithContext(ctx).Where("conversation_id = ?", req.ConversationID)
	if req.LastID > 0 {
		query = query.Where("id < ?", req.LastID)
	}
	var list []basic.IMMessage
	if err := query.Order("id DESC").Limit(pageSize + 1).Find(&list).Error; err != nil {
		global.Log.Error("查询私信消息失败", zap.Uint("conversation_id", req.ConversationID), zap.Error(err))
		return nil, fmt.Errorf("消息加载失败，请稍后重试")
	}
	hasMore := len(list) > pageSize
	if hasMore {
		list = list[:pageSize]
	}
	return response.NewMessageListResp(list, hasMore), nil
}

// MarkRead 标记已读到指定消息，返回最终已读游标
func (s *MessageService) MarkRead(ctx context.Context, req request.MessageReadReq, uid uint) (uint, error) {
	var finalRead, peerID uint

	err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conv basic.Conversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", req.ConversationID).
			First(&conv).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				global.Log.Error("查询私信会话失败", zap.Uint("conversation_id", req.ConversationID), zap.Error(err))
			}
			return fmt.Errorf("会话不存在或无权访问")
		}
		if !conv.IsParticipant(uid) {
			return fmt.Errorf("会话不存在或无权访问")
		}
		peerID = conv.PeerID(uid)

		// 已读游标只增不减：传 0 表示读到最新，且不允许回退
		lastRead := req.LastReadID
		if lastRead == 0 || lastRead > conv.LastMessageID {
			lastRead = conv.LastMessageID
		}
		if conv.UserAID == uid {
			if lastRead < conv.UserALastReadID {
				lastRead = conv.UserALastReadID
			}
		} else if lastRead < conv.UserBLastReadID {
			lastRead = conv.UserBLastReadID
		}

		// 对方发送且晚于新游标的消息即为剩余未读
		var unread int64
		if lastRead < conv.LastMessageID {
			if err := tx.Model(&basic.IMMessage{}).
				Where("conversation_id = ? AND sender_id = ? AND id > ?", conv.ID, peerID, lastRead).
				Count(&unread).Error; err != nil {
				return err
			}
		}

		updates := map[string]any{}
		if conv.UserAID == uid {
			updates["user_a_last_read_id"] = lastRead
			updates["user_a_unread"] = unread
		} else {
			updates["user_b_last_read_id"] = lastRead
			updates["user_b_unread"] = unread
		}
		if err := tx.Model(&basic.Conversation{}).Where("id = ?", conv.ID).Updates(updates).Error; err != nil {
			return err
		}
		finalRead = lastRead
		return nil
	})
	if err != nil {
		return 0, err
	}

	publish(ctx, []uint{peerID, uid},
		imx.NewEvent(imx.EventMessageRead, imx.ReadPayload{
			ConversationID: req.ConversationID,
			ReaderID:       uid,
			LastReadID:     finalRead,
		}),
		zap.Uint("conversation_id", req.ConversationID))
	return finalRead, nil
}

// Unread 未读汇总
func (s *MessageService) Unread(ctx context.Context, uid uint) (*response.UnreadResp, error) {
	var convs []basic.Conversation
	if err := global.DB.WithContext(ctx).
		Select("id", "user_a_id", "user_b_id", "user_a_unread", "user_b_unread").
		Where("user_a_id = ? OR user_b_id = ?", uid, uid).
		Find(&convs).Error; err != nil {
		global.Log.Error("查询私信未读数失败", zap.Uint("user_id", uid), zap.Error(err))
		return nil, fmt.Errorf("未读消息加载失败，请稍后重试")
	}

	resp := response.NewUnreadResp()
	for i := range convs {
		unread := convs[i].UserAUnread
		if convs[i].UserAID != uid {
			unread = convs[i].UserBUnread
		}
		if unread <= 0 {
			continue
		}
		resp.Add(convs[i].ID, unread)
	}
	return resp, nil
}

// Sync 离线同步：返回该用户收发的、ID 大于游标的消息（升序）
func (s *MessageService) Sync(ctx context.Context, afterID uint, limit int, uid uint) (*response.MessageListResp, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}

	var messages []basic.IMMessage
	if err := global.DB.WithContext(ctx).
		Where("id > ? AND (sender_id = ? OR receiver_id = ?)", afterID, uid, uid).
		Order("id ASC").Limit(limit + 1).Find(&messages).Error; err != nil {
		global.Log.Error("查询离线消息失败", zap.Uint("user_id", uid), zap.Error(err))
		return nil, fmt.Errorf("离线消息加载失败，请稍后重试")
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	return response.NewMessageListResp(messages, hasMore), nil
}

// WsTicket 生成 WebSocket 一次性连接凭证（票据仅存 Redis，握手时原子消费）
func (s *MessageService) WsTicket(ctx context.Context, uid uint) (*response.WsTicketResp, error) {
	ttlSeconds := 30
	if global.Config != nil {
		ttlSeconds = global.Config.IM.WsTicketTTL
	}

	ticket := randx.RandomUUID()
	if err := redisx.Set(ctx, imx.WsTicketKey(ticket), uid, time.Duration(ttlSeconds)*time.Second); err != nil {
		global.Log.Error("生成 WebSocket 连接凭证失败", zap.Uint("user_id", uid), zap.Error(err))
		return nil, fmt.Errorf("连接凭证生成失败，请稍后重试")
	}
	return response.NewWsTicketResp(ticket, ttlSeconds), nil
}

// Config 返回前端需要的私信规则配置（长度上限、撤回窗口等由后端配置单点决定）
func (s *MessageService) Config() *response.MessageConfigResp {
	window := global.Config.IM.RecallWindowMin
	if window <= 0 {
		window = 3
	}
	return response.NewMessageConfigResp(global.Config.IM.MaxContentLength, window)
}

// ---- 实时事件推送（私信收发、已读、解锁共用） ----

// pushTimeout 实时推送的独立超时：请求返回后仍需完成推送，但不可无限等待
const pushTimeout = 3 * time.Second

// publish 向指定用户推送实时事件；失败只记日志，不影响主流程
func publish(ctx context.Context, uids []uint, event imx.Event, fields ...zap.Field) {
	pushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), pushTimeout)
	defer cancel()

	if err := imx.Publish(pushCtx, uids, event); err != nil {
		fields = append([]zap.Field{zap.String("event", event.Type)}, fields...)
		global.Log.Warn("私信实时事件推送失败", fields...)
	}
}

// publishUnlocked 向会话双方推送解锁事件，保证两端状态实时一致
func publishUnlocked(ctx context.Context, conv *basic.Conversation) {
	unlockedAt := time.Now()
	if conv.UnlockedAt != nil {
		unlockedAt = *conv.UnlockedAt
	}
	publish(ctx, []uint{conv.UserAID, conv.UserBID},
		imx.NewEvent(imx.EventConversationUnlocked, imx.UnlockedPayload{
			ConversationID: conv.ID,
			UnlockedAt:     unlockedAt,
		}),
		zap.Uint("conversation_id", conv.ID))
}
