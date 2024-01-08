package model

type RbacRolePermission struct {
	ID           uint `json:"id" gorm:"primaryKey;comment:主键ID"`
	RoleID       uint `json:"role_id" gorm:"comment:角色ID"`
	PermissionID uint `json:"permission_id" gorm:"comment:权限ID"`
	MenuId       uint `json:"menu_id" gorm:"comment:菜单ID"`
	CreateTime   uint `json:"create_time" gorm:"comment:创建时间"`
}

func (RbacRolePermission) TableName() string {
	return "t_rbac_role_permission"
}
