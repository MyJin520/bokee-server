package middleware

import (
	"bokee/internal/mods/response"
	"bokee/pkg/casbinx"
	"net/http"
	"strconv"

	"bokee/global"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func CasbinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 JWT 中间件注入的上下文中获取角色标识码列表
		roleCodes, ok := GetRoleCodesFromContext(c)
		if !ok || len(roleCodes) == 0 {
			global.Log.Warn("Casbin 中间件：上下文中未找到角色标识码，拒绝访问")
			response.Fail(http.StatusForbidden, "未认证或无任何角色", c)
			c.Abort()
			return
		}

		// 构造 Casbin 所需的资源路径与动作
		obj := c.Request.URL.Path // 例如 /pri/user/update
		act := c.Request.Method   // 例如 POST

		// 逐个角色标识码执行权限校验，任一角色拥有权限即放行
		for _, code := range roleCodes {
			sub := strconv.Itoa(int(code))
			allowed, err := casbinx.Enforce(sub, obj, act)
			if err != nil {
				global.Log.Error("Casbin 权限校验异常",
					zap.String("sub", sub),
					zap.String("obj", obj),
					zap.String("act", act),
					zap.Error(err),
				)
				response.Fail(http.StatusInternalServerError, "权限服务异常", c)
				c.Abort()
				return
			}
			if allowed {
				c.Next()
				return
			}
		}

		global.Log.Warn("Casbin 拒绝访问",
			zap.Uints("roleCodes", roleCodes),
			zap.String("obj", obj),
			zap.String("act", act),
		)
		response.Fail(http.StatusForbidden, "无此操作权限", c)
		c.Abort()
	}
}
