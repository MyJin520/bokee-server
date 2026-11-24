package basic

import "time"

// 消息类型常量
const (
	MessageTypeText  = "text"  // 文本消息（链接由前端识别渲染）
	MessageTypeImage = "image" // 图片消息，Content 存储图片访问地址
)

type IMMessage struct {
	ID              uint      `gorm:"primarykey;autoIncrement;comment:主键ID"`
	CreatedAt       time.Time `gorm:"column:created_at;comment:创建时间"`
	ConversationID  uint      `gorm:"not null;comment:会话ID"`
	SenderID        uint      `gorm:"not null;uniqueIndex:uk_msg_client,priority:1;comment:发送者ID"`
	ReceiverID      uint      `gorm:"not null;comment:接收者ID"`
	MsgType         string    `gorm:"size:20;not null;default:text;comment:消息类型(text/image)"`
	Content         string    `gorm:"type:text;not null;comment:消息内容"`
	ClientMsgID     string    `gorm:"size:64;not null;uniqueIndex:uk_msg_client,priority:2;comment:客户端幂等ID"`
	ReplyToID       uint      `gorm:"not null;default:0;comment:被引用消息ID，0 表示无引用"`
	ReplyToSenderID uint      `gorm:"not null;default:0;comment:被引用消息发送者ID快照"`
	ReplyToMsgType  string    `gorm:"size:20;not null;default:'';comment:被引用消息类型快照"`
	ReplyToContent  string    `gorm:"type:text;comment:被引用消息内容快照"`
	ReplyRecalled   bool      `gorm:"not null;default:false;comment:被引用消息是否已撤回（引用快照失效标记）"`
	Recalled        bool      `gorm:"not null;default:false;comment:是否已撤回"`
}

// NewIMMessage 创建消息记录（引用快照由调用方在查询到被引用消息后回填）
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
