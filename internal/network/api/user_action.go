package api

import (
	"bokee/internal/mods/request"
	"bokee/internal/mods/response"
	"bokee/internal/network/middleware"
	"bokee/internal/network/service"
	"bokee/pkg/routex"

	"github.com/gin-gonic/gin"
)

type UserActionApi struct{}

var userActionService = &service.UserActionService{}

func (u *UserActionApi) Create(c *gin.Context) {
	var req request.ActionCreateReq
	if err := routex.BindCheckStruct(c, &req); err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}
	userId, _ := middleware.GetUserIDFromContext(c)
	req.UserID = userId
	if err := userActionService.Create(c.Request.Context(), req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("用户操作成功", c)
}

func (u *UserActionApi) Delete(c *gin.Context) {
	actionId, err := routex.QueryUint(c, "id")
	if err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}
	userId, _ := middleware.GetUserIDFromContext(c)
	if err := userActionService.Delete(c.Request.Context(), actionId, userId); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("用户操作删除成功", c)
}

func (u *UserActionApi) List(c *gin.Context) {
	var req request.ActionListReq
	if err := routex.BindCheckStruct(c, &req); err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}
	userId, _ := middleware.GetUserIDFromContext(c)
	req.Normalize()
	list, total, err := userActionService.List(req, userId)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithPage(list, total, req.Page, req.PageSize, "用户操作列表获取成功", c)
}
