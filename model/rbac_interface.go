package model

type RbacInterface struct {
	ID         uint   `json:"id" gorm:"primaryKey;comment:主键ID"`
	Name       string `json:"name" gorm:"size:64;comment:接口名"`
	Route      string `json:"route" gorm:"size:255;comment:路由"`
	Desc       string `json:"desc" gorm:"size:255;comment:描述"`
	Sort       uint   `json:"sort" gorm:"comment:排序"`
	CategoryID uint   `json:"category_id" gorm:"comment:分类ID"`
	CreateTime uint   `json:"create_time" gorm:"comment:创建时间"`
	UpdateTime uint   `json:"update_time" gorm:"comment:更新时间"`
}

func (RbacInterface) TableName() string {
	return "t_rbac_interface"
}
