package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"bokee/global"
	"bokee/internal/mods/basic"
	"bokee/internal/mods/request"
	"bokee/internal/mods/response"
	"bokee/pkg/commons"
	"bokee/pkg/limitx"
	"bokee/pkg/redisx"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// exportMaxRows 单次导出上限，避免一次拉取过多数据占用内存
const exportMaxRows = 5000

var (
	// minuteLimiter 同一用户 1 分钟最多提交 1 条，防止连点
	minuteLimiter = &limitx.RateLimiter{Scope: "feedback:create:minute", Window: time.Minute, Max: 1}
	// dailyLimiter 同一用户 24 小时内最多提交 20 条，防止刷屏
	dailyLimiter = &limitx.RateLimiter{Scope: "feedback:create:daily", Window: 24 * time.Hour, Max: 20}
)

type FeedbackService struct{}

// Create 提交反馈：先做频率限制与内容去重，再落库
func (s *FeedbackService) Create(ctx context.Context, req request.CreateFeedbackRequest, userID uint) error {
	if err := checkSubmitFrequency(ctx, userID); err != nil {
		return err
	}
	if err := checkDuplicate(ctx, req, userID); err != nil {
		return err
	}

	imagesJSON := ""
	if len(req.Images) > 0 {
		data, err := json.Marshal(req.Images)
		if err != nil {
			global.Log.Error("序列化反馈截图失败", zap.Error(err), zap.Uint("userID", userID))
			return fmt.Errorf("反馈提交失败，请稍后重试")
		}
		imagesJSON = string(data)
	}

	newFeedback := basic.NewFeedback(userID, req.Type, req.Title, req.Content, imagesJSON, req.PageURL, req.Contact)
	if err := global.DB.Create(newFeedback).Error; err != nil {
		global.Log.Error("提交反馈失败", zap.Error(err), zap.Uint("userID", userID), zap.String("type", req.Type))
		return fmt.Errorf("反馈提交失败，请稍后重试")
	}
	global.Log.Info("反馈提交成功", zap.Uint("feedbackID", newFeedback.ID), zap.Uint("userID", userID), zap.String("type", req.Type))
	return nil
}

// List 反馈列表：超管可见全部；其他用户仅可见「非 Bug 反馈 + 自己提交的反馈」
func (s *FeedbackService) List(req request.FeedbackQueryListReq, viewerID uint, isAdmin bool) ([]response.FeedbackListItemResp, int64, error) {
	query := global.DB.Model(&basic.Feedback{})
	if !isAdmin {
		query = query.Where("(type != ? OR user_id = ?)", basic.FeedbackTypeBug, viewerID)
	}
	query = applyFeedbackFilter(query, req)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		global.Log.Error("统计反馈总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("查询反馈列表失败，请稍后重试")
	}

	var list []basic.Feedback
	if err := query.Order("id DESC").
		Offset(req.Offset()).
		Limit(req.PageSize).
		Find(&list).Error; err != nil {
		global.Log.Error("查询反馈列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("查询反馈列表失败，请稍后重试")
	}

	return buildFeedbackList(list, viewerID, isAdmin), total, nil
}

// ListByUser 查询当前用户提交的全部反馈（含 Bug，便于跟踪处理进度）
func (s *FeedbackService) ListByUser(pageReq request.PageReq, userID uint) ([]response.FeedbackListItemResp, int64, error) {
	query := global.DB.Model(&basic.Feedback{}).Where("user_id = ?", userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		global.Log.Error("统计我的反馈总数失败", zap.Error(err), zap.Uint("userID", userID))
		return nil, 0, fmt.Errorf("查询我的反馈列表失败，请稍后重试")
	}

	var list []basic.Feedback
	if err := query.Order("id DESC").
		Offset(pageReq.Offset()).
		Limit(pageReq.PageSize).
		Find(&list).Error; err != nil {
		global.Log.Error("查询我的反馈列表失败", zap.Error(err), zap.Uint("userID", userID))
		return nil, 0, fmt.Errorf("查询我的反馈列表失败，请稍后重试")
	}

	return buildFeedbackList(list, userID, true), total, nil
}

// GetInfo 反馈详情：数据可见范围与列表一致，防止非管理员遍历 id 读取 Bug 反馈
func (s *FeedbackService) GetInfo(id, viewerID uint, isAdmin bool) (response.FeedbackInfoResp, error) {
	query := global.DB.Model(&basic.Feedback{})
	if !isAdmin {
		query = query.Where("(type != ? OR user_id = ?)", basic.FeedbackTypeBug, viewerID)
	}

	var detail basic.Feedback
	if err := query.Where("id = ?", id).First(&detail).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Warn("反馈不存在或无权查看", zap.Uint("feedbackID", id), zap.Uint("viewerID", viewerID))
			return response.FeedbackInfoResp{}, fmt.Errorf("反馈不存在")
		}
		global.Log.Error("查询反馈详情失败", zap.Error(err), zap.Uint("feedbackID", id))
		return response.FeedbackInfoResp{}, fmt.Errorf("查询反馈失败，请稍后重试")
	}

	return *response.NewFeedbackInfoResp(detail, loadUser(detail.UserID), isAdmin || detail.UserID == viewerID), nil
}

