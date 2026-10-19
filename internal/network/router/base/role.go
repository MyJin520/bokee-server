package base

import (
	"bokee/internal/network/api/base"
	"bokee/pkg/routex"
	"github.com/gin-gonic/gin"
)

var roleApi = base.RoleApi{}

func InitRoleRouter(privateGroup *gin.RouterGroup) {
	const module = "角色模块"
	rolePrivate := routex.NewGroup(module, privateGroup.Group("/role"))
	{
		rolePrivate.POST("/create", "创建角色", roleApi.Create)                      // 创建角色
		rolePrivate.PUT("/update", "更新角色", roleApi.Update)                       // 更新角色
		rolePrivate.DELETE("/delete", "删除角色", roleApi.Delete)                    // 删除角色
		rolePrivate.GET("/get_info", "获取角色详情", roleApi.GetInfo)                  // 获取角色详情
		rolePrivate.POST("/list", "分页获取角色列表", roleApi.List)                      // 分页获取角色列表
		rolePrivate.POST("/auth", "角色授权（批量）", roleApi.Auth)                      // 角色授权（批量）
		rolePrivate.POST("/get_all_pri_rule", "获取所有私有路由", roleApi.GetAllPriRule) // 获取所有私有路由
	}
}
