package files

import (
	"bokee/internal/network/api/files"
	"bokee/pkg/routex"
	"github.com/gin-gonic/gin"
)

var fileApi = files.FileApi{}

func InitFileRouter(privateGroup *gin.RouterGroup) {
	const module = "文件模块"
	filePrivate := routex.NewGroup(module, privateGroup.Group("/file"))
	{
		filePrivate.POST("/uploads", "文件上传", fileApi.Uploads) // 文件上传（需登录鉴权）
	}
}
