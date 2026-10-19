package response

import (
	"bokee/internal/mods/basic"
	"time"
)

type JwtResp struct {
	Token  string `json:"token"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
}

// NewJwtResp 构建登录响应结构体
func NewJwtResp(token string, user basic.User) *JwtResp {
	return &JwtResp{
		Token:  token,
		Name:   user.Name,
		Avatar: user.Avatar,
		Email:  user.Email,
		Phone:  user.Phone,
	}
}

// UserRoleResp 用户角色响应结构体（脱敏：仅暴露展示所需字段）
type UserRoleResp struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Code uint   `json:"code"`
}

// UserInfoResp 用户信息响应结构体（脱敏：剔除密码等敏感字段）
type UserInfoResp struct {
	ID             uint           `json:"id"`
	Name           string         `json:"name"`
	Phone          string         `json:"phone"`
	Email          string         `json:"email"`
	Status         string         `json:"status"`
	Avatar         string         `json:"avatar"`
	FollowersCount int            `json:"followersCount"`
	Roles          []UserRoleResp `json:"roles"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// PriRouteResp 私有路由项
type PriRouteResp struct {
	Path   string `json:"path"`   // 路由路径
	Method string `json:"method"` // 请求方法
	Desc   string `json:"desc"`   // 路由描述
}

// PriModuleResp 模块分组私有路由
type PriModuleResp struct {
	Module string         `json:"module"` // 模块名
	Routes []PriRouteResp `json:"routes"` // 该模块下的路由
}

// NewUserInfoResp 将用户模型转换为脱敏响应结构（剔除密码等敏感字段）
func NewUserInfoResp(user basic.User) *UserInfoResp {
	roles := make([]UserRoleResp, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, UserRoleResp{ID: role.ID, Name: role.Name, Code: role.Code})
	}
	return &UserInfoResp{
		ID:        user.ID,
		Name:      user.Name,
		Phone:     user.Phone,
		Email:     user.Email,
		Status:    user.Status,
		Avatar:    user.Avatar,
		Roles:     roles,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

type UserFindResp struct {
	ID             uint      `json:"id"`
	Name           string    `json:"name"`
	Avatar         string    `json:"avatar"`
	FollowersCount int       `json:"followersCount"`
	CreatedAt      time.Time `json:"createdAt"`
}

func NewUserFindResp(user basic.User) *UserFindResp {
	return &UserFindResp{
		ID:        user.ID,
		Name:      user.Name,
		Avatar:    user.Avatar,
		CreatedAt: user.CreatedAt,
	}
}
