package api

import (
	"errors"
	"time"

	"bokee/global"
	"bokee/internal/mods/request"
	"bokee/internal/mods/response"
	"bokee/internal/network/middleware"
	"bokee/internal/network/service"
	"bokee/pkg/randx"
	"bokee/pkg/redisx"
	"bokee/pkg/routex"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
		if _, ok := errors.AsType[*service.QuotaExceededError](err); ok {
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
	response.OkWithDetailed(&response.ReadResp{LastReadID: lastReadID}, "已读状态更新成功", c)
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
	afterID := uint(0)
	if v, ok := request.ParseQueryUint(c, "afterId"); ok {
		afterID = v
	}
	limit := 200
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

	ttlSeconds := 30
	if global.Config != nil {
		ttlSeconds = global.Config.IM.WsTicketTTL
	}
	ttl := time.Duration(ttlSeconds) * time.Second
	ticket := randx.RandomUUID()
	key := redisx.BuildKey("im", "ws", "ticket", ticket)
	if err := redisx.Set(c.Request.Context(), key, uid, ttl); err != nil {
		global.Log.Error("生成 WebSocket 连接凭证失败", zap.Uint("user_id", uid), zap.Error(err))
		response.FailWithMessage("连接凭证生成失败，请稍后重试", c)
		return
	}
	response.OkWithDetailed(&response.WsTicketResp{
		Ticket:    ticket,
		ExpiresIn: ttlSeconds,
	}, "连接凭证获取成功", c)
}
