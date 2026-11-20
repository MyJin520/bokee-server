package router

import (
	"bokee/internal/network/api"
	"bokee/pkg/routex"

	"github.com/gin-gonic/gin"
)

var feedbackApi = api.FeedbackApi{}

func InitFeedbackRouter(privateGroup *gin.RouterGroup) {
	const module = "反馈模块"

	feedbackPrivate := routex.NewGroup(module, privateGroup.Group("/feedback"))
	{
		feedbackPrivate.POST("/create", "提交反馈", feedbackApi.Create)        // 提交反馈
		feedbackPrivate.POST("/list", "反馈列表", feedbackApi.List)            // 反馈列表
		feedbackPrivate.GET("/get_info", "反馈详情", feedbackApi.GetInfo)      // 反馈详情
		feedbackPrivate.POST("/my_list", "我的反馈列表", feedbackApi.MyList)     // 我的反馈列表
		feedbackPrivate.PUT("/update", "处理反馈【仅超管】", feedbackApi.Update)    // 处理反馈（改状态 / 回复）
		feedbackPrivate.DELETE("/delete", "删除反馈【仅超管】", feedbackApi.Delete) // 删除反馈
		feedbackPrivate.GET("/summary", "反馈统计【仅超管】", feedbackApi.Summary)  // 反馈统计
		feedbackPrivate.GET("/export", "导出反馈【仅超管】", feedbackApi.Export)    // 导出反馈 CSV
	}
}
