package basic

type UserAction struct {
	ID         uint   `gorm:"primarykey;autoIncrement;comment:主键ID"`
	UserID     uint   `gorm:"not null;uniqueIndex:idx_ua_unique;index:idx_user_action;comment:当前用户ID"`
	TargetID   uint   `gorm:"not null;uniqueIndex:idx_ua_unique;comment:目标ID（文章ID或作者ID）"`
	TargetType string `gorm:"size:20;not null;uniqueIndex:idx_ua_unique;comment:目标类型(article/author)"`
	ActionType string `gorm:"size:20;not null;uniqueIndex:idx_ua_unique;index:idx_user_action;comment:动作类型(bookmark/like/follow)"`
}

func (UserAction) TableName() string {
	return "user_actions"
}

func NewUserAction(userID, targetID uint, targetType, actionType string) *UserAction {
	return &UserAction{
		UserID:     userID,     // 当前用户ID
		TargetID:   targetID,   // 目标ID（文章ID或作者ID）
		TargetType: targetType, // 目标类型(article || author)
		ActionType: actionType, // 动作类型(bookmark/like || follow)
	}
}
