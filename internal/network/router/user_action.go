package router

import (
	"bokee/internal/network/api"
	"bokee/pkg/routex"

	"github.com/gin-gonic/gin"
)

var userActionApi = api.UserActionApi{}

func InitUserActionRouter(privateGroup *gin.RouterGroup) {
	userActionPrivate := routex.NewGroup("用户操作模块", privateGroup.Group("/user_action"))
	{
		userActionPrivate.POST("/create", "创建用户操作", userActionApi.Create)
		userActionPrivate.DELETE("/delete", "删除用户操作", userActionApi.Delete)
		userActionPrivate.POST("/list", "获取用户操作列表", userActionApi.List)
	}
}
