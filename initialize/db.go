package initialize

import (
	"bokee/config"
	"bokee/global"
	"bokee/internal/mods/basic"
	"bokee/pkg/cryptox/hash"
	db "bokee/pkg/gormx"
	"bokee/pkg/randx"
	"fmt"

	"go.uber.org/zap"
)

func initDb(cfg *config.Config) {
	err := db.InitDB(cfg)
	if err != nil {
		panic("数据库初始化失败: " + err.Error())
	}
	migrateTable()
	ensureIMIndexes()
	initRolesAndUser()
}

// 迁移表结构
func migrateTable() {
	err := global.DB.AutoMigrate(
		&basic.User{},
		&basic.Role{},
		&basic.Article{},
		&basic.Files{},
		&basic.UserAction{},
		&basic.Feedback{},
		&basic.FeedbackReply{},
		&basic.Conversation{},
		&basic.IMMessage{},
	)
	if err != nil {
		panic("数据库迁移失败: " + err.Error())
	}
}

// ensureIMIndexes 创建私信模块复合索引（AutoMigrate 只负责建表）
func ensureIMIndexes() {
	indexes := []struct {
		table   string
		name    string
		columns string
	}{
		{"conversations", "idx_conv_a_last", "user_a_id, last_message_at"},
		{"conversations", "idx_conv_b_last", "user_b_id, last_message_at"},
		{"im_messages", "idx_msg_conv", "conversation_id, id"},
		{"im_messages", "idx_msg_conv_sender", "conversation_id, sender_id, id"},
		{"im_messages", "idx_msg_sender", "sender_id, id"},
		{"im_messages", "idx_msg_receiver", "receiver_id, id"},
	}
	for _, idx := range indexes {
		if global.DB.Migrator().HasIndex(idx.table, idx.name) {
			continue
		}
		ddl := fmt.Sprintf("CREATE INDEX %s ON %s (%s)", idx.name, idx.table, idx.columns)
		if err := global.DB.Exec(ddl).Error; err != nil {
			global.Log.Warn("创建私信索引失败", zap.String("index", idx.name), zap.Error(err))
		}
	}
}

// 初始化内置角色与超级管理员：每次启动都会确保内置角色存在，并为历史无角色用户补授普通用户角色
func initRolesAndUser() {
	// 1. 确保内置角色存在（超级管理员、普通用户）
	adminRole := basic.NewRole("超级管理员", uint(global.SuperRoleCode), 1, "normal", "系统内置超级管理员角色")
	if err := global.DB.Where("code = ?", adminRole.Code).FirstOrCreate(adminRole).Error; err != nil {
		panic("初始化超级管理员角色失败: " + err.Error())
	}

	userRole := basic.NewRole("普通用户", uint(global.UserRoleCode), 2, "normal", "注册用户默认角色，拥有账户自助与个人文章管理权限")
	if err := global.DB.Where("code = ?", userRole.Code).FirstOrCreate(userRole).Error; err != nil {
		panic("初始化普通用户角色失败: " + err.Error())
	}

	// 2. 首次启动（用户表为空）时创建超级管理员
	var userCount int64
	if err := global.DB.Model(&basic.User{}).Count(&userCount).Error; err != nil {
		global.Log.Warn(fmt.Sprintf("检查用户表是否为空时出错: %v", err))
		return
	}
	if userCount == 0 {
		global.Log.Info("检测到空用户表，开始初始化超级管理员...")

		defaultPwd := global.Config.System.DefaultAdminPassword
		randomPwd := defaultPwd == ""
		if randomPwd {
			defaultPwd = randx.RandomDigitCode(10)
		}

		password, err := hash.GeneratePassword(defaultPwd)
		if err != nil {
			panic("密码加密失败: " + err.Error())
		}

		newUser := basic.NewUser("superAdmin", password, "", "", "", "")
		newUser.Roles = []basic.Role{*adminRole}

		if err := global.DB.Create(newUser).Error; err != nil {
			panic("初始化超级管理员失败: " + err.Error())
		}

		global.Log.Info(fmt.Sprintf("✅ 初始化完成：已创建超级管理员账户（%s）并关联超级管理员角色", newUser.Name))
		if randomPwd {
			global.Log.Info(fmt.Sprintf("未配置 default-admin-password，已自动生成随机密码: %s，请登录后立即修改", defaultPwd))
		}
		return
	}

	global.Log.Info("用户表非空，跳过超级管理员初始化")
}
