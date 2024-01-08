package proto

import (
	"task/model"
)

type DaemonListArgs struct {
	NodeID  uint   `json:"node_id"`
	Keyword string `json:"keyword"`
	Pagination
}

type DaemonListReply struct {
	Total int64           `json:"total"`
	List  []*model.Daemon `json:"list"`
}

type DaemonArgs struct {
	UserID uint         `json:"user_id"`
	NodeID uint         `json:"node_id"`
	Daemon model.Daemon `json:"daemon"`
}

type DaemonGetArgs struct {
	NodeID   uint `json:"node_id"`
	DaemonID uint `json:"daemon_id"`
}

type DaemonActionArgs struct {
	UserID    uint   `json:"user_id"`
	NodeID    uint   `json:"node_id"`
	DaemonIDS []uint `json:"daemon_ids"`
}

type DaemonLogArgs struct {
	NodeID   uint   `json:"node_id"`
	DaemonID uint   `json:"daemon_id"`
	Date     string `json:"date"`
	Keyword  string `json:"keyword"`
	Offset   uint   `json:"offset"`
	Size     uint   `json:"size"`
}

type DaemonLogReply struct {
	Offset  uint     `json:"offset"`
	Content []string `json:"content"`
}
