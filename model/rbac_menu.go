package model

type RbacMenu struct {
	ID         uint   `json:"id" gorm:"primaryKey;comment:主键ID"`
	Name       string `json:"name" gorm:"size:30;comment:菜单名"`
	Route      string `json:"router" gorm:"size:150;comment:前端路由"`
	Icon       string `json:"icon" gorm:"size:255;comment:菜单图标"`
	ParentID   uint   `json:"parent_id" gorm:"comment:父菜单ID"`
	Level      uint   `json:"level" gorm:"comment:菜单对应层级"`
	Type       uint   `json:"type" gorm:"comment:菜单类型 0-通用 1-Web端 2-App端"`
	Sort       uint   `json:"sort" gorm:"comment:排序"`
	Status     uint   `json:"status" gorm:"comment:状态 0-禁用 1-正常"`
	CreateTime uint   `json:"create_time" gorm:"comment:创建时间"`
	UpdateTime uint   `json:"update_time" gorm:"comment:更新时间"`
}

func (RbacMenu) TableName() string {
	return "t_rbac_menu"
}
