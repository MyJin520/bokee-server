package base

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bokee/global"
	"bokee/internal/mods/basic"
	"bokee/internal/mods/request"
	"bokee/internal/mods/response"
	"bokee/pkg/cachex"
	"bokee/pkg/commons"
	"bokee/pkg/cryptox/hash"
	"bokee/pkg/jwtx"
	"bokee/pkg/limitx"
	"bokee/pkg/redisx"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserService struct{}

var (
	// loginLimiter 登录防爆破：10min 窗口内 5 次失败锁定 30min
	loginLimiter = &limitx.FailLimiter{Scope: "user:login", Window: 10 * time.Minute, Max: 5, Lock: 30 * time.Minute}
	// resetLimiter 重置密码防爆破：10min 窗口内 5 次失败锁定 30min（仅非管理员旧密码错误计数）
	resetLimiter = &limitx.FailLimiter{Scope: "user:reset", Window: 10 * time.Minute, Max: 5, Lock: 30 * time.Minute}
)

// userContactQuery 根据手机号/邮箱构造用户联系方式查询条件
func userContactQuery(db *gorm.DB, phone, email string) *gorm.DB {
	switch {
	case phone != "" && email != "":
		return db.Where("phone = ? OR email = ?", phone, email)
	case phone != "":
		return db.Where("phone = ?", phone)
	case email != "":
		return db.Where("email = ?", email)
	default:
		return db
	}
}

// hasRole 判断用户是否拥有指定角色
func hasRole(user basic.User, roleCode uint) bool {
	for _, role := range user.Roles {
		if role.Code == roleCode {
			return true
		}
	}
	return false
}

// Create 用户注册：内置注册防刷（同一手机号/邮箱窗口内仅允许尝试一次），Redis 异常放行不影响注册
func (s *UserService) Create(ctx context.Context, req request.UserCreateReq) error {
	// 注册防刷：未填手机号时以邮箱作为限流维度，避免空手机号共享同一个限流键
	limitIdentity := req.Phone
	if limitIdentity == "" {
		limitIdentity = req.Email
	}
	if ok, err := redisx.SetNX(ctx, cachex.Key(cachex.NSUser, "register", limitIdentity), "1", time.Minute); err != nil {
		global.Log.Error("注册限流检查失败", zap.Error(err), zap.String("identity", limitIdentity))
	} else if !ok {
		global.Log.Warn("注册过于频繁", zap.String("identity", limitIdentity))
		return fmt.Errorf("注册过于频繁，请稍后再试")
	}

	// 唯一性校验：空值不参与匹配
	if req.Phone != "" || req.Email != "" {
		var count int64
		err := userContactQuery(global.DB.Model(&basic.User{}), req.Phone, req.Email).Count(&count).Error
		if err != nil {
			global.Log.Error("注册唯一性校验失败", zap.Error(err))
			return fmt.Errorf("注册失败，请稍后重试")
		}
		if count > 0 {
			global.Log.Warn("注册失败：手机号或邮箱已被注册", zap.String("phone", req.Phone), zap.String("email", req.Email))
			return fmt.Errorf("注册失败：手机号或邮箱已被注册")
		}
	}

	hashedPwd, err := hash.GeneratePassword(req.Password)
	if err != nil {
		global.Log.Error("用户密码加密失败", zap.Error(err))
		return fmt.Errorf("注册失败，密码加密失败请联系管理员")
	}

	user := basic.NewUser(req.Name, hashedPwd, req.Phone, req.Email, "", req.Avatar)

	// 新用户绑定普通用户默认角色；角色缺失时不阻断注册，仅记录日志
	var defaultRole basic.Role
	if err := global.DB.Where("code = ?", global.UserRoleCode).First(&defaultRole).Error; err != nil {
		global.Log.Error("查询普通用户默认角色失败，新用户将暂无角色", zap.Error(err))
	} else {
		user.Roles = []basic.Role{defaultRole}
	}

	if err := global.DB.Create(user).Error; err != nil {
		global.Log.Error("注册失败", zap.Error(err))
		return fmt.Errorf("注册失败，请稍后重试")
	}
	return nil
}

