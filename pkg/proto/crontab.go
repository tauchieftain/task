package proto

import (
	"task/model"
)

type CrontabListArgs struct {
	NodeID  uint   `json:"node_id"`
	Keyword string `json:"keyword"`
	Pagination
}

type CrontabListReply struct {
	Total int64            `json:"total"`
	List  []*model.Crontab `json:"list"`
}

type CrontabArgs struct {
	UserID  uint          `json:"user_id"`
	NodeID  uint          `json:"node_id"`
	Crontab model.Crontab `json:"crontab"`
}

type CrontabGetArgs struct {
	NodeID    uint `json:"node_id"`
	CrontabID uint `json:"crontab_id"`
}

type CrontabActionArgs struct {
	UserID     uint   `json:"user_id"`
	NodeID     uint   `json:"node_id"`
	CrontabIDS []uint `json:"crontab_ids"`
}

type CrontabLogListArgs struct {
	NodeID    uint   `json:"node_id"`
	CrontabID uint   `json:"crontab_id"`
	StartTime uint   `json:"start_time"`
	EndTime   uint   `json:"end_time"`
	Status    string `json:"status"`
	Pagination
}

type CrontabLogListReply struct {
	Total int64               `json:"total"`
	List  []*model.CrontabLog `json:"list"`
}
