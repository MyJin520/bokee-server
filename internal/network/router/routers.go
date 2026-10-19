package router

import (
	"bokee/internal/network/middleware"
	"bokee/internal/network/router/articles"
	"bokee/internal/network/router/base"
	"bokee/internal/network/router/files"

	"github.com/gin-gonic/gin"
)

// HealthPath 健康检查探针路径
const HealthPath = "/health"

func Routers() *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: []string{HealthPath}}))

	// CORS 跨域中间件（全局）
	engine.Use(middleware.CORSMiddleware())

	engine.GET(HealthPath, func(c *gin.Context) { c.Status(204) })

	// 公开路由组
	publicGroup := engine.Group("/pub")

	// 私有路由组
	privateGroup := engine.Group("/pri")
	privateGroup.Use(middleware.AuthMiddleware())
	privateGroup.Use(middleware.CasbinMiddleware())

	base.InitUserRouter(publicGroup, privateGroup)
	base.InitRoleRouter(privateGroup)
	base.InitUserActionRouter(privateGroup)
	articles.InitArticleRouter(publicGroup, privateGroup)
	files.InitFileRouter(privateGroup)

	return engine
}
