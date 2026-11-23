package basic

import "time"

const (
	ConversationStatusStranger = "stranger" // 陌生会话：发起方受额度限制
	ConversationStatusUnlocked = "unlocked" // 已解锁：永久不限量
)

type Conversation struct {
	BaseModel
	UserAID         uint       `gorm:"not null;uniqueIndex:uk_conv_pair,priority:1;comment:A端用户ID（较小ID）"`
	UserBID         uint       `gorm:"not null;uniqueIndex:uk_conv_pair,priority:2;comment:B端用户ID（较大ID）"`
	InitiatorID     uint       `gorm:"not null;comment:会话发起人ID"`
	Status          string     `gorm:"size:20;not null;default:stranger;comment:会话状态(stranger/unlocked)"`
	UnlockedAt      *time.Time `gorm:"comment:解锁时间"`
	InitiatorSent   int        `gorm:"not null;default:0;comment:陌生阶段发起人已发送条数"`
	LastMessageID   uint       `gorm:"not null;default:0;comment:最后一条消息ID"`
	LastMessageAt   *time.Time `gorm:"comment:最后消息时间；未发过快信时为 NULL，避免 MySQL 严格模式写入零值时间失败"`
	UserALastReadID uint       `gorm:"not null;default:0;comment:A端已读消息游标"`
	UserBLastReadID uint       `gorm:"not null;default:0;comment:B端已读消息游标"`
	UserAUnread     int        `gorm:"not null;default:0;comment:A端未读消息数"`
	UserBUnread     int        `gorm:"not null;default:0;comment:B端未读消息数"`
}

func NewConversation(userAID, userBID, initiatorID uint) *Conversation {
	return &Conversation{
		UserAID:     userAID,
		UserBID:     userBID,
		InitiatorID: initiatorID,
		Status:      ConversationStatusStranger,
	}
}

func (Conversation) TableName() string {
	return "conversations"
}

// CanonicalPair 归一化用户对，保证 (a,b) 与 (b,a) 映射到同一会话
func CanonicalPair(u1, u2 uint) (uint, uint) {
	if u1 < u2 {
		return u1, u2
	}
	return u2, u1
}

// PeerID 返回会话中另一方的用户ID
func (c Conversation) PeerID(uid uint) uint {
	if c.UserAID == uid {
		return c.UserBID
	}
	return c.UserAID
}

// IsParticipant 判断用户是否为会话成员
func (c Conversation) IsParticipant(uid uint) bool {
	return c.UserAID == uid || c.UserBID == uid
}
