package model

type RbacPermission struct {
	ID         uint   `json:"id" gorm:"primaryKey;comment:主键ID"`
	Name       string `json:"name" gorm:"size:64;comment:权限名"`
	MenuID     uint   `json:"menu_id" gorm:"comment:菜单ID"`
	Code       string `json:"code" gorm:"size:64;comment:权限代码"`
	CreateTime uint   `json:"create_time" gorm:"comment:创建时间"`
	UpdateTime uint   `json:"update_time" gorm:"comment:更新时间"`
}

func (RbacPermission) TableName() string {
	return "t_rbac_permission"
}
