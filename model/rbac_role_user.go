package model

type RbacRoleUser struct {
	ID         uint `json:"id" gorm:"primaryKey;comment:主键ID"`
	UserID     uint `json:"user_id" gorm:"comment:用户ID"`
	RoleID     uint `json:"role_id" gorm:"comment:角色ID"`
	CreateTime uint `json:"create_time" gorm:"comment:创建时间"`
}

func (RbacRoleUser) TableName() string {
	return "t_rbac_role_user"
}