// Update 处理反馈：修改处理状态、填写管理员回复（仅超级管理员）
func (s *FeedbackService) Update(req request.UpdateFeedbackRequest, adminID uint) error {
	var detail basic.Feedback
	if err := global.DB.Where("id = ?", req.ID).First(&detail).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Warn("处理反馈失败：反馈不存在", zap.Uint("feedbackID", req.ID))
			return fmt.Errorf("反馈不存在")
		}
		global.Log.Error("查询反馈失败", zap.Error(err), zap.Uint("feedbackID", req.ID))
		return fmt.Errorf("查询反馈失败，请稍后重试")
	}

	updates := commons.StructToUpdateMap(req)
	if len(updates) == 0 {
		return fmt.Errorf("没有需要更新的内容")
	}
	if req.Reply != nil {
		updates["replied_by"] = adminID
		updates["replied_at"] = time.Now()
	}

	// 未填写回复、仅流转状态时，清空历史回复时间没有意义，这里保持原样即可
	result := global.DB.Model(&detail).Updates(updates)
	if result.Error != nil {
		global.Log.Error("更新反馈失败", zap.Error(result.Error), zap.Uint("feedbackID", req.ID))
		return fmt.Errorf("反馈更新失败，请稍后重试")
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("反馈更新失败，可能没有变化")
	}
	global.Log.Info("更新反馈成功", zap.Uint("feedbackID", req.ID), zap.Uint("adminID", adminID))
	return nil
}

// Delete 删除反馈（软删除，仅超级管理员）
func (s *FeedbackService) Delete(id uint) error {
	var detail basic.Feedback
	if err := global.DB.Where("id = ?", id).First(&detail).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Warn("删除反馈失败：反馈不存在", zap.Uint("feedbackID", id))
			return fmt.Errorf("反馈不存在")
		}
		global.Log.Error("查询反馈失败", zap.Error(err), zap.Uint("feedbackID", id))
		return fmt.Errorf("查询反馈失败，请稍后重试")
	}

	result := global.DB.Delete(&detail)
	if result.Error != nil {
		global.Log.Error("删除反馈失败", zap.Error(result.Error), zap.Uint("feedbackID", id))
		return fmt.Errorf("反馈删除失败，请稍后重试")
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("反馈不存在")
	}
	global.Log.Info("删除反馈成功", zap.Uint("feedbackID", id))
	return nil
}

// Summary 按类型与状态统计反馈数量（仅超级管理员）
func (s *FeedbackService) Summary() (response.FeedbackSummaryResp, error) {
	type countRow struct {
		Type   string
		Status string
		Count  int64
	}

	var rows []countRow
	if err := global.DB.Model(&basic.Feedback{}).
		Select("type, status, COUNT(*) AS count").
		Group("type, status").
		Scan(&rows).Error; err != nil {
		global.Log.Error("统计反馈数量失败", zap.Error(err))
		return response.FeedbackSummaryResp{}, fmt.Errorf("查询反馈统计失败，请稍后重试")
	}

	typeCount := map[string]int64{}
	statusCount := map[string]int64{}
	var total int64
	for _, row := range rows {
		total += row.Count
		typeCount[row.Type] += row.Count
		statusCount[row.Status] += row.Count
	}

	// 固定按枚举顺序返回（含 0），前端可直接渲染标签，不用补零
	resp := response.FeedbackSummaryResp{
		Total:    total,
		Pending:  statusCount[basic.FeedbackStatusPending],
		ByType:   make([]response.FeedbackCountItem, 0, 4),
		ByStatus: make([]response.FeedbackCountItem, 0, 4),
	}
	for _, typ := range []string{
		basic.FeedbackTypeBug,
		basic.FeedbackTypeOptimize,
		basic.FeedbackTypeFeature,
		basic.FeedbackTypeOther,
	} {
		resp.ByType = append(resp.ByType, response.FeedbackCountItem{Key: typ, Count: typeCount[typ]})
	}
	for _, status := range []string{
		basic.FeedbackStatusPending,
		basic.FeedbackStatusProcessing,
		basic.FeedbackStatusResolved,
		basic.FeedbackStatusIgnored,
	} {
		resp.ByStatus = append(resp.ByStatus, response.FeedbackCountItem{Key: status, Count: statusCount[status]})
	}
	return resp, nil
}

