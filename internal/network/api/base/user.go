package base

import (
	"bokee/global"
	"bokee/internal/mods/request"
	"bokee/internal/mods/response"
	"bokee/internal/network/middleware"
	"bokee/internal/network/service/base"
	"bokee/pkg/jwtx"
	"bokee/pkg/routex"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UserApi struct{}

var userService = &base.UserService{}

func (u *UserApi) Create(c *gin.Context) {
	var req request.UserCreateReq
	if !routex.BindCheckStruct(c, &req) {
		return
	}
	if err := userService.Create(c.Request.Context(), req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("用户创建成功", c)
}

func (u *UserApi) Login(c *gin.Context) {
	var req request.UserLoginReq
	if !routex.BindCheckStruct(c, &req) {
		return
	}
	jwtResponse, err := userService.Login(c.Request.Context(), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(jwtResponse, "登录成功", c)
}

func (u *UserApi) Update(c *gin.Context) {
	var req request.UserUpdateReq
	if !routex.BindCheckStruct(c, &req) {
		return
	}
	currentUserID, _ := middleware.GetUserIDFromContext(c)
	if err := userService.Update(c.Request.Context(), req, currentUserID); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("用户更新成功", c)
}

func (u *UserApi) Logout(c *gin.Context) {
	// 从请求头提取 token（Bearer）
	tokenString, err := jwtx.ExtractBearerToken(c.GetHeader("Authorization"))
	if err != nil {
		global.Log.Warn("登出失败：认证令牌无效", zap.String("err", err.Error()))
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := userService.Logout(c.Request.Context(), tokenString); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("登出成功", c)
}

func (u *UserApi) ForgetPassword(c *gin.Context) {
	var req request.ForgetPasswordReq
	if !routex.BindCheckStruct(c, &req) {
		return
	}
	currentUserID, _ := middleware.GetUserIDFromContext(c)
	if err := userService.ForgetPassword(c.Request.Context(), req, currentUserID); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("密码重置成功", c)
}

// OperateRoles 用户角色绑定/解绑
func (u *UserApi) OperateRoles(c *gin.Context) {
	var req request.UserRoleBindReq
	if !routex.BindCheckStruct(c, &req) {
		return
	}

	msg, err := userService.OperateRoles(c.Request.Context(), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage(msg, c)
}

func (u *UserApi) GetInfo(c *gin.Context) {
	userID, ok := request.ParseQueryUint(c, "id")
	if !ok {
		userID, _ = middleware.GetUserIDFromContext(c)
	}

	user, err := userService.GetInfo(c.Request.Context(), userID)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(user, "获取用户信息成功", c)
}

func (u *UserApi) List(c *gin.Context) {
	var req request.UserListReq
	if !routex.BindCheckStruct(c, &req) {
		return
	}
	req.Normalize()
	users, total, err := userService.List(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithPage(users, total, req.Page, req.PageSize, "获取用户列表成功", c)
}

func (u *UserApi) FindUsers(c *gin.Context) {
	var req request.UserFindReq
	if !routex.BindCheckStruct(c, &req) {
		return
	}
	req.Normalize()
	users, total, err := userService.FindUsers(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithPage(users, total, req.Page, req.PageSize, "查询用户成功", c)
}
