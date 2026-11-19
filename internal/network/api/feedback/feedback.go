package feedback

import (
	"encoding/csv"
	"net/http"
	"net/url"
	"strconv"

	"bokee/global"
	"bokee/internal/mods/basic"
	"bokee/internal/mods/request"
	"bokee/internal/mods/response"
	"bokee/internal/network/middleware"
	"bokee/internal/network/service/feedback"
	"bokee/pkg/routex"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FeedbackApi struct{}

var feedbackService = &feedback.FeedbackService{}

// 导出时使用的中文标签
var (
	feedbackTypeLabels = map[string]string{
		basic.FeedbackTypeBug:      "缺陷",
		basic.FeedbackTypeOptimize: "优化建议",
		basic.FeedbackTypeFeature:  "新功能建议",
		basic.FeedbackTypeOther:    "其他",
	}
	feedbackStatusLabels = map[string]string{
		basic.FeedbackStatusPending:    "待处理",
		basic.FeedbackStatusProcessing: "处理中",
		basic.FeedbackStatusResolved:   "已解决",
		basic.FeedbackStatusIgnored:    "不处理",
	}
)

// requireSuperAdmin 业务层的管理员兜底校验（路由级权限由 Casbin 中间件负责）
func requireSuperAdmin(c *gin.Context) bool {
	if middleware.IsSuperAdmin(c) {
		return true
	}
	global.Log.Warn("非超级管理员访问反馈管理接口", zap.String("path", c.Request.URL.Path))
	response.Fail(http.StatusForbidden, "无此操作权限", c)
	return false
}

// Create 提交反馈
func (f *FeedbackApi) Create(c *gin.Context) {
	var req request.CreateFeedbackRequest
	if !routex.BindCheckStruct(c, &req) {
		return
	}
	userID, _ := middleware.GetUserIDFromContext(c)
	if err := feedbackService.Create(c.Request.Context(), req, userID); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("反馈提交成功", c)
}

// List 反馈列表（按当前用户身份过滤可见范围）
func (f *FeedbackApi) List(c *gin.Context) {
	var req request.FeedbackQueryListReq
	if !routex.BindCheckStruct(c, &req) {
		return
	}
	req.Normalize()

	userID, _ := middleware.GetUserIDFromContext(c)
	list, total, err := feedbackService.List(req, userID, middleware.IsSuperAdmin(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithPage(list, total, req.Page, req.PageSize, "反馈列表获取成功", c)
}

// GetInfo 反馈详情
func (f *FeedbackApi) GetInfo(c *gin.Context) {
	feedbackID, ok := routex.QueryUint(c, "id")
	if !ok {
		return
	}

	userID, _ := middleware.GetUserIDFromContext(c)
	detail, err := feedbackService.GetInfo(feedbackID, userID, middleware.IsSuperAdmin(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(detail, "反馈详情获取成功", c)
}

// MyList 我提交的反馈列表（含 Bug，便于跟踪处理进度）
func (f *FeedbackApi) MyList(c *gin.Context) {
	var req request.MyFeedbackListReq
	if !routex.BindCheckStruct(c, &req) {
		return
	}
	req.Normalize()

	userID, _ := middleware.GetUserIDFromContext(c)
	list, total, err := feedbackService.ListByUser(req.PageReq, userID)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithPage(list, total, req.Page, req.PageSize, "我的反馈列表获取成功", c)
}

// Update 处理反馈：修改状态、填写回复（仅超级管理员）
func (f *FeedbackApi) Update(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	var req request.UpdateFeedbackRequest
	if !routex.BindCheckStruct(c, &req) {
		return
	}

	userID, _ := middleware.GetUserIDFromContext(c)
	if err := feedbackService.Update(req, userID); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("反馈处理成功", c)
}

// Delete 删除反馈（仅超级管理员）
func (f *FeedbackApi) Delete(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	feedbackID, ok := routex.QueryUint(c, "id")
	if !ok {
		return
	}
	if err := feedbackService.Delete(feedbackID); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("反馈删除成功", c)
}

// Summary 反馈数量统计（仅超级管理员）
func (f *FeedbackApi) Summary(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	summary, err := feedbackService.Summary()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(summary, "反馈统计获取成功", c)
}

// Export 导出反馈 CSV（仅超级管理员）
func (f *FeedbackApi) Export(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	var req request.FeedbackQueryListReq
	req.Type = c.Query("type")
	req.Status = c.Query("status")
	req.Keyword = c.Query("keyword")
	if !isValidFeedbackType(req.Type) || !isValidFeedbackStatus(req.Status) {
		response.FailWithRequest("导出筛选参数异常", c)
		return
	}

	list, err := feedbackService.Export(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+url.QueryEscape("用户反馈.csv"))
	// 写入 UTF-8 BOM，避免 Excel 打开中文乱码
	_, _ = c.Writer.WriteString("\ufeff")

	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"ID", "类型", "标题", "内容", "页面地址", "联系方式", "状态", "回复", "提交人", "提交时间"})
	for _, item := range list {
		_ = writer.Write([]string{
			strconv.FormatUint(uint64(item.ID), 10),
			feedbackTypeLabels[item.Type],
			item.Title,
			item.Content,
			item.PageURL,
			item.Contact,
			feedbackStatusLabels[item.Status],
			item.Reply,
			item.User.Name,
			item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		global.Log.Error("导出反馈 CSV 失败", zap.Error(err))
	}
}

func isValidFeedbackType(value string) bool {
	switch value {
	case "", basic.FeedbackTypeBug, basic.FeedbackTypeOptimize, basic.FeedbackTypeFeature, basic.FeedbackTypeOther:
		return true
	}
	return false
}

func isValidFeedbackStatus(value string) bool {
	switch value {
	case "", basic.FeedbackStatusPending, basic.FeedbackStatusProcessing, basic.FeedbackStatusResolved, basic.FeedbackStatusIgnored:
		return true
	}
	return false
}