// Login 用户登录：内置登录防爆破（失败计数 + 锁定），Redis 异常放行不影响正常登录
func (s *UserService) Login(ctx context.Context, req request.UserLoginReq) (*response.JwtResp, error) {
	if remain, locked := loginLimiter.IsLocked(ctx, req.Name); locked {
		global.Log.Warn("登录失败：账号已被锁定", zap.String("username", req.Name))
		return nil, fmt.Errorf("尝试次数过多，请%d分钟后再试", int(remain.Minutes())+1)
	}

	var user basic.User
	err := global.DB.Select("id", "password", "name", "status", "avatar", "email", "phone").Preload("Roles").
		Where("name = ?", req.Name).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Warn("登录失败：用户名不存在", zap.String("username", req.Name))
			loginLimiter.RecordFail(ctx, req.Name)
			return nil, fmt.Errorf("用户名或密码错误")
		}
		global.Log.Error("登录数据库查询失败", zap.Error(err), zap.String("username", req.Name))
		return nil, fmt.Errorf("系统繁忙，请稍后重试")
	}

	if user.Status != "normal" {
		global.Log.Warn("账号已被禁用", zap.String("username", req.Name))
		return nil, fmt.Errorf("账号已被禁用，请联系管理员")
	}
	if err := hash.CompareHashAndPassword(user.Password, req.Password); err != nil {
		global.Log.Warn("密码校验失败", zap.String("username", req.Name))
		loginLimiter.RecordFail(ctx, req.Name)
		return nil, fmt.Errorf("用户名或密码错误")
	}

	loginLimiter.Clear(ctx, req.Name)

	roleCodes := make([]uint, 0, len(user.Roles))
	for _, role := range user.Roles {
		roleCodes = append(roleCodes, role.Code)
	}

	token, err := jwtx.GenerateToken(user.ID, user.Name, roleCodes)
	if err != nil {
		global.Log.Error("生成Token失败", zap.Error(err), zap.Uint("userID", user.ID))
		return nil, fmt.Errorf("登录失败，请稍后重试")
	}

	return response.NewJwtResp(token, user), nil
}

// Update 更新当前用户资料，落库成功后失效用户信息缓存
func (s *UserService) Update(ctx context.Context, req request.UserUpdateReq, uid uint) error {
	updates := commons.StructToUpdateMap(req)
	if len(updates) == 0 {
		global.Log.Warn("更新用户失败：没有需要更新的字段", zap.Uint("userID", uid))
		return fmt.Errorf("没有需要更新的字段")
	}

	checkPhone := ""
	checkEmail := ""
	if req.Phone != nil {
		checkPhone = *req.Phone
	}
	if req.Email != nil {
		checkEmail = *req.Email
	}

	// 手机号/邮箱唯一性校验：仅当本次更新携带非空手机号或邮箱时执行，排除自身
	if checkPhone != "" || checkEmail != "" {
		var duplicateCount int64
		err := userContactQuery(global.DB.Model(&basic.User{}).Where("id != ?", uid), checkPhone, checkEmail).
			Count(&duplicateCount).Error
		if err != nil {
			global.Log.Error("更新用户唯一性校验失败", zap.Error(err), zap.Uint("userID", uid))
			return fmt.Errorf("更新用户失败，请稍后重试")
		}
		if duplicateCount > 0 {
			global.Log.Warn("更新用户失败：手机号或邮箱已被占用", zap.Uint("userID", uid))
			return fmt.Errorf("手机号或邮箱已被其他用户使用")
		}
	}

	// TODO: 更新邮箱前需验证邮箱验证码
	result := global.DB.Model(&basic.User{}).Where("id = ?", uid).Updates(updates)
	if result.Error != nil {
		global.Log.Error("更新用户失败", zap.Error(result.Error), zap.Uint("userID", uid))
		return fmt.Errorf("更新用户失败，请稍后重试")
	}
	if result.RowsAffected == 0 {
		global.Log.Warn("更新用户失败：用户不存在或未做任何更改", zap.Uint("userID", uid))
		return fmt.Errorf("用户不存在或未做任何更改")
	}
	return nil
}

// Logout 用户登出：将当前 token 加入 Redis 黑名单
// token 由 API 层从请求头提取后传入，不依赖 gin
func (s *UserService) Logout(ctx context.Context, tokenString string) error {
	claims, err := jwtx.ParseToken(tokenString)
	if err != nil {
		global.Log.Error("登出失败：无效的认证令牌", zap.Error(err))
		return fmt.Errorf("无效的认证令牌")
	}

	remaining := time.Until(claims.ExpiresAt.Time)
	if remaining <= 0 {
		return nil
	}

	if err := redisx.BlacklistToken(ctx, tokenString, remaining); err != nil {
		global.Log.Error("登出写入 Redis 黑名单失败", zap.Error(err))
		return fmt.Errorf("登出失败，请稍后重试")
	}

	global.Log.Info("用户登出成功",
		zap.Uint("userID", claims.UserID),
		zap.String("username", claims.UserName),
		zap.Duration("blacklist_ttl", remaining),
	)
	return nil
}

