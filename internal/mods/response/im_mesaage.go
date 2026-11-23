package response

import (
	"bokee/internal/mods/basic"
	"time"
)

// CodeQuotaExceeded 私信额度耗尽业务码（HTTP 状态仍为 200）
const CodeQuotaExceeded = 4003

type MessageResp struct {
	ID             uint      `json:"id"`
	ConversationID uint      `json:"conversationId"`
	SenderID       uint      `json:"senderId"`
	ReceiverID     uint      `json:"receiverId"`
	MsgType        string    `json:"msgType"`
	Content        string    `json:"content"`
	ClientMsgID    string    `json:"clientMsgId"`
	CreatedAt      time.Time `json:"createdAt"`
	Status         string    `json:"status"`         // 发送后的会话状态
	QuotaRemaining int       `json:"quotaRemaining"` // -1 表示不限
	Unlocked       bool      `json:"unlocked"`       // 本条消息是否触发解锁
}

func NewMessageResp(msg *basic.IMMessage, status string, quotaRemaining int) *MessageResp {
	return &MessageResp{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		ReceiverID:     msg.ReceiverID,
		MsgType:        msg.MsgType,
		Content:        msg.Content,
		ClientMsgID:    msg.ClientMsgID,
		CreatedAt:      msg.CreatedAt,
		Status:         status,
		QuotaRemaining: quotaRemaining,
	}
}

type MessageItemResp struct {
	ID             uint      `json:"id"`
	ConversationID uint      `json:"conversationId"`
	SenderID       uint      `json:"senderId"`
	ReceiverID     uint      `json:"receiverId"`
	MsgType        string    `json:"msgType"`
	Content        string    `json:"content"`
	ClientMsgID    string    `json:"clientMsgId"`
	CreatedAt      time.Time `json:"createdAt"`
}

func NewMessageItemResp(msg basic.IMMessage) MessageItemResp {
	return MessageItemResp{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		ReceiverID:     msg.ReceiverID,
		MsgType:        msg.MsgType,
		Content:        msg.Content,
		ClientMsgID:    msg.ClientMsgID,
		CreatedAt:      msg.CreatedAt,
	}
}

type MessageListResp struct {
	List       []MessageItemResp `json:"list"`
	NextCursor uint              `json:"nextCursor"`
	HasMore    bool              `json:"hasMore"`
}

func NewMessageListResp(list []basic.IMMessage, hasMore bool) *MessageListResp {
	items := make([]MessageItemResp, 0, len(list))
	for _, msg := range list {
		items = append(items, NewMessageItemResp(msg))
	}
	var nextCursor uint
	if len(list) > 0 {
		nextCursor = list[len(list)-1].ID
	}
	return &MessageListResp{List: items, NextCursor: nextCursor, HasMore: hasMore}
}

type ConversationUnreadResp struct {
	ConversationID uint `json:"conversationId"`
	Unread         int  `json:"unread"`
}

type UnreadResp struct {
	Total         int                      `json:"total"`
	Conversations []ConversationUnreadResp `json:"conversations"`
}

type ReadResp struct {
	LastReadID uint `json:"lastReadId"`
}

type WsTicketResp struct {
	Ticket    string `json:"ticket"`
	ExpiresIn int    `json:"expiresIn"`
}
