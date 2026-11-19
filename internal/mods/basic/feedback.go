package basic

import "time"

// 反馈类型：使用字符串常量而非数据库枚举，新增类型时只需扩展常量与校验规则
const (
	FeedbackTypeBug      = "bug"      // 缺陷
	FeedbackTypeOptimize = "optimize" // 优化建议
	FeedbackTypeFeature  = "feature"  // 新功能建议
	FeedbackTypeOther    = "other"    // 其他
)

// 反馈处理状态
const (
	FeedbackStatusPending    = "pending"    // 待处理
	FeedbackStatusProcessing = "processing" // 处理中
	FeedbackStatusResolved   = "resolved"   // 已解决
	FeedbackStatusIgnored    = "ignored"    // 不处理
)

// Feedback 用户反馈模型
type Feedback struct {
	BaseModel
	UserID    uint       `gorm:"not null;index;comment:提交用户ID"`
	Type      string     `gorm:"size:20;not null;index:idx_feedback_type_status,priority:1;comment:反馈类型(bug/optimize/feature/other)"`
	Title     string     `gorm:"size:100;not null;comment:反馈标题"`
	Content   string     `gorm:"type:text;not null;comment:反馈内容"`
	Images    string     `gorm:"type:text;comment:截图地址，JSON数组字符串"`
	PageURL   string     `gorm:"size:500;comment:问题页面地址"`
	Contact   string     `gorm:"size:64;comment:联系方式"`
	Status    string     `gorm:"size:20;not null;default:pending;index:idx_feedback_type_status,priority:2;comment:处理状态(pending/processing/resolved/ignored)"`
	Reply     string     `gorm:"type:text;comment:管理员回复"`
	RepliedBy uint       `gorm:"comment:回复人ID"`
	RepliedAt *time.Time `gorm:"comment:回复时间"`
}

func NewFeedback(userID uint, typ, title, content, images, pageURL, contact string) *Feedback {
	return &Feedback{
		UserID:  userID,
		Type:    typ,
		Title:   title,
		Content: content,
		Images:  images,
		PageURL: pageURL,
		Contact: contact,
		Status:  FeedbackStatusPending,
	}
}

func (Feedback) TableName() string {
	return "feedbacks"
}