// Export 导出反馈数据（仅超级管理员），按筛选条件取最近的数据，上限 exportMaxRows 条
func (s *FeedbackService) Export(req request.FeedbackQueryListReq) ([]response.FeedbackListItemResp, error) {
	query := applyFeedbackFilter(global.DB.Model(&basic.Feedback{}), req)

	var list []basic.Feedback
	if err := query.Order("id DESC").Limit(exportMaxRows).Find(&list).Error; err != nil {
		global.Log.Error("导出反馈查询失败", zap.Error(err))
		return nil, fmt.Errorf("导出反馈失败，请稍后重试")
	}
	return buildFeedbackList(list, 0, true), nil
}

// applyFeedbackFilter 追加类型、状态、关键词筛选条件
func applyFeedbackFilter(query *gorm.DB, req request.FeedbackQueryListReq) *gorm.DB {
	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.Keyword != "" {
		like := "%" + req.Keyword + "%"
		query = query.Where("(title LIKE ? OR content LIKE ?)", like, like)
	}
	return query
}

// buildFeedbackList 批量补齐提交人信息并转换为响应结构
func buildFeedbackList(list []basic.Feedback, viewerID uint, isAdmin bool) []response.FeedbackListItemResp {
	users := batchLoadUsers(list)
	resp := make([]response.FeedbackListItemResp, 0, len(list))
	for _, item := range list {
		canViewPrivate := isAdmin || item.UserID == viewerID
		resp = append(resp, *response.NewFeedbackListItemResp(item, users[item.UserID], canViewPrivate))
	}
	return resp
}

// batchLoadUsers 批量查询反馈提交人信息，返回 userID -> User 的映射
func batchLoadUsers(list []basic.Feedback) map[uint]basic.User {
	result := make(map[uint]basic.User, len(list))
	if len(list) == 0 {
		return result
	}

	seen := make(map[uint]struct{}, len(list))
	ids := make([]uint, 0, len(list))
	for _, item := range list {
		if _, ok := seen[item.UserID]; ok {
			continue
		}
		seen[item.UserID] = struct{}{}
		ids = append(ids, item.UserID)
	}

	var users []basic.User
	if err := global.DB.Select("id", "name", "avatar").Where("id IN ?", ids).Find(&users).Error; err != nil {
		global.Log.Warn("批量查询反馈提交人失败", zap.Error(err))
		return result
	}
	for _, user := range users {
		result[user.ID] = user
	}
	return result
}

// loadUser 查询单个用户基础信息，失败时返回零值，不阻断反馈查询主流程
func loadUser(userID uint) basic.User {
	var user basic.User
	if err := global.DB.Select("id", "name", "avatar").Where("id = ?", userID).First(&user).Error; err != nil {
		global.Log.Warn("查询反馈提交人失败", zap.Error(err), zap.Uint("userID", userID))
	}
	return user
}

// checkSubmitFrequency 提交频率限制，Redis 异常时由 RateLimiter 内部放行
func checkSubmitFrequency(ctx context.Context, userID uint) error {
	id := strconv.Itoa(int(userID))
	if !minuteLimiter.Allow(ctx, id) {
		return fmt.Errorf("提交过于频繁，请 1 分钟后再试")
	}
	if !dailyLimiter.Allow(ctx, id) {
		return fmt.Errorf("24 小时内反馈提交已达上限（20 条），请稍后再试")
	}
	return nil
}

// checkDuplicate 基于「类型 + 标题 + 内容」的内容指纹做 1 分钟去重，防止网络重试与重复点击
func checkDuplicate(ctx context.Context, req request.CreateFeedbackRequest, userID uint) error {
	raw := req.Type + "|" + req.Title + "|" + req.Content
	sum := sha256.Sum256([]byte(raw))
	key := redisx.BuildKey("feedback", "fingerprint", strconv.Itoa(int(userID)), fmt.Sprintf("%x", sum))

	wrote, err := redisx.SetNX(ctx, key, "1", time.Minute)
	if err != nil {
		global.Log.Error("反馈内容去重校验失败，已放行", zap.Error(err), zap.Uint("userID", userID))
		return nil
	}
	if !wrote {
		return fmt.Errorf("请勿重复提交相同反馈")
	}
	return nil
}