// ForgetPassword 重置密码：内置防爆破（非管理员旧密码错误计数 + 锁定），Redis 异常放行不影响主流程
func (s *UserService) ForgetPassword(ctx context.Context, req request.ForgetPasswordReq, currentUserID uint) error {
	operatorID := fmt.Sprintf("%d", currentUserID)
	if remain, locked := resetLimiter.IsLocked(ctx, operatorID); locked {
		global.Log.Warn("重置密码失败：操作已被锁定", zap.Uint("userID", currentUserID))
		return fmt.Errorf("尝试次数过多，请%d分钟后再试", int(remain.Minutes())+1)
	}

	var targetUserID uint
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		var currentUser basic.User
		err := tx.Where("id = ?", currentUserID).Preload("Roles").First(&currentUser).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				global.Log.Warn("重置密码失败：用户不存在", zap.Uint("userID", currentUserID))
				return fmt.Errorf("用户不存在")
			}
			global.Log.Error("查询用户失败", zap.Error(err), zap.Uint("userID", currentUserID))
			return fmt.Errorf("查询用户失败，请稍后重试")
		}

		isAdmin := hasRole(currentUser, global.SuperRoleCode)
		targetUserID = currentUserID
		if isAdmin && req.SpecifyUserID != 0 {
			targetUserID = req.SpecifyUserID
		}

		if !isAdmin && req.SpecifyUserID != 0 && req.SpecifyUserID != currentUserID {
			global.Log.Warn("重置密码失败：非管理员不能修改其他用户的密码", zap.Uint("userID", currentUserID))
			return fmt.Errorf("非管理员不能修改其他用户的密码")
		}

		if !isAdmin {
			if err := hash.CompareHashAndPassword(currentUser.Password, req.OldPassword); err != nil {
				global.Log.Warn("重置密码失败：旧密码错误", zap.Uint("userID", currentUserID))
				resetLimiter.RecordFail(ctx, operatorID)
				return fmt.Errorf("旧密码错误，请输入正确的旧密码")
			}
		}

		if isAdmin && targetUserID != currentUserID {
			var exists int64
			if err := tx.Model(&basic.User{}).Where("id = ?", targetUserID).Count(&exists).Error; err != nil {
				global.Log.Error("查询目标用户失败", zap.Error(err), zap.Uint("targetUserID", targetUserID))
				return fmt.Errorf("查询目标用户失败，请稍后重试")
			}
			if exists == 0 {
				global.Log.Warn("重置密码失败：目标用户不存在", zap.Uint("targetUserID", targetUserID))
				return fmt.Errorf("目标用户不存在")
			}
		}

		newPassword, err := hash.GeneratePassword(req.NewPassword)
		if err != nil {
			global.Log.Error("密码加密失败", zap.Error(err))
			return fmt.Errorf("密码加密失败，请稍后重试")
		}

		if err := tx.Model(&basic.User{}).Where("id = ?", targetUserID).Update("password", newPassword).Error; err != nil {
			global.Log.Error("更新密码失败", zap.Error(err), zap.Uint("targetUserID", targetUserID))
			return fmt.Errorf("修改密码失败，请稍后重试")
		}

		global.Log.Info("密码修改成功",
			zap.Uint("operatorID", currentUserID),
			zap.Uint("targetUserID", targetUserID),
			zap.Bool("isAdmin", isAdmin),
		)
		return nil
	})
	if err != nil {
		return err
	}

	resetLimiter.Clear(ctx, operatorID)
	return nil
}

func (s *UserService) OperateRoles(ctx context.Context, req request.UserRoleBindReq) (string, error) {
	var msg string
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		var user basic.User
		if err := tx.First(&user, req.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("用户不存在")
			}
			global.Log.Error("查询用户失败", zap.Error(err), zap.Uint("userID", req.UserID))
			return fmt.Errorf("查询用户失败，请稍后重试")
		}

		var roles []basic.Role
		if err := tx.Where("id IN ?", req.RoleIDs).Find(&roles).Error; err != nil {
			global.Log.Error("查询角色失败", zap.Error(err), zap.Any("roleIDs", req.RoleIDs))
			return fmt.Errorf("查询角色失败，请稍后重试")
		}
		if len(roles) != len(req.RoleIDs) {
			return fmt.Errorf("部分角色不存在，请检查角色ID")
		}

		association := tx.Model(&user).Association("Roles")
		switch req.Operate {
		case "bind":
			if err := association.Append(&roles); err != nil {
				global.Log.Error("角色绑定失败", zap.Error(err), zap.Uint("userID", req.UserID), zap.Any("roleIDs", req.RoleIDs))
				return fmt.Errorf("角色绑定失败，请稍后重试")
			}
			msg = "角色绑定操作成功"
		case "unbind":
			if err := association.Delete(&roles); err != nil {
				global.Log.Error("角色解绑失败", zap.Error(err), zap.Uint("userID", req.UserID), zap.Any("roleIDs", req.RoleIDs))
				return fmt.Errorf("角色解绑失败，请稍后重试")
			}
			msg = "角色解绑操作成功"
		default:
			return fmt.Errorf("不支持的角色操作：%s", req.Operate)
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return msg, nil
}

