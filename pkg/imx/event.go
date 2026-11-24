package imx

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bokee/global"
	"bokee/pkg/redisx"
)

const (
	EventMessageNew           = "message.new"
	EventMessageRead          = "message.read"
	EventMessageRecalled      = "message.recalled"
	EventConversationUnlocked = "conversation.unlocked"

	userChannelPrefix = "bokee:im:user:"
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
	At   int64  `json:"at"`
}

type NewMessagePayload struct {
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
	Status          string    `json:"status"`
	Unlocked        bool      `json:"unlocked"`
}

// RecallPayload 消息撤回事件：两端据此把原气泡替换为撤回提示
type RecallPayload struct {
	ConversationID uint `json:"conversationId"`
	MessageID      uint `json:"messageId"`
	SenderID       uint `json:"senderId"`
}

type ReadPayload struct {
	ConversationID uint `json:"conversationId"`
	ReaderID       uint `json:"readerId"`
	LastReadID     uint `json:"lastReadId"`
}

type UnlockedPayload struct {
	ConversationID uint      `json:"conversationId"`
	UnlockedAt     time.Time `json:"unlockedAt"`
}

// UserChannel 返回用户实时事件频道
func UserChannel(uid uint) string {
	return userChannelPrefix + strconv.FormatUint(uint64(uid), 10)
}

// ParseUserChannel 从频道名解析用户ID
func ParseUserChannel(channel string) (uint, bool) {
	if !strings.HasPrefix(channel, userChannelPrefix) {
		return 0, false
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(channel, userChannelPrefix), 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

// NewEvent 构造实时事件，At 统一取当前毫秒时间戳
func NewEvent(typ string, data any) Event {
	return Event{Type: typ, Data: data, At: time.Now().UnixMilli()}
}

// WsTicketKey WebSocket 一次性票据 Key（签发与消费共用，避免两侧拼接不一致）
func WsTicketKey(ticket string) string {
	return redisx.BuildKey("im", "ws", "ticket", ticket)
}

// MutualFollowKey 互关缓存 Key（用户对归一化）
func MutualFollowKey(u1, u2 uint) string {
	if u1 > u2 {
		u1, u2 = u2, u1
	}
	return redisx.BuildKey("im", "mutual",
		strconv.FormatUint(uint64(u1), 10),
		strconv.FormatUint(uint64(u2), 10))
}

// Publish 向多个用户频道发布同一事件（Pipeline 合并为一次 RTT）
func Publish(ctx context.Context, uids []uint, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("实时事件序列化失败: %w", err)
	}
	pipe := global.Redis.Pipeline()
	for _, uid := range uids {
		pipe.Publish(ctx, UserChannel(uid), payload)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("实时事件推送失败: %w", err)
	}
	return nil
}
