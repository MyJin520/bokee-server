package base

import (
	"bokee/internal/network/api/base"
	"bokee/pkg/routex"

	"github.com/gin-gonic/gin"
)

var userApi = base.UserApi{}

func InitUserRouter(publicGroup *gin.RouterGroup, privateGroup *gin.RouterGroup) {
	const module = "用户模块"
	userPublic := routex.NewGroup(module, publicGroup.Group("/user"))
	{
		userPublic.POST("/create", "创建用户", userApi.Create) // 创建用户
		userPublic.POST("/login", "用户登录", userApi.Login)   // 用户登录
	}

	userPrivate := routex.NewGroup(module, privateGroup.Group("/user"))
	{
		userPrivate.PUT("/update", "更新用户信息", userApi.Update)                  // 更新用户信息
		userPrivate.GET("/logout", "用户登出", userApi.Logout)                    // 用户登出
		userPrivate.GET("/get_info", "获取用户信息", userApi.GetInfo)               // 获取用户信息
		userPrivate.POST("/forget_password", "忘记密码", userApi.ForgetPassword)  // 忘记密码
		userPrivate.POST("/operate_roles", "用户角色绑定/解绑", userApi.OperateRoles) // 用户角色绑定/解绑
		userPrivate.POST("/list", "用户列表", userApi.List)                       // 用户列表【仅超管】
		userPrivate.POST("/find_users", "查询用户", userApi.FindUsers)            // 查询用户【用户间查询】
	}
}