// List 分页获取用户列表（返回脱敏结构，剔除密码等敏感字段）
func (s *UserService) List(req request.UserListReq) ([]response.UserInfoResp, int64, error) {
	query := global.DB.Model(&basic.User{})

	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Phone != "" {
		query = query.Where("phone LIKE ?", "%"+req.Phone+"%")
	}
	if req.Email != "" {
		query = query.Where("email LIKE ?", "%"+req.Email+"%")
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		global.Log.Error("统计用户总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("查询用户列表失败，请稍后重试")
	}

	var modelUsers []basic.User
	if err := query.Offset(req.Offset()).Limit(req.PageSize).Preload("Roles").Find(&modelUsers).Error; err != nil {
		global.Log.Error("查询用户列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("查询用户列表失败，请稍后重试")
	}

	// 批量聚合当前页用户的粉丝数（一条 GROUP BY 搞定，避免 N+1）
	followerCountMap := batchCountFollowers(modelUsers)

	users := make([]response.UserInfoResp, 0, len(modelUsers))
	for _, user := range modelUsers {
		resp := response.NewUserInfoResp(user)
		resp.FollowersCount = followerCountMap[user.ID]
		users = append(users, *resp)
	}
	return users, total, nil
}

// batchCountFollowers 批量统计一批用户的粉丝数，返回 userID -> 粉丝数 的 map
func batchCountFollowers(users []basic.User) map[uint]int {
	result := make(map[uint]int, len(users))
	if len(users) == 0 {
		return result
	}
	ids := make([]uint, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}

	type followerCountRow struct {
		TargetID uint
		Count    int
	}
	var rows []followerCountRow
	err := global.DB.Model(&basic.UserAction{}).
		Select("target_id, COUNT(*) AS count").
		Where("target_id IN ? AND target_type = ? AND action_type = ?", ids, "author", "follow").
		Group("target_id").
		Scan(&rows).Error
	if err != nil {
		global.Log.Error("批量统计用户粉丝数失败", zap.Error(err))
		return result
	}
	for _, row := range rows {
		result[row.TargetID] = row.Count
	}
	return result
}

// GetInfo 获取用户信息
func (s *UserService) GetInfo(ctx context.Context, id uint) (response.UserInfoResp, error) {
	var user basic.User
	if err := global.DB.Preload("Roles").First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Warn("获取用户信息失败：用户不存在", zap.Uint("userID", id))
			return response.UserInfoResp{}, fmt.Errorf("用户不存在")
		}
		global.Log.Error("查询用户失败", zap.Error(err), zap.Uint("userID", id))
		return response.UserInfoResp{}, fmt.Errorf("查询用户失败，请稍后重试")
	}

	// 查询该用户的粉丝数
	var followersCount int64
	global.DB.Model(&basic.UserAction{}).
		Where("target_id = ? AND target_type = ? AND action_type = ?", id, "author", "follow").
		Count(&followersCount)

	resp := *response.NewUserInfoResp(user)
	resp.FollowersCount = int(followersCount)
	return resp, nil
}

func (s *UserService) FindUsers(req request.UserFindReq) ([]response.UserFindResp, int64, error) {
	query := global.DB.Model(&basic.User{})
	if req.ID != 0 {
		query = query.Where("id = ?", req.ID)
	}
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		global.Log.Error("统计用户总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("查询用户失败，请稍后重试")
	}
	var users []basic.User
	if err := query.Offset(req.Offset()).Limit(req.PageSize).Find(&users).Error; err != nil {
		global.Log.Error("查询用户失败", zap.Error(err))
		return nil, 0, fmt.Errorf("查询用户失败，请稍后重试")
	}

	// 批量聚合当前页用户的粉丝数，避免 N+1
	followerCountMap := batchCountFollowers(users)

	resp := make([]response.UserFindResp, 0, len(users))
	for _, user := range users {
		item := response.NewUserFindResp(user)
		item.FollowersCount = followerCountMap[user.ID]
		resp = append(resp, *item)
	}
	return resp, total, nil
}
