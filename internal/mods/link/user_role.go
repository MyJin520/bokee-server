package link

type UserRole struct {
	UserID uint `gorm:"primaryKey;not null;index;comment:用户ID"`
	RoleID uint `gorm:"primaryKey;not null;index;comment:角色ID"`
}

func (UserRole) TableName() string {
	return "sys_user_roles"
}
