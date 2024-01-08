package model

type Config struct {
	ID         uint   `json:"id" gorm:"primaryKey;commit:主键ID"`
	Name       string `json:"name" gorm:"size:30;commit:名称"`
	Type       string `json:"type" gorm:"size:30;commit:类型"`
	Value      string `json:"value" gorm:"size:1000;commit:值"`
	CreateTime uint   `json:"create_time" gorm:"comment:创建时间"`
	UpdateTime uint   `json:"update_time" gorm:"comment:更新时间"`
}

func (Config) TableName() string {
	return "t_config"
}
