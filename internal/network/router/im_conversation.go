package router

import (
	"bokee/internal/network/api"
	"bokee/pkg/routex"

	"github.com/gin-gonic/gin"
)

var conversationApi = api.ConversationApi{}

func InitConversationRouter(privateGroup *gin.RouterGroup) {
	conversationPrivate := routex.NewGroup("私信会话模块", privateGroup.Group("/conversation"))
	{
		conversationPrivate.POST("/create", "创建或打开私信会话", conversationApi.Create)
		conversationPrivate.POST("/list", "获取私信会话列表", conversationApi.List)
		conversationPrivate.GET("/get_info", "获取私信会话详情", conversationApi.GetInfo)
	}
}
