package model

type RbacInterfaceCategory struct {
	ID         uint   `json:"id" gorm:"primaryKey;comment:主键ID"`
	Name       string `json:"name" gorm:"size:32;comment:分类名"`
	ParentID   uint   `json:"parent_id" gorm:"comment:父分类ID"`
	Level      uint   `json:"level" gorm:"comment:描述"`
	Sort       uint   `json:"sort" gorm:"comment:排序"`
	CreateTime uint   `json:"create_time" gorm:"comment:创建时间"`
	UpdateTime uint   `json:"update_time" gorm:"comment:更新时间"`
}

func (RbacInterfaceCategory) TableName() string {
	return "t_rbac_interface_category"
}
