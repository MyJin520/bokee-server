package articles

import (
	"context"
	"fmt"

	"bokee/global"
	"bokee/internal/mods/basic"
	"bokee/internal/mods/request"
	"bokee/internal/mods/response"
	"bokee/pkg/commons"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ArticleService struct{}

func (a *ArticleService) Create(req request.CreateArticleRequest, userId uint) error {
	newArticle := basic.NewArticle(userId, req.Title, req.Content, req.Summary, req.Cover)
	if err := global.DB.Create(newArticle).Error; err != nil {
		global.Log.Error("发表文章失败", zap.Error(err), zap.Uint("userID", userId))
		return fmt.Errorf("文章发表失败，请稍后重试")
	}
	return nil
}

func (a *ArticleService) Update(ctx context.Context, req request.UpdateArticleRequest, userId uint) error {
	var article basic.Article
	err := global.DB.Where("id = ?", req.ID).First(&article).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Warn("更新文章失败：文章不存在", zap.Uint("articleID", req.ID))
			return fmt.Errorf("文章不存在")
		}
		global.Log.Error("查询文章失败", zap.Error(err), zap.Uint("articleID", req.ID))
		return fmt.Errorf("查询文章失败，请稍后重试")
	}

	if err := commons.CheckOwnership(article, userId); err != nil {
		global.Log.Warn("更新文章失败：非文章作者", zap.Uint("articleID", req.ID), zap.Uint("userID", userId))
		return err
	}

	updates := commons.StructToUpdateMap(req)
	delete(updates, "id")

	if len(updates) == 0 {
		return nil
	}

	result := global.DB.Model(&article).Updates(updates)
	if result.Error != nil {
		global.Log.Error("更新文章失败", zap.Error(result.Error), zap.Uint("articleID", req.ID))
		return fmt.Errorf("文章更新失败，请稍后重试")
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("文章更新失败，可能没有变化")
	}
	return nil
}

func (a *ArticleService) Delete(ctx context.Context, id uint, userId uint) error {
	var article basic.Article
	err := global.DB.Where("id = ?", id).First(&article).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Warn("删除文章失败：文章不存在", zap.Uint("articleID", id))
			return fmt.Errorf("文章不存在")
		}
		global.Log.Error("查询文章失败", zap.Error(err), zap.Uint("articleID", id))
		return fmt.Errorf("查询文章失败，请稍后重试")
	}

	if err := commons.CheckOwnership(article, userId); err != nil {
		global.Log.Warn("删除文章失败：非文章作者", zap.Uint("articleID", id), zap.Uint("userID", userId))
		return err
	}

	result := global.DB.Delete(&article)
	if result.Error != nil {
		global.Log.Error("删除文章失败", zap.Error(result.Error), zap.Uint("articleID", id))
		return fmt.Errorf("文章删除失败，请稍后重试")
	}
	if result.RowsAffected == 0 {
		global.Log.Warn("删除文章失败：文章不存在", zap.Uint("articleID", id))
		return fmt.Errorf("文章不存在")
	}
	global.Log.Info("删除文章成功", zap.Uint("articleID", id))
	return nil
}

func (a *ArticleService) GetInfo(ctx context.Context, id uint) (response.ArticleInfoResp, error) {
	// 每次访问阅读量 +1（原子自增）
	if err := global.DB.Model(&basic.Article{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error; err != nil {
		global.Log.Warn("文章阅读量自增失败", zap.Error(err), zap.Uint("articleID", id))
	}

	var article basic.Article
	err := global.DB.Where("id = ?", id).First(&article).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.ArticleInfoResp{}, fmt.Errorf("文章不存在")
		}
		global.Log.Error("查询文章详情失败", zap.Error(err), zap.Uint("articleID", id))
		return response.ArticleInfoResp{}, fmt.Errorf("查询文章失败，请稍后重试")
	}

	// 关联查询作者信息
	var user basic.User
	if err := global.DB.Select("id", "name", "avatar").Where("id = ?", article.UserID).First(&user).Error; err != nil {
		global.Log.Warn("查询文章作者信息失败", zap.Error(err), zap.Uint("userID", article.UserID))
	}

	// 查询该文章的收藏数
	var favoriteCount int64
	global.DB.Model(&basic.UserAction{}).
		Where("target_id = ? AND target_type = ? AND action_type = ?", id, "article", "bookmark").
		Count(&favoriteCount)

	// 查询该文章的点赞数
	var likeCount int64
	global.DB.Model(&basic.UserAction{}).
		Where("target_id = ? AND target_type = ? AND action_type = ?", id, "article", "like").
		Count(&likeCount)

	resp := *response.NewArticleInfoResp(article, user)
	resp.FavoriteCount = int(favoriteCount)
	resp.LikeCount = int(likeCount)
	return resp, nil
}

