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
		msgType = "text"
	}

	conv, err := s.conv.GetOrCreate(ctx, senderID, req.TargetID)
	if err != nil {
		return nil, err
	}

	unlockedByMutual := false
	if conv.Status == basic.ConversationStatusStranger {
		mutual, mErr := s.conv.IsMutualFollow(ctx, conv.UserAID, conv.UserBID)
		if mErr != nil {
			global.Log.Warn("私信互关检测失败，按陌生会话处理", zap.Error(mErr))
		} else if mutual {
			// 互关即使曾发满 3 条也应解锁
			if _, uErr := s.conv.UnlockIfStranger(ctx, conv.ID); uErr != nil {
				return nil, uErr
			}
			conv.Status = basic.ConversationStatusUnlocked
			conv.UnlockedAt = new(time.Now())
			unlockedByMutual = true
		}
	}

	msg := basic.NewIMMessage(conv.ID, senderID, conv.PeerID(senderID), msgType, content, req.ClientMsgID)
	unlockedByReply, err := s.persist(ctx, conv, msg, conf.StrangerQuota)
	if err != nil {
		if _, ok := errors.AsType[*QuotaExceededError](err); ok {
			return nil, err
		}
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			exist, qErr := s.findByClientMsgID(ctx, senderID, req.ClientMsgID)
			if qErr != nil {
				return nil, qErr
			}
			// 幂等重试：以库中实际状态与额度返回，避免本地状态过期
			var latest basic.Conversation
			if qErr = global.DB.WithContext(ctx).
				Select("status", "initiator_sent", "unlocked_at").
				Where("id = ?", conv.ID).First(&latest).Error; qErr == nil {
				conv.Status = latest.Status
				conv.InitiatorSent = latest.InitiatorSent
			}
			return response.NewMessageResp(exist, conv.Status, QuotaRemaining(conv, senderID)), nil
		}
		return nil, err
	}

	if unlockedByReply {
		conv.Status = basic.ConversationStatusUnlocked
		if conv.UnlockedAt == nil {
			now := time.Now()
			conv.UnlockedAt = &now
		}
	}
	// 并发场景下以库中额度为准，避免客户端剩余条数显示错误
	if conv.Status == basic.ConversationStatusStranger {
		var latest basic.Conversation
		if err := global.DB.WithContext(ctx).
			Select("status", "initiator_sent", "unlocked_at").
			Where("id = ?", conv.ID).First(&latest).Error; err == nil {
			conv.Status = latest.Status
			conv.InitiatorSent = latest.InitiatorSent
			if latest.UnlockedAt != nil {
				conv.UnlockedAt = latest.UnlockedAt
			}
		}
	}
	unlocked := unlockedByMutual || unlockedByReply

	resp := response.NewMessageResp(msg, conv.Status, QuotaRemaining(conv, senderID))
	resp.Unlocked = unlocked
	s.publishAfterCommit(ctx, conv, msg, unlocked)
	return resp, nil
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
				res := tx.Model(&basic.Conversation{}).
					Where("id = ? AND status = ?", conv.ID, basic.ConversationStatusStranger).
					Updates(map[string]any{
						"status":      basic.ConversationStatusUnlocked,
						"unlocked_at": time.Now(),
					})
				if res.Error != nil {
					return res.Error
				}
				conv.Status = basic.ConversationStatusUnlocked
				if res.RowsAffected > 0 {
					unlockedByReply = true
				}
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
		if _, ok := errors.AsType[*QuotaExceededError](err); ok {
			return false, err
		}
		if errors.Is(err, gorm.ErrDuplicatedKey) {
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

// findByClientMsgID 幂等重试时查询已存在消息
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
	pushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	event := imx.Event{
		Type: imx.EventMessageNew,
		Data: imx.NewMessagePayload{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			SenderID:       msg.SenderID,
			ReceiverID:     msg.ReceiverID,
			MsgType:        msg.MsgType,
			Content:        msg.Content,
			ClientMsgID:    msg.ClientMsgID,
			CreatedAt:      msg.CreatedAt,
			Status:         conv.Status,
			Unlocked:       unlocked,
		},
		At: time.Now().UnixMilli(),
	}
	if err := imx.Publish(pushCtx, []uint{conv.UserAID, conv.UserBID}, event); err != nil {
		global.Log.Warn("私信实时推送失败", zap.Uint("message_id", msg.ID), zap.Error(err))
	}

	if unlocked {
		unlockedAt := time.Now()
		if conv.UnlockedAt != nil {
			unlockedAt = *conv.UnlockedAt
		}
		unlockEvent := imx.Event{
			Type: imx.EventConversationUnlocked,
			Data: imx.UnlockedPayload{ConversationID: conv.ID, UnlockedAt: unlockedAt},
			At:   time.Now().UnixMilli(),
		}
		if err := imx.Publish(pushCtx, []uint{conv.UserAID, conv.UserBID}, unlockEvent); err != nil {
			global.Log.Warn("私信解锁事件推送失败", zap.Uint("conversation_id", conv.ID), zap.Error(err))
		}
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
	var finalRead uint
	var peerID uint

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

		lastRead := req.LastReadID
		if lastRead == 0 || lastRead > conv.LastMessageID {
			lastRead = conv.LastMessageID
		}

		updates := map[string]any{}
		var unread int64
		if conv.UserAID == uid {
			if lastRead < conv.UserALastReadID {
				lastRead = conv.UserALastReadID
			}
			if lastRead < conv.LastMessageID {
				if err := tx.Model(&basic.IMMessage{}).
					Where("conversation_id = ? AND sender_id = ? AND id > ?", conv.ID, peerID, lastRead).
					Count(&unread).Error; err != nil {
					return err
				}
			}
			updates["user_a_last_read_id"] = lastRead
			updates["user_a_unread"] = unread
		} else {
			if lastRead < conv.UserBLastReadID {
				lastRead = conv.UserBLastReadID
			}
			if lastRead < conv.LastMessageID {
				if err := tx.Model(&basic.IMMessage{}).
					Where("conversation_id = ? AND sender_id = ? AND id > ?", conv.ID, peerID, lastRead).
					Count(&unread).Error; err != nil {
					return err
				}
			}
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

	pushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	event := imx.Event{
		Type: imx.EventMessageRead,
		Data: imx.ReadPayload{
			ConversationID: req.ConversationID,
			ReaderID:       uid,
			LastReadID:     finalRead,
		},
		At: time.Now().UnixMilli(),
	}
	if err := imx.Publish(pushCtx, []uint{peerID, uid}, event); err != nil {
		global.Log.Warn("已读回执推送失败", zap.Uint("conversation_id", req.ConversationID), zap.Error(err))
	}
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

	resp := &response.UnreadResp{Conversations: make([]response.ConversationUnreadResp, 0, len(convs))}
	for i := range convs {
		unread := convs[i].UserAUnread
		if convs[i].UserAID != uid {
			unread = convs[i].UserBUnread
		}
		if unread <= 0 {
			continue
		}
		resp.Total += unread
		resp.Conversations = append(resp.Conversations, response.ConversationUnreadResp{
			ConversationID: convs[i].ID,
			Unread:         unread,
		})
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

	var received []basic.IMMessage
	if err := global.DB.WithContext(ctx).
		Where("receiver_id = ? AND id > ?", uid, afterID).
		Order("id ASC").Limit(limit + 1).Find(&received).Error; err != nil {
		global.Log.Error("查询离线接收消息失败", zap.Uint("user_id", uid), zap.Error(err))
		return nil, fmt.Errorf("离线消息加载失败，请稍后重试")
	}

	var sent []basic.IMMessage
	if err := global.DB.WithContext(ctx).
		Where("sender_id = ? AND id > ?", uid, afterID).
		Order("id ASC").Limit(limit + 1).Find(&sent).Error; err != nil {
		global.Log.Error("查询离线发送消息失败", zap.Uint("user_id", uid), zap.Error(err))
		return nil, fmt.Errorf("离线消息加载失败，请稍后重试")
	}

	merged := mergeMessages(received, sent)
	hasMore := len(merged) > limit
	if hasMore {
		merged = merged[:limit]
	}
	return response.NewMessageListResp(merged, hasMore), nil
}

// mergeMessages 合并两段升序消息
func mergeMessages(a, b []basic.IMMessage) []basic.IMMessage {
	out := make([]basic.IMMessage, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i].ID < b[j].ID:
			out = append(out, a[i])
			i++
		case b[j].ID < a[i].ID:
			out = append(out, b[j])
			j++
		default:
			out = append(out, a[i])
			i++
			j++
		}
	}
	out = append(out, a[i:]...)
	out = append(out, b[j:]...)
	return out
}
