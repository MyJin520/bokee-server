package basic

import "time"

type IMMessage struct {
	ID             uint      `gorm:"primarykey;autoIncrement;comment:主键ID"`
	CreatedAt      time.Time `gorm:"column:created_at;comment:创建时间"`
	ConversationID uint      `gorm:"not null;comment:会话ID"`
	SenderID       uint      `gorm:"not null;uniqueIndex:uk_msg_client,priority:1;comment:发送者ID"`
	ReceiverID     uint      `gorm:"not null;comment:接收者ID"`
	MsgType        string    `gorm:"size:20;not null;default:text;comment:消息类型"`
	Content        string    `gorm:"type:text;not null;comment:消息内容"`
	ClientMsgID    string    `gorm:"size:64;not null;uniqueIndex:uk_msg_client,priority:2;comment:客户端幂等ID"`
}

func NewIMMessage(conversationID, senderID, receiverID uint, msgType, content, clientMsgID string) *IMMessage {
	return &IMMessage{
		ConversationID: conversationID,
		SenderID:       senderID,
		ReceiverID:     receiverID,
		MsgType:        msgType,
		Content:        content,
		ClientMsgID:    clientMsgID,
	}
}

func (IMMessage) TableName() string {
	return "im_messages"
}
