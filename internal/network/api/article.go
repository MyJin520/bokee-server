package api

import (
	"bokee/internal/mods/request"
	"bokee/internal/mods/response"
	"bokee/internal/network/middleware"
	"bokee/internal/network/service"
	"bokee/pkg/routex"

	"github.com/gin-gonic/gin"
)

type ArticleApi struct{}

var articleService = &service.ArticleService{}

func (a *ArticleApi) Create(c *gin.Context) {
	var req request.CreateArticleRequest
	if err := routex.BindCheckStruct(c, &req); err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}
	userId, _ := middleware.GetUserIDFromContext(c)
	err := articleService.Create(req, userId)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("文章发表成功", c)
}

func (a *ArticleApi) Update(c *gin.Context) {
	var req request.UpdateArticleRequest
	if err := routex.BindCheckStruct(c, &req); err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}
	userId, _ := middleware.GetUserIDFromContext(c)
	err := articleService.Update(c.Request.Context(), req, userId)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("文章更新成功", c)
}

func (a *ArticleApi) Delete(c *gin.Context) {
	articleID, err := routex.QueryUint(c, "id")
	if err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}
	userId, _ := middleware.GetUserIDFromContext(c)
	if err := articleService.Delete(c.Request.Context(), articleID, userId); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("文章删除成功", c)
}

func (a *ArticleApi) GetInfo(c *gin.Context) {
	articleID, err := routex.QueryUint(c, "id")
	if err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}
	article, err := articleService.GetInfo(c.Request.Context(), articleID)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(article, "文章详情获取成功", c)
}

func (a *ArticleApi) ListByUser(c *gin.Context) {
	var req request.UserArticleListReq
	if err := routex.BindCheckStruct(c, &req); err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}
	req.Normalize()
	articleList, total, err := articleService.ListByUser(req.UserID, req.PageReq)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithPage(articleList, total, req.Page, req.PageSize, "用户文章列表获取成功", c)
}

func (a *ArticleApi) List(c *gin.Context) {
	var req request.ArticleQueryListReq
	if err := routex.BindCheckStruct(c, &req); err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}
	articleList, total, err := articleService.List(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithPage(articleList, total, req.Page, req.PageSize, "文章列表获取成功", c)
}
