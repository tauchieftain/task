package model

type RbacRole struct {
	ID         uint   `json:"id" gorm:"primaryKey;comment:主键ID"`
	Name       string `json:"name" gorm:"size:30;comment:角色名"`
	CreateTime uint   `json:"create_time" gorm:"comment:创建时间"`
	UpdateTime uint   `json:"update_time" gorm:"comment:更新时间"`
}

func (RbacRole) TableName() string {
	return "t_rbac_role"
}
