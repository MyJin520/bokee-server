package router

import (
	"bokee/internal/network/api"
	"bokee/internal/network/ws"
	"bokee/pkg/routex"

	"github.com/gin-gonic/gin"
)

var messageApi = api.MessageApi{}

func InitMessageRouter(privateGroup *gin.RouterGroup, engine *gin.Engine) {
	messagePrivate := routex.NewGroup("私信消息模块", privateGroup.Group("/message"))
	{
		messagePrivate.POST("/send", "发送私信", messageApi.Send)
		messagePrivate.POST("/list", "获取私信消息列表", messageApi.List)
		messagePrivate.POST("/read", "标记私信已读", messageApi.Read)
		messagePrivate.GET("/unread", "获取私信未读数", messageApi.Unread)
		messagePrivate.GET("/sync", "同步离线私信", messageApi.Sync)
		messagePrivate.POST("/ws_ticket", "获取 WebSocket 连接凭证", messageApi.WsTicket)
	}

	// WebSocket 握手不走 JWT/Casbin，使用一次性 ticket 鉴权
	engine.GET("/ws", ws.HandleUpgrade)
}
