package service

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"task/model"
	"task/pkg/proto"
	"time"
)

var nodeService *node

type node struct{}

type nodeList struct {
	Total uint          `json:"total"`
	List  []*model.Node `json:"list"`
}

func (n *node) list(ctx *gin.Context) {
	m := model.Task()
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	if user.IsAdmin != 1 && !rbacService.isAdmin(user.ID) {
		nodeIDS := rbacService.getUserNodeIDS(user.ID)
		m.Where("id in (?)", nodeIDS)
	}
	var nodes []*model.Node
	err := m.Order("create_time desc").Find(&nodes).Error
	if err == nil {
		var updateNodeIds []uint
		var msg string
		for _, nd := range nodes {
			if nd.UpdateTime < uint(time.Now().Unix())-uint(proto.NodeAliveTime) && nd.Status == model.NodeStatusOk {
				updateNodeIds = append(updateNodeIds, nd.ID)
				nd.Status = 0
				msg = fmt.Sprintf(model.ContentNodeStatus, time.Now().Format(proto.TimeLayout), nd.Address, model.NodeStatusDisabled)
				model.Task().Create(&model.NodeLog{
					UserID:     1,
					Action:     model.ActionEdit,
					Object:     model.ObjectNode,
					ObjectID:   nd.ID,
					NodeID:     nd.ID,
					Content:    msg,
					CreateTime: nd.UpdateTime + uint(proto.NodeAliveTime),
				})
				WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(nd.ID), msg)
			}
		}
		if len(updateNodeIds) > 0 {
			model.Task().Model(&model.Node{}).Where("id in (?)", updateNodeIds).Update("status", 0)
		}
	}
	nl := &nodeList{
		Total: uint(len(nodes)),
		List:  nodes,
	}
	success(ctx, "查询成功", nl)
}

type nodeLogList struct {
	Total int64              `json:"total"`
	List  []*nodeLogListItem `json:"list"`
}

type nodeLogListItem struct {
	ID       uint   `json:"id"`
	UserName string `json:"user_name"`
	Action   string `json:"action"`
	Object   string `json:"object"`
	ObjectID uint   `json:"object_id"`
	NodeID   uint   `json:"node_id"`
	Content  string `json:"content"`
	Time     string `json:"time"`
}

func (n *node) logList(ctx *gin.Context) {
	var listArgs proto.NodeListArgs
	if err := ctx.ShouldBindJSON(&listArgs); err != nil {
		failed(ctx, 2000, "请求参数不合法")
		return
	}
	l := &nodeLogList{
		Total: 0,
		List:  make([]*nodeLogListItem, 0),
	}
	m := model.Task().Model(&model.NodeLog{})
	if listArgs.Object != "" {
		m.Where("object=?", listArgs.Object)
	}
	if listArgs.Action != "" {
		m.Where("action=?", listArgs.Action)
	}
	if listArgs.NodeID > 0 {
		m.Where("node_id=?", listArgs.NodeID)
	}
	if listArgs.UserID > 0 {
		m.Where("user_id=?", listArgs.UserID)
	}
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	if user.IsAdmin != 1 && !rbacService.isAdmin(user.ID) {
		nodeIDS := rbacService.getUserNodeIDS(user.ID)
		m.Where("node_id in (?)", nodeIDS)
	}
	m.Count(&l.Total)
	var logs []model.NodeLog
	err := m.Order("id desc").Offset((listArgs.Page - 1) * listArgs.PageSize).Limit(listArgs.PageSize).Find(&logs).Error
	if err == nil {
		users := rbacService.getUsers(ctx)
		for _, lg := range logs {
			l.List = append(l.List, &nodeLogListItem{
				ID:       lg.ID,
				UserName: rbacService.getUserName(&users, lg.UserID),
				Action:   lg.Action,
				Object:   lg.Object,
				ObjectID: lg.ObjectID,
				NodeID:   lg.NodeID,
				Content:  lg.Content,
				Time:     time.Unix(int64(lg.CreateTime), 0).Format(proto.TimeLayout),
			})
		}
	}
	success(ctx, "查询成功", l)
}
