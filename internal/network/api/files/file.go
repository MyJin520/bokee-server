package files

import (
	"bokee/global"
	"bokee/internal/mods/response"
	"bokee/internal/network/service/files"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FileApi struct{}

var fileService = &files.FileService{}

func (f *FileApi) Uploads(ctx *gin.Context) {
	form, err := ctx.MultipartForm()
	if err != nil {
		global.Log.Error("表单解析失败", zap.Error(err))
		response.FailWithRequest("表单解析失败", ctx)
		return
	}
	fileHeaders := form.File["files"]
	if len(fileHeaders) == 0 {
		response.FailWithRequest("未找到上传文件", ctx)
		return
	}

	uploaded := make([]response.FileResp, 0, len(fileHeaders))
	for _, fileHeader := range fileHeaders {
		fileRecord, err := fileService.Uploads(fileHeader)
		if err != nil {
			global.Log.Error("文件上传失败", zap.Error(err))
			response.FailWithMessage("文件上传失败", ctx)
			return
		}
		uploaded = append(uploaded, *fileRecord)
	}

	response.OkWithDetailed(uploaded, "文件上传成功", ctx)
}