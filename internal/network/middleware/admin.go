package middleware

import (
	"bokee/global"

	"github.com/gin-gonic/gin"
)

// IsSuperAdmin 判断当前登录用户是否拥有超级管理员角色（角色标识码取自 JWT 上下文）。
// 仅用于业务层的数据可见范围判断；路由级权限仍由 Casbin 中间件负责。
func IsSuperAdmin(c *gin.Context) bool {
	roleCodes, ok := GetRoleCodesFromContext(c)
	if !ok {
		return false
	}
	for _, code := range roleCodes {
		if code == global.SuperRoleCode {
			return true
		}
	}
	return false
}
