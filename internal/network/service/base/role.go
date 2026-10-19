package base

import (
	"bokee/global"
	"bokee/internal/mods/basic"
	"bokee/internal/mods/request"
	"bokee/internal/mods/response"
	"bokee/pkg/casbinx"
	"bokee/pkg/commons"
	"bokee/pkg/routex"
	"cmp"
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"slices"
	"strconv"
	"strings"
)

type RoleService struct{}

// Create 创建角色
func (s *RoleService) Create(req request.RoleCreateReq) (*response.RoleResp, error) {
	// 校验角色名称是否已存在
	var nameCount int64
	global.DB.Model(&basic.Role{}).Where("name = ?", req.Name).Count(&nameCount)
	if nameCount > 0 {
		global.Log.Warn("创建角色失败：角色名称已存在", zap.String("roleName", req.Name))
		return nil, fmt.Errorf("角色名称已存在")
	}

	// 校验角色标识是否已存在
	var codeCount int64
	global.DB.Model(&basic.Role{}).Where("code = ?", req.Code).Count(&codeCount)
	if codeCount > 0 {
		global.Log.Warn("创建角色失败：角色标识已存在", zap.Uint("roleCode", req.Code))
		return nil, fmt.Errorf("角色标识已存在")
	}

	// 默认值
	status := req.Status
	if status == "" {
		status = "normal"
	}

	role := basic.NewRole(req.Name, req.Code, req.Sort, status, req.Remark)

	if err := global.DB.Create(role).Error; err != nil {
		global.Log.Error("创建角色失败", zap.Error(err))
		return nil, fmt.Errorf("创建角色失败，请稍后重试")
	}

	return response.NewRoleResp(*role), nil
}

// Update 更新角色
func (s *RoleService) Update(ctx context.Context, req request.RoleUpdateReq) error {
	// 查询原角色
	var role basic.Role
	err := global.DB.Where("id = ?", req.ID).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Warn("更新角色失败：角色不存在", zap.Uint("roleID", req.ID))
			return fmt.Errorf("角色不存在")
		}
		global.Log.Error("查询角色失败", zap.Error(err))
		return fmt.Errorf("查询角色失败，请稍后重试")
	}

	// 禁止修改超级管理员角色信息
	if role.Code == uint(global.SuperRoleCode) {
		global.Log.Warn("更新角色失败：超级管理员角色不可修改", zap.Uint("roleID", req.ID))
		return fmt.Errorf("超级管理员角色不可修改")
	}

	// 校验角色名称是否与其他角色冲突
	if req.Name != nil && *req.Name != role.Name {
		var nameCount int64
		global.DB.Model(&basic.Role{}).Where("name = ? AND id != ?", *req.Name, req.ID).Count(&nameCount)
		if nameCount > 0 {
			global.Log.Warn("更新角色失败：角色名称已存在", zap.String("roleName", *req.Name))
			return fmt.Errorf("角色名称已存在")
		}
	}

	// 使用公共更新方法构建更新 map
	updates := commons.StructToUpdateMap(req)
	delete(updates, "id")

	// Sort 需与旧值比较，若未变更则不提交（避免因 Sort 相等而触发无意义的 UPDATE）
	if req.Sort != nil && *req.Sort == role.Sort {
		delete(updates, "sort")
	}

	if len(updates) == 0 {
		global.Log.Warn("更新角色失败：没有需要更新的字段", zap.Uint("roleID", req.ID))
		return fmt.Errorf("没有需要更新的字段")
	}

	result := global.DB.Model(&basic.Role{}).Where("id = ?", req.ID).Updates(updates)
	if result.Error != nil {
		global.Log.Error("更新角色失败", zap.Error(result.Error))
		return fmt.Errorf("更新角色失败，请稍后重试")
	}
	if result.RowsAffected == 0 {
		global.Log.Warn("更新角色失败：角色不存在或未做任何更改", zap.Uint("roleID", req.ID))
		return fmt.Errorf("角色不存在或未做任何更改")
	}

	return nil
}

// Delete 删除角色（软删除）
func (s *RoleService) Delete(ctx context.Context, id uint) error {
	// 查询角色
	var role basic.Role
	err := global.DB.Where("id = ?", id).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Warn("删除角色失败：角色不存在", zap.Uint("roleID", id))
			return fmt.Errorf("角色不存在")
		}
		global.Log.Error("查询角色失败", zap.Error(err))
		return fmt.Errorf("查询角色失败，请稍后重试")
	}

	// 禁止删除超级管理员角色
	if role.Code == uint(global.SuperRoleCode) {
		global.Log.Warn("删除角色失败：超级管理员角色不可删除", zap.Uint("roleID", id))
		return fmt.Errorf("超级管理员角色不可删除")
	}

	// 检查是否有用户关联此角色
	var userCount int64
	global.DB.Table("sys_user_roles").Where("role_id = ?", id).Count(&userCount)
	if userCount > 0 {
		global.Log.Warn("删除角色失败：该角色下存在用户", zap.Uint("roleID", id), zap.Int64("userCount", userCount))
		return fmt.Errorf("该角色下存在用户，无法删除，请先解除用户关联")
	}

	if err := global.DB.Delete(&role).Error; err != nil {
		global.Log.Error("删除角色失败", zap.Error(err), zap.Uint("roleID", id))
		return fmt.Errorf("删除角色失败，请稍后重试")
	}

	global.Log.Info("删除角色成功", zap.Uint("roleID", id), zap.String("roleName", role.Name))
	return nil
}