func (a *ArticleService) ListByUser(userId uint, pageReq request.PageReq) ([]response.ArticleListItemResp, int64, error) {
	query := global.DB.Model(&basic.Article{}).Where("user_id = ?", userId)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		global.Log.Error("统计用户文章总数失败", zap.Error(err), zap.Uint("userID", userId))
		return nil, 0, fmt.Errorf("查询用户文章列表失败，请稍后重试")
	}

	var articles []basic.Article
	if err := query.Order("is_top DESC, id DESC").
		Offset(pageReq.Offset()).
		Limit(pageReq.PageSize).
		Find(&articles).Error; err != nil {
		global.Log.Error("查询用户文章列表失败", zap.Error(err), zap.Uint("userID", userId))
		return nil, 0, fmt.Errorf("查询用户文章列表失败，请稍后重试")
	}

	favoriteCountMap := batchCountFavorites(articles)
	likeCountMap := batchCountLikes(articles)

	list := make([]response.ArticleListItemResp, 0, len(articles))
	for _, article := range articles {
		resp := response.NewArticleListItemResp(article)
		resp.FavoriteCount = favoriteCountMap[article.ID]
		resp.LikeCount = likeCountMap[article.ID]
		list = append(list, *resp)
	}
	return list, total, nil
}

func (a *ArticleService) List(req request.ArticleQueryListReq) ([]response.ArticleListItemResp, int64, error) {
	query := global.DB.Model(&basic.Article{})

	if req.Title != "" {
		query = query.Where("title LIKE ?", "%"+req.Title+"%")
	}
	if req.Summary != "" {
		query = query.Where("summary LIKE ?", "%"+req.Summary+"%")
	}
	if req.UserName != "" {
		query = query.Joins("JOIN users ON users.id = articles.user_id").
			Where("users.name LIKE ?", "%"+req.UserName+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		global.Log.Error("统计文章总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("查询文章列表失败，请稍后重试")
	}

	var articles []basic.Article
	if err := query.Order("is_top DESC, id DESC").
		Offset(req.Offset()).
		Limit(req.PageSize).
		Find(&articles).Error; err != nil {
		global.Log.Error("查询文章列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("查询文章列表失败，请稍后重试")
	}

	favoriteCountMap := batchCountFavorites(articles)
	likeCountMap := batchCountLikes(articles)

	list := make([]response.ArticleListItemResp, 0, len(articles))
	for _, article := range articles {
		resp := response.NewArticleListItemResp(article)
		resp.FavoriteCount = favoriteCountMap[article.ID]
		resp.LikeCount = likeCountMap[article.ID]
		list = append(list, *resp)
	}
	return list, total, nil
}

// batchCountFavorites 批量统计一批文章的收藏数，返回 articleID -> 收藏数 的 map
func batchCountFavorites(articles []basic.Article) map[uint]int {
	result := make(map[uint]int, len(articles))
	if len(articles) == 0 {
		return result
	}
	ids := make([]uint, 0, len(articles))
	for _, a := range articles {
		ids = append(ids, a.ID)
	}

	type favoriteCountRow struct {
		TargetID uint
		Count    int
	}
	var rows []favoriteCountRow
	err := global.DB.Model(&basic.UserAction{}).
		Select("target_id, COUNT(*) AS count").
		Where("target_id IN ? AND target_type = ? AND action_type = ?", ids, "article", "bookmark").
		Group("target_id").
		Scan(&rows).Error
	if err != nil {
		global.Log.Error("批量统计文章收藏数失败", zap.Error(err))
		return result
	}
	for _, row := range rows {
		result[row.TargetID] = row.Count
	}
	return result
}

// batchCountLikes 批量统计一批文章的点赞数，返回 articleID -> 点赞数 的 map
func batchCountLikes(articles []basic.Article) map[uint]int {
	result := make(map[uint]int, len(articles))
	if len(articles) == 0 {
		return result
	}
	ids := make([]uint, 0, len(articles))
	for _, a := range articles {
		ids = append(ids, a.ID)
	}

	type likeCountRow struct {
		TargetID uint
		Count    int
	}
	var rows []likeCountRow
	err := global.DB.Model(&basic.UserAction{}).
		Select("target_id, COUNT(*) AS count").
		Where("target_id IN ? AND target_type = ? AND action_type = ?", ids, "article", "like").
		Group("target_id").
		Scan(&rows).Error
	if err != nil {
		global.Log.Error("批量统计文章点赞数失败", zap.Error(err))
		return result
	}
	for _, row := range rows {
		result[row.TargetID] = row.Count
	}
	return result
}
