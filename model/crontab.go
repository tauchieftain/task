package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

const (
	StatusUnaudited  string = "Unaudited"
	StatusOk         string = "Ok"
	StatusTiming     string = "Timing"
	StatusRunning    string = "Running"
	StatusStopped    string = "Stopped"
	StatusRestarting string = "Restarting"
)

const (
	ExecStatusError   string = "Error"
	ExecStatusSuccess string = "Success"
	ExecStatusTimeout string = "Timeout"
)

const (
	ForceKill      string = "ForceKill"
	DingTalkNotify string = "DingTalkNotify"
)

type Crontab struct {
	ID             uint        `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	Name           string      `json:"name" gorm:"size:100;commit:任务名"`
	Command        string      `json:"command" gorm:"size:255;commit:执行命令"`
	User           string      `json:"user" gorm:"size:30;commit:执行用户"`
	Env            StringSlice `json:"env" gorm:"type:varchar(255);commit:执行环境变量"`
	Dir            string      `json:"dir" gorm:"size:256;commit:执行目录"`
	Timeout        uint        `json:"timeout" gorm:"执行超时时间"`
	LastExecStatus string      `json:"last_exec_status" gorm:"size:30;commit:上次执行状态"`
	LastExecMsg    string      `json:"last_exec_msg" gorm:"type:varchar(1000);commit:上次执行信息"`
	LastCostTime   float64     `json:"last_cost_time" gorm:"commit:上次执行耗时"`
	LastExecTime   uint        `json:"last_exec_time" gorm:"commit:上次执行时间"`
	NextExecTime   uint        `json:"next_exec_time" gorm:"commit:下次执行时间"`
	TimeExpr       string      `json:"time_expr" gorm:"type:varchar(100);commit:cron表达式"`
	Status         string      `json:"status" gorm:"size:30;commit:状态"`
	TimeoutTrigger StringSlice `json:"timeout_trigger" gorm:"type:varchar(255);commit:超时触发方式"`
	ErrorTrigger   StringSlice `json:"error_trigger" gorm:"type:varchar(255);commit:错误触发方式"`
	DingTalkAddr   StringSlice `json:"ding_talk_addr"  gorm:"type:varchar(1000);commit:钉钉通知地址"`
	CreateUserID   uint        `json:"create_user_id" gorm:"commit:创建人ID"`
	UpdateUserID   uint        `json:"update_user_id" gorm:"commit:更新人ID"`
	CreateTime     uint        `json:"create_time" gorm:"comment:创建时间"`
	UpdateTime     uint        `json:"update_time" gorm:"comment:更新时间"`
}

func (Crontab) TableName() string {
	return "t_crontab"
}

type StringSlice []string

func (s *StringSlice) Scan(v interface{}) error {

	switch val := v.(type) {
	case string:
		return json.Unmarshal([]byte(val), s)
	case []byte:
		return json.Unmarshal(val, s)
	default:
		return errors.New("not support")
	}
}

func (s StringSlice) MarshalJSON() ([]byte, error) {
	if s == nil {
		s = make(StringSlice, 0)
	}
	return json.Marshal([]string(s))
}

func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		s = make(StringSlice, 0)
	}
	bts, err := json.Marshal(s)
	return string(bts), err
}
