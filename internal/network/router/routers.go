package router

import (
	"bokee/internal/network/middleware"

	"github.com/gin-gonic/gin"
)

// HealthPath 健康检查探针路径
const HealthPath = "/health"

func Routers() *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: []string{HealthPath, "/ws"}}))

	// CORS 跨域中间件（全局）
	engine.Use(middleware.CORSMiddleware())

	engine.GET(HealthPath, func(c *gin.Context) { c.Status(204) })

	// 公开路由组
	publicGroup := engine.Group("/pub")

	// 私有路由组
	privateGroup := engine.Group("/pri")
	privateGroup.Use(middleware.AuthMiddleware())
	privateGroup.Use(middleware.CasbinMiddleware())

	// 业务路由组
	InitUserRouter(publicGroup, privateGroup)
	InitRoleRouter(privateGroup)
	InitUserActionRouter(privateGroup)
	InitArticleRouter(publicGroup, privateGroup)
	InitFeedbackRouter(privateGroup)
	InitFileRouter(privateGroup)
	InitConversationRouter(privateGroup)
	InitMessageRouter(privateGroup, engine)

	return engine
}
