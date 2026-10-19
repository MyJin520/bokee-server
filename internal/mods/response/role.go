package response

import (
	"bokee/internal/mods/basic"
	"time"
)

// RoleResp 角色响应结构体
type RoleResp struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Code      uint      `json:"code"`
	Sort      int       `json:"sort"`
	Status    string    `json:"status"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewRoleResp 将角色模型转换为角色响应结构体
func NewRoleResp(role basic.Role) *RoleResp {
	return &RoleResp{
		ID:        role.ID,
		Name:      role.Name,
		Code:      role.Code,
		Sort:      role.Sort,
		Status:    role.Status,
		Remark:    role.Remark,
		CreatedAt: role.CreatedAt,
		UpdatedAt: role.UpdatedAt,
	}
}

// NewPriRouteResp 构建私有路由响应项
func NewPriRouteResp(path, method, desc string) *PriRouteResp {
	return &PriRouteResp{
		Path:   path,
		Method: method,
		Desc:   desc,
	}
}

// NewPriModuleResp 构建模块分组私有路由响应
func NewPriModuleResp(module string, routes []PriRouteResp) *PriModuleResp {
	return &PriModuleResp{
		Module: module,
		Routes: routes,
	}
}
