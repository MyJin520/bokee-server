package request

// UserCreateReq 创建用户
type UserCreateReq struct {
	Name     string `json:"name" label:"用户名" validate:"required"`
	Password string `json:"password" label:"密码" validate:"required,min=6"`
	Phone    string `json:"phone" label:"手机号"`
	Email    string `json:"email" label:"邮箱" validate:"omitempty,email"`
	Avatar   string `json:"avatar"`
}

// UserLoginReq 用户登陆
type UserLoginReq struct {
	Name     string `json:"name" label:"用户名" validate:"required"`
	Password string `json:"password" label:"密码" validate:"required"`
}

// UserUpdateReq 更新用户
type UserUpdateReq struct {
	Name   *string `json:"name" label:"用户名"`
	Phone  *string `json:"phone" label:"手机号"`
	Email  *string `json:"email" label:"邮箱" validate:"omitempty,email"`
	Avatar *string `json:"avatar" label:"头像"`
}

// ForgetPasswordReq 忘记密码
type ForgetPasswordReq struct {
	OldPassword   string `json:"oldPassword" label:"旧密码" validate:"required"`
	NewPassword   string `json:"newPassword" label:"新密码" validate:"required,min=6"`
	SpecifyUserID uint   `json:"specifyUserId"`
}

// UserListReq 用户列表查询请求
type UserListReq struct {
	PageReq
	Name   string `json:"name"`   // 用户名模糊搜索
	Phone  string `json:"phone"`  // 手机号模糊搜索
	Email  string `json:"email"`  // 邮箱模糊搜索
	Status string `json:"status"` // 账号状态精确匹配
}

// UserRoleBindReq 用户角色绑定请求（支持批量绑定多个角色）
type UserRoleBindReq struct {
	UserID  uint   `json:"userId" label:"用户ID" validate:"required"`                    // 用户ID
	RoleIDs []uint `json:"roleIds" label:"角色ID列表" validate:"required,min=1"`           // 角色ID列表（传单个角色ID也使用数组格式）
	Operate string `json:"operate" label:"操作行为" validate:"required,oneof=bind unbind"` // 绑定或解绑 bind || unbind
}

type UserFindReq struct {
	PageReq
	ID   uint   `json:"id"`   // 用户ID精确查询（作者主页直达，可选）
	Name string `json:"name"` // 仅用户名模糊查询
}
