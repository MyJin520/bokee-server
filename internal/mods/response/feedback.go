package response

import (
	"bokee/internal/mods/basic"
	"encoding/json"
	"time"
)

// FeedbackUserInfo 反馈提交者公开信息（脱敏：仅展示必要字段）
type FeedbackUserInfo struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// FeedbackListItemResp 反馈列表项响应结构体
type FeedbackListItemResp struct {
	ID        uint             `json:"id"`
	Type      string           `json:"type"`
	Title     string           `json:"title"`
	Content   string           `json:"content"`
	Images    []string         `json:"images"`
	PageURL   string           `json:"pageUrl"`
	Contact   string           `json:"contact"`
	Status    string           `json:"status"`
	Reply     string           `json:"reply"`
	RepliedAt *time.Time       `json:"repliedAt"`
	UserID    uint             `json:"userId"`
	User      FeedbackUserInfo `json:"user"`
	CreatedAt time.Time        `json:"createdAt"`
}

// FeedbackInfoResp 反馈详情响应结构体
type FeedbackInfoResp struct {
	FeedbackListItemResp
	UpdatedAt time.Time `json:"updatedAt"`
}

// FeedbackCountItem 反馈数量统计项
type FeedbackCountItem struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

// FeedbackSummaryResp 反馈统计响应结构体
type FeedbackSummaryResp struct {
	Total    int64               `json:"total"`
	Pending  int64               `json:"pending"`
	ByType   []FeedbackCountItem `json:"byType"`
	ByStatus []FeedbackCountItem `json:"byStatus"`
}

// NewFeedbackListItemResp 将反馈模型转换为列表响应结构体。
// canViewPrivate 为 true 时（超级管理员或反馈提交者本人）才返回页面地址与联系方式。
func NewFeedbackListItemResp(f basic.Feedback, user basic.User, canViewPrivate bool) *FeedbackListItemResp {
	resp := &FeedbackListItemResp{
		ID:        f.ID,
		Type:      f.Type,
		Title:     f.Title,
		Content:   f.Content,
		Images:    parseFeedbackImages(f.Images),
		Status:    f.Status,
		Reply:     f.Reply,
		RepliedAt: f.RepliedAt,
		UserID:    f.UserID,
		User: FeedbackUserInfo{
			ID:     user.ID,
			Name:   user.Name,
			Avatar: user.Avatar,
		},
		CreatedAt: f.CreatedAt,
	}
	if canViewPrivate {
		resp.PageURL = f.PageURL
		resp.Contact = f.Contact
	}
	return resp
}

// NewFeedbackInfoResp 将反馈模型转换为详情响应结构体
func NewFeedbackInfoResp(f basic.Feedback, user basic.User, canViewPrivate bool) *FeedbackInfoResp {
	return &FeedbackInfoResp{
		FeedbackListItemResp: *NewFeedbackListItemResp(f, user, canViewPrivate),
		UpdatedAt:            f.UpdatedAt,
	}
}

// parseFeedbackImages 解析截图 JSON 数组，数据异常时返回空数组而非 nil，保证前端拿到稳定的数组类型
func parseFeedbackImages(raw string) []string {
	images := make([]string, 0)
	if raw == "" {
		return images
	}
	if err := json.Unmarshal([]byte(raw), &images); err != nil {
		return make([]string, 0)
	}
	return images
}
