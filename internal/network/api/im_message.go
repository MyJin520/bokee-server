package api

import (
	"bokee/internal/mods/request"
	"bokee/internal/mods/response"
	"bokee/internal/network/middleware"
	"bokee/internal/network/service"
	"bokee/pkg/routex"

	"github.com/gin-gonic/gin"
)

type MessageApi struct{}

var messageService = service.NewMessageService()

func (a *MessageApi) Send(c *gin.Context) {
	var req request.MessageSendReq
	if !routex.BindCheckStruct(c, &req) {
		return
	}
	uid, _ := middleware.GetUserIDFromContext(c)
	resp, err := messageService.Send(c.Request.Context(), req, uid)
	if err != nil {
		if service.IsQuotaExceeded(err) {
			response.Fail(response.CodeQuotaExceeded, err.Error(), c)
			return
		}
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(resp, "消息发送成功", c)
}

func (a *MessageApi) List(c *gin.Context) {
	var req request.MessageListReq
	if !routex.BindCheckStruct(c, &req) {
		return
	}
	uid, _ := middleware.GetUserIDFromContext(c)
	resp, err := messageService.List(c.Request.Context(), req, uid)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(resp, "消息列表获取成功", c)
}

func (a *MessageApi) Read(c *gin.Context) {
	var req request.MessageReadReq
	if !routex.BindCheckStruct(c, &req) {
		return
	}
	uid, _ := middleware.GetUserIDFromContext(c)
	lastReadID, err := messageService.MarkRead(c.Request.Context(), req, uid)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(response.NewReadResp(lastReadID), "已读状态更新成功", c)
}

func (a *MessageApi) Unread(c *gin.Context) {
	uid, _ := middleware.GetUserIDFromContext(c)
	resp, err := messageService.Unread(c.Request.Context(), uid)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(resp, "未读消息获取成功", c)
}

func (a *MessageApi) Sync(c *gin.Context) {
	var afterID uint
	if v, ok := request.ParseQueryUint(c, "afterId"); ok {
		afterID = v
	}
	var limit int
	if v, ok := request.ParseQueryUint(c, "limit"); ok {
		limit = int(v)
	}
	uid, _ := middleware.GetUserIDFromContext(c)
	resp, err := messageService.Sync(c.Request.Context(), afterID, limit, uid)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(resp, "离线消息同步成功", c)
}

func (a *MessageApi) WsTicket(c *gin.Context) {
	uid, _ := middleware.GetUserIDFromContext(c)
	resp, err := messageService.WsTicket(c.Request.Context(), uid)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(resp, "连接凭证获取成功", c)
}

func (a *MessageApi) Config(c *gin.Context) {
	response.OkWithDetailed(messageService.Config(), "私信配置获取成功", c)
}
