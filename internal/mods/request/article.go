package request

type CreateArticleRequest struct {
	Title   string `json:"title" label:"文章标题" validate:"required"`
	Content string `json:"content" label:"正文内容" validate:"required"`
	Summary string `json:"summary"`
	Cover   string `json:"cover"`
}

type UpdateArticleRequest struct {
	ID      uint    `json:"id" label:"文章ID" validate:"required"` // 文章 ID
	Title   *string `json:"title,omitempty"`                     // 文章标题
	Content *string `json:"content,omitempty"`                   // 正文内容
	Summary *string `json:"summary,omitempty"`                   // 文章摘要
	Cover   *string `json:"cover,omitempty"`                     // 封面图 URL
	IsTop   *bool   `json:"isTop,omitempty"`                     // 是否置顶
}

type ArticleQueryListReq struct {
	Title    string `json:"title"`
	UserName string `json:"userName"`
	Summary  string `json:"summary"`
	PageReq
}

type UserArticleListReq struct {
	UserID uint `json:"userId"` // 用户 ID
	PageReq
}
