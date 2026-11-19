package request

// CreateFeedbackRequest 提交反馈请求体
type CreateFeedbackRequest struct {
	Type    string   `json:"type" label:"反馈类型" validate:"required,oneof=bug optimize feature other"`
	Title   string   `json:"title" label:"反馈标题" validate:"required,max=100"`
	Content string   `json:"content" label:"反馈内容" validate:"required,max=2000"`
	Images  []string `json:"images" label:"截图地址" validate:"omitempty,max=5,dive,max=500"`
	PageURL string   `json:"pageUrl" label:"问题页面地址" validate:"omitempty,max=500"`
	Contact string   `json:"contact" label:"联系方式" validate:"omitempty,max=64"`
}

// UpdateFeedbackRequest 处理反馈请求体（仅超级管理员可用）
type UpdateFeedbackRequest struct {
	ID     uint    `json:"id" label:"反馈ID" validate:"required"`
	Status *string `json:"status,omitempty" label:"处理状态" validate:"omitempty,oneof=pending processing resolved ignored"`
	Reply  *string `json:"reply,omitempty" label:"管理员回复" validate:"omitempty,max=2000"`
}

// FeedbackQueryListReq 反馈列表查询请求体
type FeedbackQueryListReq struct {
	Type    string `json:"type" label:"反馈类型" validate:"omitempty,oneof=bug optimize feature other"`
	Status  string `json:"status" label:"处理状态" validate:"omitempty,oneof=pending processing resolved ignored"`
	Keyword string `json:"keyword" label:"搜索关键词" validate:"omitempty,max=100"`
	PageReq
}

// MyFeedbackListReq 我的反馈列表请求体
type MyFeedbackListReq struct {
	PageReq
}
