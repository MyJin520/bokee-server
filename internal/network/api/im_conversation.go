package api

import (
	"bokee/internal/mods/request"
	"bokee/internal/mods/response"
	"bokee/internal/network/middleware"
	"bokee/internal/network/service"
	"bokee/pkg/routex"

	"github.com/gin-gonic/gin"
)

type ConversationApi struct{}

var conversationService = service.NewConversationService()

func (a *ConversationApi) Create(c *gin.Context) {
	var req request.ConversationCreateReq
	if err := routex.BindCheckStruct(c, &req); err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}
	uid, _ := middleware.GetUserIDFromContext(c)
	resp, err := conversationService.Create(c.Request.Context(), req, uid)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(resp, "会话创建成功", c)
}

func (a *ConversationApi) List(c *gin.Context) {
	var req request.ConversationListReq
	if err := routex.BindCheckStruct(c, &req); err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}
	uid, _ := middleware.GetUserIDFromContext(c)
	req.Normalize()
	list, total, err := conversationService.List(c.Request.Context(), req, uid)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithPage(list, total, req.Page, req.PageSize, "会话列表获取成功", c)
}

func (a *ConversationApi) GetInfo(c *gin.Context) {
	convID, err := routex.QueryUint(c, "id")
	if err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}
	uid, _ := middleware.GetUserIDFromContext(c)
	resp, err := conversationService.GetInfo(c.Request.Context(), convID, uid)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(resp, "会话信息获取成功", c)
}
