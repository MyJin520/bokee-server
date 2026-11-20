package request

type CreateArticleRequest struct {
	Title   string `json:"title" label:"文章标题" validate:"required,max=200"`
	Content string `json:"content" label:"正文内容" validate:"required"`
	Summary string `json:"summary" label:"文章摘要" validate:"omitempty,max=300"`
	Cover   string `json:"cover" label:"封面图" validate:"omitempty,max=500"`
}

type UpdateArticleRequest struct {
	ID      uint    `json:"id" label:"文章ID" validate:"required"`                         // 文章 ID
	Title   *string `json:"title,omitempty" label:"文章标题" validate:"omitempty,max=200"`   // 文章标题
	Content *string `json:"content,omitempty"`                                           // 正文内容
	Summary *string `json:"summary,omitempty" label:"文章摘要" validate:"omitempty,max=300"` // 文章摘要
	Cover   *string `json:"cover,omitempty" label:"封面图" validate:"omitempty,max=500"`    // 封面图 URL
	IsTop   *bool   `json:"isTop,omitempty"`                                             // 是否置顶
}

type ArticleQueryListReq struct {
	Title    string `json:"title" label:"文章标题" validate:"omitempty,max=100"`
	UserName string `json:"userName" label:"用户名" validate:"omitempty,max=10"`
	Summary  string `json:"summary" label:"文章摘要" validate:"omitempty,max=100"`
	PageReq
}

type UserArticleListReq struct {
	UserID uint `json:"userId" label:"用户ID" validate:"required"` // 用户 ID
	PageReq
}
