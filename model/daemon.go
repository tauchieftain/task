package model

type Daemon struct {
	ID               uint        `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	Name             string      `json:"name" gorm:"size:100;commit:任务名"`
	Command          string      `json:"command" gorm:"size:255;commit:执行命令"`
	User             string      `json:"user" gorm:"size:30;commit:执行用户"`
	Env              StringSlice `json:"env" gorm:"type:varchar(255);commit:执行环境变量"`
	Dir              string      `json:"dir" gorm:"size:256;commit:执行目录"`
	StartTime        uint        `json:"start_time" gorm:"comment:开启时间"`
	EndTime          uint        `json:"end_time" gorm:"comment:结束时间"`
	FailedRestartNum uint        `json:"failed_restart_num" gorm:"commit:失败重启次数"`
	Status           string      `json:"status" gorm:"size:30;commit:状态"`
	Failed           uint        `json:"failed" gorm:"commit:是否执行失败 0-否 1-是"`
	FailedReason     string      `json:"failed_msg" gorm:"commit:失败原因"`
	FailedNotice     StringSlice `json:"failed_notice" gorm:"type:varchar(255);commit:失败通知方式"`
	DingTalkAddr     StringSlice `json:"ding_talk_addr"  gorm:"type:varchar(1000);commit:钉钉通知地址"`
	CreateUserID     uint        `json:"create_user_id" gorm:"commit:创建人ID"`
	UpdateUserID     uint        `json:"update_user_id" gorm:"commit:更新人ID"`
	CreateTime       uint        `json:"create_time" gorm:"comment:创建时间"`
	UpdateTime       uint        `json:"update_time" gorm:"comment:更新时间"`
}

func (Daemon) TableName() string {
	return "t_daemon"
}
