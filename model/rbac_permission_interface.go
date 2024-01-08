package model

type RbacPermissionInterface struct {
	ID           uint `json:"id" gorm:"primaryKey;comment:主键ID"`
	PermissionID uint `json:"permission_id" gorm:"comment:权限ID"`
	InterfaceID  uint `json:"interface_id" gorm:"comment:接口ID"`
	CreateTime   uint `json:"create_time" gorm:"comment:创建时间"`
}

func (RbacPermissionInterface) TableName() string {
	return "t_rbac_permission_interface"
}
