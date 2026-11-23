package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bokee/global"
	"bokee/internal/mods/basic"
	"bokee/internal/mods/request"
	"bokee/internal/mods/response"
	"bokee/pkg/imx"
	"bokee/pkg/redisx"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ConversationService struct{}

func NewConversationService() *ConversationService {
	return &ConversationService{}
}

// QuotaRemaining 返回当前用户在会话中的剩余额度，-1 表示不限
func QuotaRemaining(conv *basic.Conversation, uid uint) int {
	if conv == nil || conv.Status == basic.ConversationStatusUnlocked || conv.InitiatorID != uid {
		return -1
	}
	left := global.Config.IM.StrangerQuota - conv.InitiatorSent
	if left < 0 {
		left = 0
	}
	return left
}

// GetOrCreate 获取或创建用户对之间的唯一会话
func (s *ConversationService) GetOrCreate(ctx context.Context, senderID, targetID uint) (*basic.Conversation, error) {
	if senderID == targetID {
		return nil, fmt.Errorf("不能给自己发私信")
	}

	var target basic.User
	if err := global.DB.WithContext(ctx).Select("id", "status").Where("id = ?", targetID).First(&target).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Error("查询私信目标用户失败", zap.Uint("target_id", targetID), zap.Error(err))
		}
		return nil, fmt.Errorf("对方不存在或账号已被禁用")
	}
	if target.Status != "normal" {
		return nil, fmt.Errorf("对方不存在或账号已被禁用")
	}

	a, b := basic.CanonicalPair(senderID, targetID)
	var conv basic.Conversation
	err := global.DB.WithContext(ctx).
		Where("user_a_id = ? AND user_b_id = ?", a, b).
		First(&conv).Error
	if err == nil {
		return &conv, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		global.Log.Error("查询私信会话失败", zap.Error(err))
		return nil, fmt.Errorf("私信发送失败，请稍后重试")
	}

	newConv := basic.NewConversation(a, b, senderID)
	if err := global.DB.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(newConv).Error; err != nil {
		global.Log.Error("创建私信会话失败", zap.Error(err))
		return nil, fmt.Errorf("私信发送失败，请稍后重试")
	}

	// 并发首次发送时以唯一键为准，回查后返回
	var stored basic.Conversation
	if err := global.DB.WithContext(ctx).
		Where("user_a_id = ? AND user_b_id = ?", a, b).
		First(&stored).Error; err != nil {
		global.Log.Error("回查私信会话失败", zap.Error(err))
		return nil, fmt.Errorf("私信发送失败，请稍后重试")
	}
	return &stored, nil
}

// Create 打开（或创建）会话，供前端从作者主页发起私信
func (s *ConversationService) Create(ctx context.Context, req request.ConversationCreateReq, uid uint) (*response.ConversationResp, error) {
	conv, err := s.GetOrCreate(ctx, uid, req.TargetID)
	if err != nil {
		return nil, err
	}
	if conv.Status == basic.ConversationStatusStranger {
		mutual, mErr := s.IsMutualFollow(ctx, conv.UserAID, conv.UserBID)
		if mErr != nil {
			global.Log.Warn("互关检测失败，按陌生会话处理", zap.Error(mErr))
		} else if mutual {
			changed, uErr := s.UnlockIfStranger(ctx, conv.ID)
			if uErr != nil {
				return nil, uErr
			}
			if changed || conv.Status != basic.ConversationStatusUnlocked {
				conv.Status = basic.ConversationStatusUnlocked
				now := time.Now()
				conv.UnlockedAt = &now
			}
			if changed {
				s.publishUnlocked(ctx, conv)
			}
		}
	}
	return s.BuildResp(ctx, conv, uid)
}

