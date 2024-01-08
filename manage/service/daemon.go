package service

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"math"
	"task/model"
	"task/pkg/helper"
	"task/pkg/mrpc"
	"task/pkg/proto"
	"time"
)

var daemonService *daemon

type daemon struct{}

type daemonListReply struct {
	Total int64                  `json:"total"`
	List  []*daemonListReplyItem `json:"list"`
}

type daemonListReplyItem struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Command    string `json:"command"`
	Dir        string `json:"dir"`
	RunStatus  string `json:"run_status"`
	Status     string `json:"status"`
	CreateUser string `json:"create_user"`
	CreateTime uint   `json:"create_time"`
}

func (d *daemon) list(ctx *gin.Context) {
	var listArgs proto.DaemonListArgs
	if err := ctx.ShouldBindJSON(&listArgs); err != nil {
		failed(ctx, 4001, "请求参数不合法")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", listArgs.NodeID).Error
	if err != nil {
		failed(ctx, 4002, "节点不存在")
		return
	}
	users := rbacService.getUsers(ctx)
	reply := proto.DaemonListReply{
		Total: 0,
		List:  make([]*model.Daemon, 0),
	}
	err = mrpc.Call(fn.Address, "DaemonServe.List", context.TODO(), listArgs, &reply)
	if err != nil {
		failed(ctx, 4003, "查询失败")
		return
	}
	r := &daemonListReply{
		Total: reply.Total,
		List:  make([]*daemonListReplyItem, 0),
	}
	if len(reply.List) > 0 {
		var runStatus string
		for _, i := range reply.List {
			if i.Status == model.StatusUnaudited {
				runStatus = "待审核"
			} else if i.Status == model.StatusOk {
				runStatus = "审核通过"
			} else if i.Status == model.StatusRunning {
				s := int64(math.Round(time.Now().Sub(time.Unix(int64(i.StartTime), 0)).Seconds()))
				runStatus = fmt.Sprintf("正在运行(时长：%s)", helper.HumanTime(s))
			} else if i.Status == model.StatusStopped {
				runStatus = fmt.Sprintf("已停止(原因：%s)", i.FailedReason)
			}
			r.List = append(r.List, &daemonListReplyItem{
				ID:         i.ID,
				Name:       i.Name,
				Command:    i.Command,
				Dir:        i.Dir,
				RunStatus:  runStatus,
				Status:     i.Status,
				CreateUser: rbacService.getUserName(&users, i.CreateUserID),
				CreateTime: i.CreateTime,
			})
		}
	}
	success(ctx, "查询成功", r)
}

func (d *daemon) addDaemon(ctx *gin.Context) {
	var addArgs proto.DaemonArgs
	if err := ctx.ShouldBindJSON(&addArgs); err != nil {
		failed(ctx, 4004, "请求参数不合法")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", addArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 4005, "节点不存在或不可用")
		return
	}
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	addArgs.UserID = user.ID
	reply := model.Daemon{}
	err = mrpc.Call(fn.Address, "DaemonServe.Add", context.TODO(), addArgs, &reply)
	if err != nil {
		failed(ctx, 4006, "添加失败")
		return
	}
	msg := fmt.Sprintf(model.ContentDaemonAdd, time.Now().Format(proto.TimeLayout), user.RealName, fn.Address, reply.Name)
	model.Task().Create(&model.NodeLog{
		UserID:     user.ID,
		Action:     model.ActionAdd,
		Object:     model.ObjectDaemon,
		ObjectID:   reply.ID,
		NodeID:     fn.ID,
		Content:    msg,
		CreateTime: uint(time.Now().Unix()),
	})
	WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(fn.ID), msg)
	success(ctx, "添加成功", reply)
}

func (d *daemon) getDaemon(ctx *gin.Context) {
	var getArgs proto.DaemonGetArgs
	if err := ctx.ShouldBindJSON(&getArgs); err != nil {
		failed(ctx, 4007, "请求参数不合法")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", getArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 4008, "节点不存在或不可用")
		return
	}
	reply := model.Daemon{}
	err = mrpc.Call(fn.Address, "DaemonServe.Get", context.TODO(), getArgs, &reply)
	if err != nil {
		failed(ctx, 4009, "查询失败")
		return
	}
	success(ctx, "查询成功", reply)
}

func (d *daemon) editDaemon(ctx *gin.Context) {
	var editArgs proto.DaemonArgs
	if err := ctx.ShouldBindJSON(&editArgs); err != nil {
		failed(ctx, 4010, "请求参数不合法")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", editArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 4011, "节点不存在或不可用")
		return
	}
	if editArgs.Daemon.ID == 0 {
		failed(ctx, 4012, "定时任务ID不允许为空")
		return
	}
	var reply model.Daemon
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	editArgs.UserID = user.ID
	err = mrpc.Call(fn.Address, "DaemonServe.Edit", context.TODO(), editArgs, &reply)
	if err != nil {
		failed(ctx, 4013, "修改失败")
		return
	}
	msg := fmt.Sprintf(model.ContentDaemonEdit, time.Now().Format(proto.TimeLayout), user.RealName, fn.Address, reply.Name)
	model.Task().Create(&model.NodeLog{
		UserID:     user.ID,
		Action:     model.ActionEdit,
		Object:     model.ObjectDaemon,
		ObjectID:   reply.ID,
		NodeID:     fn.ID,
		Content:    msg,
		CreateTime: uint(time.Now().Unix()),
	})
	WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(fn.ID), msg)
	success(ctx, "修改成功", reply)
}

