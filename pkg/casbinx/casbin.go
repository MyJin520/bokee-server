package casbinx

import (
	"bokee/global"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	syncedEnforcer *casbin.SyncedCachedEnforcer
	once           sync.Once
	initErr        error
)

// casbinModelText 定义 RBAC 模型：
// - g(r.sub, p.sub) 支持角色继承（用户→角色→权限）
// - keyMatch3 支持 RESTful 路径匹配（/api/user/:id）
// - regexMatch 支持正则匹配 HTTP 方法
const casbinModelText = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch3(r.obj, p.obj) && regexMatch(r.act, p.act)
`

// InitEnforcer 初始化 Casbin 执行器（程序启动时调用一次）。
// 失败时返回 error，调用方应阻止程序继续启动。
func InitEnforcer() error {
	once.Do(func() {
		// 1. 创建 GORM 适配器（策略存储在 casbin_rule 表中）
		adapter, err := gormadapter.NewAdapterByDB(global.DB)
		if err != nil {
			initErr = fmt.Errorf("casbin 适配器创建失败: %w", err)
			global.Log.Error("casbin 适配器创建失败", zap.Error(err))
			return
		}

		// 2. 从字符串解析 RBAC 模型
		m, err := model.NewModelFromString(casbinModelText)
		if err != nil {
			initErr = fmt.Errorf("casbin 模型解析失败: %w", err)
			global.Log.Error("casbin 模型解析失败", zap.Error(err))
			return
		}

		// 3. 创建带缓存和并发安全的执行器
		enforcer, err := casbin.NewSyncedCachedEnforcer(m, adapter)
		if err != nil {
			initErr = fmt.Errorf("casbin 执行器创建失败: %w", err)
			global.Log.Error("casbin 执行器创建失败", zap.Error(err))
			return
		}

		// 4. 设置缓存过期时间（秒），过期后自动从数据库重新加载策略
		enforcer.SetExpireTime(60 * 60)

		// 5. 启动时预加载所有策略到内存缓存
		if err := enforcer.LoadPolicy(); err != nil {
			initErr = fmt.Errorf("casbin 策略加载失败: %w", err)
			global.Log.Error("casbin 策略加载失败", zap.Error(err))
			return
		}

		syncedEnforcer = enforcer
		global.Log.Info("casbin 初始化完成")
	})
	return initErr
}

// GetEnforcer 返回已初始化的 Casbin 执行器。
// 注意：必须先调用 InitEnforcer 并确保成功，否则返回 nil 和对应的错误。
func GetEnforcer() (*casbin.SyncedCachedEnforcer, error) {
	if syncedEnforcer == nil {
		return nil, initErr
	}
	return syncedEnforcer, nil
}

// Enforce 权限校验。
// sub: 用户名，obj: 请求路径（如 /pri/user/info），act: HTTP 方法（GET/POST/PUT/DELETE）。
func Enforce(sub, obj, act string) (bool, error) {
	enforcer, err := GetEnforcer()
	if err != nil {
		return false, err
	}
	return enforcer.Enforce(sub, obj, act)
}

// AddPolicyForUser 为用户直接添加权限策略。
func AddPolicyForUser(user, path, method string) (bool, error) {
	enforcer, err := GetEnforcer()
	if err != nil {
		return false, err
	}
	return enforcer.AddPolicy(user, path, method)
}

// RemovePolicyForUser 移除用户的直接权限策略。
func RemovePolicyForUser(user, path, method string) (bool, error) {
	enforcer, err := GetEnforcer()
	if err != nil {
		return false, err
	}
	return enforcer.RemovePolicy(user, path, method)
}

// AddRoleForUser 为用户分配角色（写入 g 规则）。
func AddRoleForUser(user, role string) (bool, error) {
	enforcer, err := GetEnforcer()
	if err != nil {
		return false, err
	}
	return enforcer.AddGroupingPolicy(user, role)
}

// RemoveRoleForUser 移除用户的角色。
func RemoveRoleForUser(user, role string) (bool, error) {
	enforcer, err := GetEnforcer()
	if err != nil {
		return false, err
	}
	return enforcer.RemoveGroupingPolicy(user, role)
}

// GetRolesForUser 获取用户的直接角色（不包含继承）。
func GetRolesForUser(user string) ([]string, error) {
	enforcer, err := GetEnforcer()
	if err != nil {
		return nil, err
	}
	return enforcer.GetRolesForUser(user)
}

// GetImplicitRolesForUser 获取用户的所有角色（包含继承链路）。
func GetImplicitRolesForUser(user string) ([]string, error) {
	enforcer, err := GetEnforcer()
	if err != nil {
		return nil, err
	}
	return enforcer.GetImplicitRolesForUser(user)
}

// ReloadPolicy 从数据库重新加载所有策略到内存缓存。
// 管理员通过 API 修改权限后，调用此方法使其立即生效。
func ReloadPolicy() error {
	enforcer, err := GetEnforcer()
	if err != nil {
		return err
	}
	return enforcer.LoadPolicy()
}

// GetAllPolicies 获取所有权限策略（p 规则）。
func GetAllPolicies() ([][]string, error) {
	enforcer, err := GetEnforcer()
	if err != nil {
		return nil, err
	}
	return enforcer.GetPolicy()
}

// GetAllRoles 获取所有角色分配关系（g 规则）。
func GetAllRoles() ([][]string, error) {
	enforcer, err := GetEnforcer()
	if err != nil {
		return nil, err
	}
	return enforcer.GetGroupingPolicy()
}

func InitSuperRoleCasbin(engine *gin.Engine) {
	roleCodeStr := strconv.Itoa(global.SuperRoleCode)
	enforcer, err := GetEnforcer()
	if err != nil {
		global.Log.Error("获取 Casbin 执行器失败", zap.Error(err))
		return
	}

	routes := engine.Routes()
	global.Log.Info(fmt.Sprintf("开始为超级管理员 [%s] 同步路由权限，共扫描 %d 条路由", roleCodeStr, len(routes)))

	// 构建期望的策略集合（path + method）
	desired := make(map[string]struct{}, len(routes))
	desiredPolicies := make([][]string, 0, len(routes))
	for _, route := range routes {
		if strings.HasPrefix(route.Path, "/pub/") {
			continue
		}
		key := route.Path + "|" + route.Method
		if _, exists := desired[key]; exists {
			continue // 防止重复路由
		}
		desired[key] = struct{}{}
		desiredPolicies = append(desiredPolicies, []string{roleCodeStr, route.Path, route.Method})
	}

	// 获取当前超级角色已有的策略
	existingPolicies, err := enforcer.GetFilteredPolicy(0, roleCodeStr)
	if err != nil {
		global.Log.Error("获取超级管理员现有策略失败", zap.Error(err))
		return
	}

	existing := make(map[string]struct{}, len(existingPolicies))
	for _, p := range existingPolicies {
		if len(p) < 3 {
			continue
		}
		key := p[1] + "|" + p[2]
		existing[key] = struct{}{}
	}

	// 计算需要新增和删除的策略
	var toAdd [][]string
	var toRemove [][]string

	for _, p := range desiredPolicies {
		key := p[1] + "|" + p[2]
		if _, ok := existing[key]; !ok {
			toAdd = append(toAdd, p)
		}
	}

	for _, p := range existingPolicies {
		if len(p) < 3 {
			continue
		}
		key := p[1] + "|" + p[2]
		if _, ok := desired[key]; !ok {
			toRemove = append(toRemove, p)
		}
	}

	// 执行增量更新
	if len(toRemove) > 0 {
		if _, err := enforcer.RemovePolicies(toRemove); err != nil {
			global.Log.Error("移除过期策略失败", zap.Error(err))
			return
		}
		global.Log.Info(fmt.Sprintf("超级管理员移除过期策略 %d 条", len(toRemove)))
	}

	if len(toAdd) > 0 {
		if _, err := enforcer.AddPolicies(toAdd); err != nil {
			global.Log.Error("批量添加新策略失败", zap.Error(err))
			return
		}
		global.Log.Info(fmt.Sprintf("超级管理员新增策略 %d 条", len(toAdd)))
	}

	if len(toAdd) == 0 && len(toRemove) == 0 {
		global.Log.Info("超级管理员策略已是最新，无需更新")
	} else {
		global.Log.Info(fmt.Sprintf("超级管理员权限同步完成（新增 %d，删除 %d）", len(toAdd), len(toRemove)))
	}
}

// defaultUserRoleRules 普通用户（注册用户）默认可访问的自助路由：
var defaultUserRoleRules = [][2]string{
	{"/pri/user/get_info", "GET"},
	{"/pri/user/update", "PUT"},
	{"/pri/user/logout", "GET"},
	{"/pri/user/forget_password", "POST"},
	{"/pri/user/find_users", "POST"},
	{"/pri/article/create", "POST"},
	{"/pri/article/update", "PUT"},
	{"/pri/article/delete", "DELETE"},
	{"/pri/user_action/create", "POST"},
	{"/pri/user_action/delete", "DELETE"},
	{"/pri/user_action/list", "POST"},
	{"/pri/file/uploads", "POST"},
}

// InitDefaultUserRoleCasbin 为普通用户角色幂等补齐自助路由权限（只增不删，
// 管理员后续通过角色授权接口追加的权限不会被回收）
func InitDefaultUserRoleCasbin(engine *gin.Engine) {
	roleCodeStr := strconv.Itoa(global.UserRoleCode)
	enforcer, err := GetEnforcer()
	if err != nil {
		global.Log.Error("获取 Casbin 执行器失败", zap.Error(err))
		return
	}

	// 已注册路由集合，用于校验权限清单中的路径真实存在
	registered := make(map[string]struct{}, len(engine.Routes()))
	for _, route := range engine.Routes() {
		registered[route.Path+"|"+route.Method] = struct{}{}
	}

	desiredPolicies := make([][]string, 0, len(defaultUserRoleRules))
	for _, rule := range defaultUserRoleRules {
		path, method := rule[0], rule[1]
		if _, ok := registered[path+"|"+method]; !ok {
			global.Log.Warn("普通用户默认权限路由不存在，已跳过", zap.String("path", path), zap.String("method", method))
			continue
		}
		desiredPolicies = append(desiredPolicies, []string{roleCodeStr, path, method})
	}

	existingPolicies, err := enforcer.GetFilteredPolicy(0, roleCodeStr)
	if err != nil {
		global.Log.Error("获取普通用户现有策略失败", zap.Error(err))
		return
	}
	existing := make(map[string]struct{}, len(existingPolicies))
	for _, p := range existingPolicies {
		if len(p) >= 3 {
			existing[p[1]+"|"+p[2]] = struct{}{}
		}
	}

	var toAdd [][]string
	for _, p := range desiredPolicies {
		key := p[1] + "|" + p[2]
		if _, ok := existing[key]; !ok {
			toAdd = append(toAdd, p)
		}
	}

	if len(toAdd) == 0 {
		global.Log.Info("普通用户角色策略已是最新，无需更新")
		return
	}
	if _, err := enforcer.AddPolicies(toAdd); err != nil {
		global.Log.Error("批量添加普通用户策略失败", zap.Error(err))
		return
	}
	global.Log.Info(fmt.Sprintf("普通用户角色权限同步完成（新增 %d 条）", len(toAdd)))
}

// GetPrivateRoutes 获取所有私有路由策略
func GetPrivateRoutes() ([][]string, error) {
	allPolicies, err := GetAllPolicies()
	if err != nil {
		return nil, err
	}
	var privateRoutes [][]string
	for _, policy := range allPolicies {
		// policy 结构: [sub, obj, act]
		if len(policy) >= 3 && !strings.HasPrefix(policy[1], "/pub/") {
			privateRoutes = append(privateRoutes, policy)
		}
	}
	return privateRoutes, nil
}
