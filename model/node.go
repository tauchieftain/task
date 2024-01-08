package model

const (
	NodeStatusOk        = 1
	NodeStatusAvailable = "可用"
	NodeStatusDisabled  = "不可用"
)

type Node struct {
	ID              uint   `json:"id" gorm:"primaryKey;commit:主键ID"`
	Name            string `json:"name" gorm:"size:64;commit:节点名"`
	Address         string `json:"address" gorm:"unique;size:100;commit:节点地址"`
	Status          uint   `json:"status" gorm:"commit:节点状态 1-可用 2-不可用"`
	CrontabNum      uint   `json:"crontab_num" gorm:"commit:定时任务数"`
	AuditCrontabNum uint   `json:"audit_crontab_num" gorm:"commit:待审核的定时任务数"`
	FailCrontabNum  uint   `json:"fail_crontab_num" gorm:"commit:执行失败的定时任务数"`
	DaemonNum       uint   `json:"daemon_num" gorm:"commit:常驻任务数"`
	AuditDaemonNum  uint   `json:"audit_daemon_num" gorm:"commit:待审核的常驻任务数"`
	FailDaemonNum   uint   `json:"fail_daemon_num" gorm:"commit:执行失败的常驻任务数"`
	CreateTime      uint   `json:"create_time" gorm:"comment:创建时间"`
	UpdateTime      uint   `json:"update_time" gorm:"comment:更新时间"`
}

func (Node) TableName() string {
	return "t_node"
}