// List 分页查询当前用户的会话列表
func (s *ConversationService) List(ctx context.Context, req request.ConversationListReq, uid uint) ([]response.ConversationResp, int64, error) {
	query := func() *gorm.DB {
		return global.DB.WithContext(ctx).Model(&basic.Conversation{}).
			Where("(user_a_id = ? OR user_b_id = ?) AND last_message_id > 0", uid, uid)
	}

	var total int64
	if err := query().Count(&total).Error; err != nil {
		global.Log.Error("统计私信会话数失败", zap.Uint("user_id", uid), zap.Error(err))
		return nil, 0, fmt.Errorf("会话列表加载失败，请稍后重试")
	}

	var convs []basic.Conversation
	if err := query().
		Order("last_message_at DESC, id DESC").
		Offset(req.Offset()).
		Limit(req.PageSize).
		Find(&convs).Error; err != nil {
		global.Log.Error("查询私信会话列表失败", zap.Uint("user_id", uid), zap.Error(err))
		return nil, 0, fmt.Errorf("会话列表加载失败，请稍后重试")
	}
	if len(convs) == 0 {
		return []response.ConversationResp{}, total, nil
	}

	peerIDs := make([]uint, 0, len(convs))
	lastIDs := make([]uint, 0, len(convs))
	for i := range convs {
		peerIDs = append(peerIDs, convs[i].PeerID(uid))
		if convs[i].LastMessageID > 0 {
			lastIDs = append(lastIDs, convs[i].LastMessageID)
		}
	}

	peerMap := make(map[uint]basic.User, len(peerIDs))
	var users []basic.User
	if err := global.DB.WithContext(ctx).Select("id", "name", "avatar").
		Where("id IN ?", peerIDs).Find(&users).Error; err != nil {
		global.Log.Warn("批量查询私信对端用户失败", zap.Error(err))
	} else {
		for _, u := range users {
			peerMap[u.ID] = u
		}
	}

	msgMap := make(map[uint]basic.IMMessage, len(lastIDs))
	if len(lastIDs) > 0 {
		var msgs []basic.IMMessage
		if err := global.DB.WithContext(ctx).Where("id IN ?", lastIDs).Find(&msgs).Error; err != nil {
			global.Log.Warn("批量查询会话最后消息失败", zap.Error(err))
		} else {
			for _, m := range msgs {
				msgMap[m.ID] = m
			}
		}
	}

	list := make([]response.ConversationResp, 0, len(convs))
	for i := range convs {
		conv := &convs[i]
		peerID := conv.PeerID(uid)
		peer, ok := peerMap[peerID]
		if !ok {
			peer.ID = peerID
			peer.Name = "已注销用户"
		}
		var last *basic.IMMessage
		if m, ok := msgMap[conv.LastMessageID]; ok {
			last = &m
		}
		resp := response.NewConversationResp(conv, peer, last, uid, QuotaRemaining(conv, uid))
		list = append(list, *resp)
	}
	return list, total, nil
}

// GetInfo 获取会话详情
func (s *ConversationService) GetInfo(ctx context.Context, convID, uid uint) (*response.ConversationResp, error) {
	var conv basic.Conversation
	if err := global.DB.WithContext(ctx).Where("id = ?", convID).First(&conv).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Error("查询私信会话失败", zap.Uint("conversation_id", convID), zap.Error(err))
		}
		return nil, fmt.Errorf("会话不存在或无权访问")
	}
	if !conv.IsParticipant(uid) {
		return nil, fmt.Errorf("会话不存在或无权访问")
	}

	// 互关用户打开会话即解锁，保证 canSend / 状态展示准确
	if conv.Status == basic.ConversationStatusStranger {
		if mutual, err := s.IsMutualFollow(ctx, conv.UserAID, conv.UserBID); err == nil && mutual {
			changed, uErr := s.UnlockIfStranger(ctx, conv.ID)
			if uErr != nil {
				return nil, uErr
			}
			if changed || conv.Status != basic.ConversationStatusUnlocked {
				conv.Status = basic.ConversationStatusUnlocked
				now := time.Now()
				conv.UnlockedAt = &now
			}
		}
	}
	return s.BuildResp(ctx, &conv, uid)
}

