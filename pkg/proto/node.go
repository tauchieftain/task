package proto

import "task/model"

type NodeSync struct {
	Address string `json:"address"`
	Node    *model.Node
}

type NodeListArgs struct {
	UserID uint   `json:"user_id"`
	Action string `json:"action"`
	Object string `json:"object"`
	NodeID uint   `json:"node_id"`
	Pagination
}
