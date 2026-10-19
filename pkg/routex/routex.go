package routex

import (
	"bokee/global"
	"bokee/internal/mods/response"
	"bokee/pkg/verifyx"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// RouteMeta 路由元信息：描述 + 所属模块
type RouteMeta struct {
	Module string // 所属模块，如 "用户" / "角色"
	Desc   string // 路由描述
}

// Group 内嵌 gin.RouterGroup 的路由组包装：注册路由的同时记录路由描述与所属模块，
// 调用方式与 gin 原生一致，仅将描述作为第二个参数：g.POST("/x", "描述", handler)。
type Group struct {
	*gin.RouterGroup
	module string
}

// NewGroup 包装 gin 路由组：注册路由时记录描述，并标记所属模块
func NewGroup(module string, rg *gin.RouterGroup) *Group {
	return &Group{RouterGroup: rg, module: module}
}

var (
	mu      sync.RWMutex
	descMap = make(map[string]RouteMeta)
)

func key(method, path string) string {
	return method + "|" + path
}

// joinPath 拼接路由组前缀与相对路径，避免双斜杠
func joinPath(base, relative string) string {
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(relative, "/")
}

// Set 记录路由元信息（启动注册阶段调用，后写覆盖）
func Set(method, path, module, desc string) {
	mu.Lock()
	defer mu.Unlock()
	descMap[key(method, path)] = RouteMeta{Module: module, Desc: desc}
}

// Get 获取路由元信息（模块 + 描述），未注册返回零值
func Get(method, path string) RouteMeta {
	mu.RLock()
	defer mu.RUnlock()
	return descMap[key(method, path)]
}

func (g *Group) POST(path, desc string, handler gin.HandlerFunc) {
	g.RouterGroup.POST(path, handler)
	Set(http.MethodPost, joinPath(g.BasePath(), path), g.module, desc)
}

func (g *Group) GET(path, desc string, handler gin.HandlerFunc) {
	g.RouterGroup.GET(path, handler)
	Set(http.MethodGet, joinPath(g.BasePath(), path), g.module, desc)
}

func (g *Group) PUT(path, desc string, handler gin.HandlerFunc) {
	g.RouterGroup.PUT(path, handler)
	Set(http.MethodPut, joinPath(g.BasePath(), path), g.module, desc)
}

func (g *Group) DELETE(path, desc string, handler gin.HandlerFunc) {
	g.RouterGroup.DELETE(path, handler)
	Set(http.MethodDelete, joinPath(g.BasePath(), path), g.module, desc)
}

func QueryUint(c *gin.Context, key string) (uint, bool) {
	str := c.Query(key)
	if str == "" {
		response.FailWithMessage(key+"不能为空", c)
		return 0, false
	}
	id, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		response.FailWithMessage(key+"参数异常", c)
		return 0, false
	}
	return uint(id), true
}

// BindCheckStruct 绑定并校验 JSON 请求体；失败时已写入错误响应并返回 false
func BindCheckStruct(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		global.Log.Error("请求参数异常", zap.Error(err))
		response.FailWithRequest("请求参数异常", c)
		return false
	}
	if errMsg := verifyx.CheckStruct(req); errMsg != "" {
		global.Log.Error("请求参数校验失败", zap.String("errMsg", errMsg))
		response.FailWithRequest(errMsg, c)
		return false
	}
	return true
}
