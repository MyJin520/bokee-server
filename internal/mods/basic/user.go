package basic

type User struct {
	BaseModel
	Name     string `gorm:"size:10;not null;index;comment:用户名"`
	Password string `gorm:"size:64;not null;comment:密码"`
	Phone    string `gorm:"size:11;comment:手机号"`
	Email    string `gorm:"size:64;comment:邮箱"`
	Status   string `gorm:"size:10;not null;default:normal;comment:账号状态"`
	Avatar   string `gorm:"size:255;comment:头像地址"`
	Roles    []Role `gorm:"many2many:sys_user_roles;constraint:OnDelete:CASCADE"`
}

func NewUser(name, password, phone, email, status, avatar string) *User {
	return &User{
		Name:     name,
		Password: password,
		Phone:    phone,
		Email:    email,
		Status:   status,
		Avatar:   avatar,
	}
}

func (User) TableName() string {
	return "users"
}
