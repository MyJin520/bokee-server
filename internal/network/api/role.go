package api

import (
	"bokee/internal/mods/request"
	"bokee/internal/mods/response"
	"bokee/internal/network/service"
	"bokee/pkg/routex"

	"github.com/gin-gonic/gin"
)

type RoleApi struct{}

var roleService = &service.RoleService{}

// Create 创建角色
func (a *RoleApi) Create(c *gin.Context) {
	var req request.RoleCreateReq
	if err := routex.BindCheckStruct(c, &req); err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}

	resp, err := roleService.Create(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(resp, "创建角色成功", c)
}

// Update 更新角色
func (a *RoleApi) Update(c *gin.Context) {
	var req request.RoleUpdateReq
	if err := routex.BindCheckStruct(c, &req); err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}

	if err := roleService.Update(c.Request.Context(), req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("更新角色成功", c)
}

// Delete 删除角色
func (a *RoleApi) Delete(c *gin.Context) {
	id, err := routex.QueryUint(c, "id")
	if err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}

	if err := roleService.Delete(c.Request.Context(), id); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除角色成功", c)
}

// GetInfo 获取角色详情
func (a *RoleApi) GetInfo(c *gin.Context) {
	id, err := routex.QueryUint(c, "id")
	if err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}

	resp, err := roleService.GetInfo(c.Request.Context(), id)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(resp, "获取角色详情成功", c)
}

// List 分页获取角色列表
func (a *RoleApi) List(c *gin.Context) {
	var req request.RoleQueryReq
	if err := routex.BindCheckStruct(c, &req); err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}
	req.Normalize()
	list, total, err := roleService.List(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithPage(list, total, req.Page, req.PageSize, "获取角色列表成功", c)
}

// Auth 角色授权
func (a *RoleApi) Auth(c *gin.Context) {
	var req request.RoleAuthReq
	if err := routex.BindCheckStruct(c, &req); err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}

	if err := roleService.Auth(req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("角色授权成功", c)
}

// GetAllPriRule 获取所有私有路由（保留原有功能）
func (a *RoleApi) GetAllPriRule(c *gin.Context) {
	var page request.PageReq
	if err := routex.BindCheckStruct(c, &page); err != nil {
		response.FailWithRequest(err.Error(), c)
		return
	}
	page.Normalize()
	routes, total, err := roleService.GetAllPriRule(page)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithPage(routes, total, page.Page, page.PageSize, "获取私有路由列表成功", c)
}
