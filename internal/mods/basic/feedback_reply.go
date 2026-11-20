package basic

// FeedbackReply 反馈回复（对话流中的一条消息）
// 所有登录用户都可在可见的反馈下回复，形成多人对话；管理员的回复额外打上 IsAdmin 标记
type FeedbackReply struct {
	BaseModel
	FeedbackID uint   `gorm:"not null;index:idx_reply_feedback,priority:1;comment:反馈ID"`
	UserID     uint   `gorm:"not null;index:idx_reply_user;comment:回复用户ID"`
	ParentID   uint   `gorm:"not null;default:0;comment:父回复ID，0 表示直接回复反馈"`
	Content    string `gorm:"type:text;not null;comment:回复内容"`
	IsAdmin    bool   `gorm:"not null;default:false;comment:写入时是否超管回复"`
}

func NewFeedbackReply(feedbackID, userID, parentID uint, content string, isAdmin bool) *FeedbackReply {
	return &FeedbackReply{
		FeedbackID: feedbackID,
		UserID:     userID,
		ParentID:   parentID,
		Content:    content,
		IsAdmin:    isAdmin,
	}
}

func (FeedbackReply) TableName() string {
	return "feedback_replies"
}
