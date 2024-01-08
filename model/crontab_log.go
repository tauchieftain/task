package model

type CrontabLog struct {
	ID         uint    `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	CrontabID  uint    `json:"crontab_id" gorm:"定时任务ID"`
	Status     string  `json:"status" gorm:"size:30;commit:执行状态"`
	Once       uint    `json:"once" gorm:"commit:是否为手动执行"`
	StartTime  uint    `json:"start_time" gorm:"commit:执行开始时间"`
	EndTime    uint    `json:"end_time" gorm:"commit:执行结束时间"`
	CostTime   float64 `json:"cost_time" gorm:"commit:耗时"`
	Result     string  `json:"result" gorm:"type:varchar(1000);commit:执行结果"`
	ExecUserID uint    `json:"exec_user_id" gorm:"commit:执行人ID"`
	CreateTime uint    `json:"create_time" gorm:"comment:创建时间"`
}

func (CrontabLog) TableName() string {
	return "t_crontab_log"
}
