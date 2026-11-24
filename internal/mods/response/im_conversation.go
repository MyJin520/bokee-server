package response

import (
	"bokee/internal/mods/basic"
	"time"
)

type IMUserBrief struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type IMMessageBrief struct {
	ID             uint      `json:"id"`
	SenderID       uint      `json:"senderId"`
	MsgType        string    `json:"msgType"`
	Content        string    `json:"content"`
	ReplyToContent string    `json:"replyToContent"`
	Recalled       bool      `json:"recalled"`
	CreatedAt      time.Time `json:"createdAt"`
}

type ConversationResp struct {
	ID             uint            `json:"id"`
	Peer           IMUserBrief     `json:"peer"`
	Status         string          `json:"status"`
	CanSend        bool            `json:"canSend"`
	QuotaRemaining int             `json:"quotaRemaining"` // -1表示不限额
	LastMessage    *IMMessageBrief `json:"lastMessage,omitempty"`
	Unread         int             `json:"unread"`
	LastMessageAt  time.Time       `json:"lastMessageAt"`
}

func NewConversationResp(conv *basic.Conversation, peer basic.User, last *basic.IMMessage, uid uint, quotaRemaining int) *ConversationResp {
	unread := conv.UserAUnread
	if conv.UserAID != uid {
		unread = conv.UserBUnread
	}
	// 未发过消息时数据库为空值，响应里统一用零值时间（前端按「无时间」处理）
	var lastMessageAt time.Time
	if conv.LastMessageAt != nil {
		lastMessageAt = *conv.LastMessageAt
	}
	resp := &ConversationResp{
		ID:             conv.ID,
		Peer:           IMUserBrief{ID: peer.ID, Name: peer.Name, Avatar: peer.Avatar},
		Status:         conv.Status,
		CanSend:        conv.Status == basic.ConversationStatusUnlocked || conv.InitiatorID != uid || quotaRemaining > 0,
		QuotaRemaining: quotaRemaining,
		Unread:         unread,
		LastMessageAt:  lastMessageAt,
	}
	if last != nil {
		resp.LastMessage = &IMMessageBrief{
			ID:             last.ID,
			SenderID:       last.SenderID,
			MsgType:        last.MsgType,
			Content:        last.Content,
			ReplyToContent: last.ReplyToContent,
			Recalled:       last.Recalled,
			CreatedAt:      last.CreatedAt,
		}
	}
	return resp
}
