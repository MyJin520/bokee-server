package response

import (
	"bokee/internal/mods/basic"
	"time"
)

// AuthorInfo 文章作者公开信息（脱敏：仅展示必要字段）
type AuthorInfo struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// ArticleInfoResp 文章详情响应结构体
type ArticleInfoResp struct {
	ID            uint       `json:"id"`
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	Summary       string     `json:"summary"`
	Cover         string     `json:"cover"`
	ViewCount     int        `json:"viewCount"`
	FavoriteCount int        `json:"favoriteCount"`
	LikeCount     int        `json:"likeCount"`
	IsTop         bool       `json:"isTop"`
	UserID        uint       `json:"userId"`
	Author        AuthorInfo `json:"author"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// ArticleListItemResp 文章列表项响应结构体
type ArticleListItemResp struct {
	ID            uint      `json:"id"`
	Title         string    `json:"title"`
	Summary       string    `json:"summary"`
	Cover         string    `json:"cover"`
	ViewCount     int       `json:"viewCount"`
	FavoriteCount int       `json:"favoriteCount"`
	LikeCount     int       `json:"likeCount"`
	IsTop         bool      `json:"isTop"`
	UserID        uint      `json:"userId"`
	CreatedAt     time.Time `json:"createdAt"`
}

// NewArticleInfoResp 将文章模型转换为详情响应结构体
func NewArticleInfoResp(article basic.Article, user basic.User) *ArticleInfoResp {
	return &ArticleInfoResp{
		ID:        article.ID,
		Title:     article.Title,
		Content:   article.Content,
		Summary:   article.Summary,
		Cover:     article.Cover,
		ViewCount: article.ViewCount,
		IsTop:     article.IsTop,
		UserID:    article.UserID,
		Author: AuthorInfo{
			ID:     user.ID,
			Name:   user.Name,
			Avatar: user.Avatar,
		},
		CreatedAt: article.CreatedAt,
		UpdatedAt: article.UpdatedAt,
	}
}

// NewArticleListItemResp 将文章模型转换为列表响应结构体（不含正文内容）
func NewArticleListItemResp(article basic.Article) *ArticleListItemResp {
	return &ArticleListItemResp{
		ID:        article.ID,
		Title:     article.Title,
		Summary:   article.Summary,
		Cover:     article.Cover,
		ViewCount: article.ViewCount,
		IsTop:     article.IsTop,
		UserID:    article.UserID,
		CreatedAt: article.CreatedAt,
	}
}
