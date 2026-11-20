package request

// RoleCreateReq 创建角色请求
type RoleCreateReq struct {
	Name   string `json:"name" label:"角色名称" validate:"required,max=20"`
	Code   uint   `json:"code" label:"角色标识" validate:"required"`
	Sort   int    `json:"sort"`
	Status string `json:"status" label:"状态" validate:"omitempty,max=10"`
	Remark string `json:"remark" label:"备注" validate:"omitempty,max=255"`
}

// RoleUpdateReq 更新角色请求
type RoleUpdateReq struct {
	ID     uint    `json:"id" label:"角色ID" validate:"required"`
	Name   *string `json:"name,omitempty" label:"角色名称" validate:"omitempty,max=20"`
	Sort   *int    `json:"sort,omitempty" label:"排序"`
	Status *string `json:"status,omitempty" label:"状态" validate:"omitempty,max=10"`
	Remark *string `json:"remark,omitempty" label:"备注" validate:"omitempty,max=255"`
}

// RoleQueryReq 角色分页查询请求
type RoleQueryReq struct {
	Name    string `json:"name" label:"角色名称" validate:"omitempty,max=20"`
	Status  string `json:"status" label:"状态" validate:"omitempty,max=10"`
	PageReq        // 嵌入分页参数
}

// RoleAuthItem 单条授权规则
type RoleAuthItem struct {
	Path   string `json:"path" label:"请求路径" validate:"required,max=100"`                           // 请求路径
	Method string `json:"method" label:"请求方法" validate:"required,oneof=GET POST PUT DELETE PATCH"` // 请求方法（GET/POST/PUT/DELETE/PATCH）
}

// RoleAuthReq 角色授权请求（支持批量）
type RoleAuthReq struct {
	RoleID uint           `json:"roleId" label:"角色ID" validate:"required"`             // 角色ID
	Rules  []RoleAuthItem `json:"rules" label:"授权规则列表" validate:"required,min=1,dive"` // 授权规则列表
}
