package response

import (
	"time"

	"bokee/internal/mods/basic"
)

// CodeQuotaExceeded 私信额度耗尽业务码（HTTP 状态仍为 200）
const CodeQuotaExceeded = 4003

// MessageItemResp 单条消息（历史消息列表、离线同步返回）
type MessageItemResp struct {
	ID              uint      `json:"id"`
	ConversationID  uint      `json:"conversationId"`
	SenderID        uint      `json:"senderId"`
	ReceiverID      uint      `json:"receiverId"`
	MsgType         string    `json:"msgType"`
	Content         string    `json:"content"`
	ClientMsgID     string    `json:"clientMsgId"`
	ReplyToID       uint      `json:"replyToId"`
	ReplyToSenderID uint      `json:"replyToSenderId"`
	ReplyToMsgType  string    `json:"replyToMsgType"`
	ReplyToContent  string    `json:"replyToContent"`
	ReplyRecalled   bool      `json:"replyRecalled"`
	Recalled        bool      `json:"recalled"`
	CreatedAt       time.Time `json:"createdAt"`
}

func NewMessageItemResp(msg *basic.IMMessage) *MessageItemResp {
	return &MessageItemResp{
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
	}
}

// MessageResp 发送结果：消息条目 + 发送后的会话状态与额度
type MessageResp struct {
	MessageItemResp
	Status         string `json:"status"`         // 发送后的会话状态
	QuotaRemaining int    `json:"quotaRemaining"` // -1 表示不限
	Unlocked       bool   `json:"unlocked"`       // 本条消息是否触发解锁
}

func NewMessageResp(msg *basic.IMMessage, status string, quotaRemaining int) *MessageResp {
	return &MessageResp{
		MessageItemResp: *NewMessageItemResp(msg),
		Status:          status,
		QuotaRemaining:  quotaRemaining,
	}
}

type MessageListResp struct {
	List       []MessageItemResp `json:"list"`
	NextCursor uint              `json:"nextCursor"`
	HasMore    bool              `json:"hasMore"`
}

func NewMessageListResp(list []basic.IMMessage, hasMore bool) *MessageListResp {
	items := make([]MessageItemResp, 0, len(list))
	for i := range list {
		items = append(items, *NewMessageItemResp(&list[i]))
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

// NewUnreadResp 创建未读汇总，Conversations 初始化为空数组避免前端拿到 null
func NewUnreadResp() *UnreadResp {
	return &UnreadResp{Conversations: make([]ConversationUnreadResp, 0)}
}

// Add 累加一个会话的未读数
func (r *UnreadResp) Add(conversationID uint, unread int) {
	r.Total += unread
	r.Conversations = append(r.Conversations, ConversationUnreadResp{
		ConversationID: conversationID,
		Unread:         unread,
	})
}

type ReadResp struct {
	LastReadID uint `json:"lastReadId"`
}

func NewReadResp(lastReadID uint) *ReadResp {
	return &ReadResp{LastReadID: lastReadID}
}

type WsTicketResp struct {
	Ticket    string `json:"ticket"`
	ExpiresIn int    `json:"expiresIn"`
}

func NewWsTicketResp(ticket string, expiresIn int) *WsTicketResp {
	return &WsTicketResp{Ticket: ticket, ExpiresIn: expiresIn}
}

// MessageConfigResp 私信模块下发给前端的规则配置，避免前端硬编码与后端配置漂移
type MessageConfigResp struct {
	MaxContentLength int `json:"maxContentLength"`
	RecallWindowMin  int `json:"recallWindowMin"`
}

func NewMessageConfigResp(maxContentLength, recallWindowMin int) *MessageConfigResp {
	return &MessageConfigResp{MaxContentLength: maxContentLength, RecallWindowMin: recallWindowMin}
}