func (d *daemon) auditDaemon(ctx *gin.Context) {
	var auditArgs proto.DaemonActionArgs
	if err := ctx.ShouldBindJSON(&auditArgs); err != nil {
		failed(ctx, 4015, "请求参数不合法")
		return
	}
	if len(auditArgs.DaemonIDS) == 0 {
		failed(ctx, 4016, "还未选择定时任务")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", auditArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 4017, "节点不存在或不可用")
		return
	}
	reply := make([]*model.Daemon, 0)
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	auditArgs.UserID = user.ID
	err = mrpc.Call(fn.Address, "DaemonServe.Audit", context.TODO(), auditArgs, &reply)
	if err != nil {
		failed(ctx, 4018, "审核失败")
		return
	}
	var msg string
	for _, i := range reply {
		msg = fmt.Sprintf(model.ContentDaemonAudit, time.Now().Format(proto.TimeLayout), user.RealName, fn.Address, i.Name)
		model.Task().Create(&model.NodeLog{
			UserID:     user.ID,
			Action:     model.ActionAudit,
			Object:     model.ObjectDaemon,
			ObjectID:   i.ID,
			NodeID:     fn.ID,
			Content:    msg,
			CreateTime: uint(time.Now().Unix()),
		})
		WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(fn.ID), msg)
	}
	success(ctx, "审核成功", reply)
}

func (d *daemon) startDaemon(ctx *gin.Context) {
	var startArgs proto.DaemonActionArgs
	if err := ctx.ShouldBindJSON(&startArgs); err != nil {
		failed(ctx, 4019, "请求参数不合法")
		return
	}
	if len(startArgs.DaemonIDS) == 0 {
		failed(ctx, 4020, "还未选择定时任务")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", startArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 4021, "节点不存在或不可用")
		return
	}
	reply := make([]*model.Daemon, 0)
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	startArgs.UserID = user.ID
	err = mrpc.Call(fn.Address, "DaemonServe.Start", context.TODO(), startArgs, &reply)
	if err != nil {
		failed(ctx, 4022, "操作失败")
		return
	}
	var msg string
	for _, i := range reply {
		msg = fmt.Sprintf(model.ContentDaemonStart, time.Now().Format(proto.TimeLayout), user.RealName, fn.Address, i.Name)
		model.Task().Create(&model.NodeLog{
			UserID:     user.ID,
			Action:     model.ActionStart,
			Object:     model.ObjectDaemon,
			ObjectID:   i.ID,
			NodeID:     fn.ID,
			Content:    msg,
			CreateTime: uint(time.Now().Unix()),
		})
		WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(fn.ID), msg)
	}
	success(ctx, "操作成功", reply)
}

func (d *daemon) stopDaemon(ctx *gin.Context) {
	var startArgs proto.DaemonActionArgs
	if err := ctx.ShouldBindJSON(&startArgs); err != nil {
		failed(ctx, 4023, "请求参数不合法")
		return
	}
	if len(startArgs.DaemonIDS) == 0 {
		failed(ctx, 4024, "还未选择定时任务")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", startArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 4025, "节点不存在或不可用")
		return
	}
	reply := make([]*model.Daemon, 0)
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	startArgs.UserID = user.ID
	err = mrpc.Call(fn.Address, "DaemonServe.Stop", context.TODO(), startArgs, &reply)
	if err != nil {
		failed(ctx, 4026, "操作失败,"+err.Error())
		return
	}
	var msg string
	for _, i := range reply {
		msg = fmt.Sprintf(model.ContentDaemonStop, time.Now().Format(proto.TimeLayout), user.RealName, fn.Address, i.Name)
		model.Task().Create(&model.NodeLog{
			UserID:     user.ID,
			Action:     model.ActionStop,
			Object:     model.ObjectDaemon,
			ObjectID:   i.ID,
			NodeID:     fn.ID,
			Content:    msg,
			CreateTime: uint(time.Now().Unix()),
		})
		WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(fn.ID), msg)
	}
	success(ctx, "操作成功", reply)
}

func (d *daemon) delDaemon(ctx *gin.Context) {
	var delArgs proto.DaemonActionArgs
	if err := ctx.ShouldBindJSON(&delArgs); err != nil {
		failed(ctx, 4027, "请求参数不合法")
		return
	}
	if len(delArgs.DaemonIDS) == 0 {
		failed(ctx, 4028, "还未选择定时任务")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", delArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 4029, "节点不存在或不可用")
		return
	}
	reply := make([]*model.Daemon, 0)
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	delArgs.UserID = user.ID
	err = mrpc.Call(fn.Address, "DaemonServe.Del", context.TODO(), delArgs, &reply)
	if err != nil {
		failed(ctx, 4030, "操作失败")
		return
	}
	var msg string
	for _, i := range reply {
		msg = fmt.Sprintf(model.ContentDaemonDel, time.Now().Format(proto.TimeLayout), user.RealName, fn.Address, i.Name)
		model.Task().Create(&model.NodeLog{
			UserID:     user.ID,
			Action:     model.ActionDel,
			Object:     model.ObjectDaemon,
			ObjectID:   i.ID,
			NodeID:     fn.ID,
			Content:    msg,
			CreateTime: uint(time.Now().Unix()),
		})
		WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(fn.ID), msg)
	}
	success(ctx, "操作成功", reply)
}

func (d *daemon) log(ctx *gin.Context) {
	var logArgs proto.DaemonLogArgs
	if err := ctx.ShouldBindJSON(&logArgs); err != nil {
		failed(ctx, 4031, "请求参数不合法")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", logArgs.NodeID).Error
	if err != nil {
		failed(ctx, 4032, "节点不存在")
		return
	}
	var reply proto.DaemonLogReply
	err = mrpc.Call(fn.Address, "DaemonServe.Log", context.TODO(), logArgs, &reply)
	if err != nil {
		failed(ctx, 4033, "日志不存在")
		return
	}
	success(ctx, "查询成功", reply)
}