// IsMutualFollow 检测双方是否互相关注（仅缓存正结果）
func (s *ConversationService) IsMutualFollow(ctx context.Context, u1, u2 uint) (bool, error) {
	key := imx.MutualFollowKey(u1, u2)
	if v, err := redisx.Get(ctx, key); err == nil && v == "1" {
		return true, nil
	}

	var count int64
	err := global.DB.WithContext(ctx).Model(&basic.UserAction{}).
		Where("target_type = ? AND action_type = ? AND ((user_id = ? AND target_id = ?) OR (user_id = ? AND target_id = ?))",
			"author", "follow", u1, u2, u2, u1).
		Count(&count).Error
	if err != nil {
		global.Log.Error("互关检测失败", zap.Uint("user1", u1), zap.Uint("user2", u2), zap.Error(err))
		return false, fmt.Errorf("互关检测失败")
	}
	mutual := count == 2
	if mutual {
		if err := redisx.Set(ctx, key, "1", 5*time.Minute); err != nil {
			global.Log.Warn("写入互关缓存失败", zap.Error(err))
		}
	}
	return mutual, nil
}

// UnlockIfStranger 将陌生会话永久解锁，返回是否发生状态变更
func (s *ConversationService) UnlockIfStranger(ctx context.Context, convID uint) (bool, error) {
	res := global.DB.WithContext(ctx).Model(&basic.Conversation{}).
		Where("id = ? AND status = ?", convID, basic.ConversationStatusStranger).
		Updates(map[string]any{
			"status":      basic.ConversationStatusUnlocked,
			"unlocked_at": time.Now(),
		})
	if res.Error != nil {
		global.Log.Error("私信会话解锁失败", zap.Uint("conversation_id", convID), zap.Error(res.Error))
		return false, fmt.Errorf("私信会话解锁失败")
	}
	return res.RowsAffected > 0, nil
}

// BuildResp 组装会话响应（对端用户 + 最后一条消息）
func (s *ConversationService) BuildResp(ctx context.Context, conv *basic.Conversation, uid uint) (*response.ConversationResp, error) {
	peerID := conv.PeerID(uid)
	var peer basic.User
	if err := global.DB.WithContext(ctx).Select("id", "name", "avatar").
		Where("id = ?", peerID).First(&peer).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Error("查询私信对端用户失败", zap.Uint("peer_id", peerID), zap.Error(err))
			return nil, fmt.Errorf("会话信息加载失败，请稍后重试")
		}
		peer.ID = peerID
		peer.Name = "已注销用户"
	}

	var last *basic.IMMessage
	if conv.LastMessageID > 0 {
		var msg basic.IMMessage
		if err := global.DB.WithContext(ctx).Where("id = ?", conv.LastMessageID).First(&msg).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				global.Log.Warn("查询会话最后消息失败", zap.Uint("message_id", conv.LastMessageID), zap.Error(err))
			}
		} else {
			last = &msg
		}
	}
	return response.NewConversationResp(conv, peer, last, uid, QuotaRemaining(conv, uid)), nil
}

// publishUnlocked 互关触发解锁时通知双方
func (s *ConversationService) publishUnlocked(ctx context.Context, conv *basic.Conversation) {
	pushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	unlockedAt := time.Now()
	if conv.UnlockedAt != nil {
		unlockedAt = *conv.UnlockedAt
	}
	event := imx.Event{
		Type: imx.EventConversationUnlocked,
		Data: imx.UnlockedPayload{ConversationID: conv.ID, UnlockedAt: unlockedAt},
		At:   time.Now().UnixMilli(),
	}
	if err := imx.Publish(pushCtx, []uint{conv.UserAID, conv.UserBID}, event); err != nil {
		global.Log.Warn("私信解锁事件推送失败", zap.Uint("conversation_id", conv.ID), zap.Error(err))
	}
}