// GetInfo 获取单个角色详情
func (s *RoleService) GetInfo(ctx context.Context, id uint) (response.RoleResp, error) {
	var role basic.Role
	if err := global.DB.First(&role, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Warn("获取角色详情失败：角色不存在", zap.Uint("roleID", id))
			return response.RoleResp{}, fmt.Errorf("角色不存在")
		}

		global.Log.Error("查询角色失败", zap.Error(err))
		return response.RoleResp{}, fmt.Errorf("查询角色失败，请稍后重试")
	}

	return *response.NewRoleResp(role), nil
}

// List 分页获取角色列表
func (s *RoleService) List(req request.RoleQueryReq) ([]response.RoleResp, int64, error) {
	// 构建查询
	query := global.DB.Model(&basic.Role{})

	// 可选过滤条件
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		global.Log.Error("统计角色总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("查询角色列表失败，请稍后重试")
	}

	// 分页查询
	var roles []basic.Role
	if err := query.Order("sort ASC, id ASC").
		Offset(req.Offset()).
		Limit(req.PageSize).
		Find(&roles).Error; err != nil {
		global.Log.Error("查询角色列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("查询角色列表失败，请稍后重试")
	}
	list := make([]response.RoleResp, 0, len(roles))
	for _, role := range roles {
		list = append(list, *response.NewRoleResp(role))
	}

	return list, total, nil
}

// Auth 为角色批量授权（添加 Casbin 策略）
func (s *RoleService) Auth(req request.RoleAuthReq) error {
	// 查询角色
	var role basic.Role
	err := global.DB.Where("id = ?", req.RoleID).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Warn("角色授权失败：角色不存在", zap.Uint("roleID", req.RoleID))
			return fmt.Errorf("角色不存在")
		}
		global.Log.Error("查询角色失败", zap.Error(err))
		return fmt.Errorf("查询角色失败，请稍后重试")
	}

	// 禁止为超级管理员手动授权（超级管理员自动拥有全部权限）
	if role.Code == uint(global.SuperRoleCode) {
		global.Log.Warn("角色授权失败：超级管理员无需手动授权", zap.Uint("roleID", req.RoleID))
		return fmt.Errorf("超级管理员无需手动授权")
	}

	if len(req.Rules) == 0 {
		global.Log.Warn("角色授权失败：授权规则不能为空", zap.Uint("roleID", req.RoleID))
		return fmt.Errorf("授权规则不能为空")
	}

	// 构建 Casbin 策略：p = sub(roleCode), obj(path), act(method)
	roleCodeStr := strconv.Itoa(int(role.Code))
	policies := make([][]string, 0, len(req.Rules))
	for _, rule := range req.Rules {
		if rule.Path == "" || rule.Method == "" {
			global.Log.Warn("角色授权失败：路由路径和请求方法不能为空", zap.Uint("roleID", req.RoleID))
			return fmt.Errorf("路由路径和请求方法不能为空")
		}
		policies = append(policies, []string{roleCodeStr, rule.Path, strings.ToUpper(rule.Method)})
	}

	enforcer, err := casbinx.GetEnforcer()
	if err != nil {
		global.Log.Error("获取 Casbin 执行器失败", zap.Error(err))
		return fmt.Errorf("授权失败，请稍后重试")
	}

	// 批量添加策略
	success, err := enforcer.AddPolicies(policies)
	if err != nil {
		global.Log.Error("批量添加 Casbin 策略失败", zap.Error(err))
		return fmt.Errorf("授权失败，请稍后重试")
	}
	if !success {
		global.Log.Warn("角色授权失败：授权规则与已有规则重复", zap.Uint("roleID", req.RoleID))
		return fmt.Errorf("授权规则与已有规则重复，无需重复添加")
	}

	// 重载策略使更改立即生效
	if err := casbinx.ReloadPolicy(); err != nil {
		global.Log.Error("重载 Casbin 策略失败", zap.Error(err))
		return fmt.Errorf("授权成功但策略重载失败，请稍后重试")
	}

	global.Log.Info("角色授权成功",
		zap.Uint("roleID", req.RoleID),
		zap.String("roleName", role.Name),
		zap.Int("ruleCount", len(req.Rules)),
	)
	return nil
}

// GetAllPriRule 获取所有私有路由（按模块聚合，组内路由按路径排序）
func (s *RoleService) GetAllPriRule(page request.PageReq) ([]response.PriModuleResp, int64, error) {
	policies, err := casbinx.GetPrivateRoutes()
	if err != nil {
		global.Log.Error("获取私有路由失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取私有路由列表失败，请稍后重试")
	}

	list := groupPriRoutes(policies)
	total := int64(len(list))

	start := page.Offset()
	if start >= len(list) {
		return []response.PriModuleResp{}, total, nil
	}

	end := min(start+page.PageSize, len(list))
	return list[start:end], total, nil
}

func groupPriRoutes(policies [][]string) []response.PriModuleResp {
	modules := make(map[string][]response.PriRouteResp)
	order := make([]string, 0)
	seen := make(map[string]struct{})

	for _, p := range policies {
		if len(p) < 3 {
			continue
		}

		key := p[2] + "|" + p[1]
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		meta := routex.Get(p[2], p[1])
		module := meta.Module
		if module == "" {
			module = "未分类"
		}

		if _, exists := modules[module]; !exists {
			order = append(order, module)
		}

		modules[module] = append(modules[module], *response.NewPriRouteResp(p[1], p[2], meta.Desc))
	}

	list := make([]response.PriModuleResp, 0, len(order))
	for _, module := range order {
		routes := modules[module]

		slices.SortFunc(routes, func(a, b response.PriRouteResp) int {
			return cmp.Compare(a.Path, b.Path)
		})

		list = append(list, *response.NewPriModuleResp(module, routes))
	}

	return list
}
