package middleware

import (
	"bokee/global"
	"bokee/internal/mods/response"
	"bokee/pkg/jwtx"
	"bokee/pkg/redisx"
	"go.uber.org/zap"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware JWT 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := jwtx.ExtractBearerToken(c.GetHeader("Authorization"))
		if err != nil {
			global.Log.Warn("Authorization 头无效", zap.String("header", c.GetHeader("Authorization")), zap.Error(err))
			response.Fail(http.StatusUnauthorized, err.Error(), c)
			c.Abort()
			return
		}

		// 解析并验证 token
		claims, err := jwtx.ParseToken(tokenString)
		if err != nil {
			global.Log.Error("无效的 JWT 令牌", zap.Error(err))
			response.Fail(http.StatusUnauthorized, "无效或过期的认证令牌", c)
			c.Abort()
			return
		}

		// 检查 Redis 黑名单（token 是否已登出）
		blocked, err := redisx.IsTokenBlacklisted(c.Request.Context(), tokenString)
		if err != nil {
			global.Log.Error("Redis 黑名单查询失败", zap.String("jti", claims.ID), zap.Error(err))
			// 查询失败放行，不阻断请求
		} else if blocked {
			global.Log.Warn("Token 已被登出", zap.String("jti", claims.ID), zap.Uint("userID", claims.UserID))
			response.Fail(http.StatusUnauthorized, "令牌已失效，请重新登录", c)
			c.Abort()
			return
		}

		// 将用户信息存入 Gin 上下文
		c.Set("userID", claims.UserID)
		c.Set("userName", claims.UserName)
		c.Set("roleCodes", claims.RoleCodes)

		c.Next()
	}
}

// GetUserIDFromContext 从上下文获取当前登录用户的 ID
func GetUserIDFromContext(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return 0, false
	}
	id, ok := userID.(uint)
	return id, ok
}

// GetUsernameFromContext 从上下文获取当前登录用户名
func GetUsernameFromContext(c *gin.Context) (string, bool) {
	username, exists := c.Get("userName")
	if !exists {
		return "", false
	}
	name, ok := username.(string)
	return name, ok
}

// GetRoleCodesFromContext 从上下文获取当前登录用户的角色标识码列表
func GetRoleCodesFromContext(c *gin.Context) ([]uint, bool) {
	roleCodes, exists := c.Get("roleCodes")
	if !exists {
		return nil, false
	}
	codes, ok := roleCodes.([]uint)
	return codes, ok
}
