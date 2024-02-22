package service

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"task/model"
	"task/pkg/mrpc"
	"task/pkg/proto"
	"time"
)

var cronService *cron

type cron struct{}

type listReply struct {
	Total int64            `json:"total"`
	List  []*listReplyItem `json:"list"`
}

type listReplyItem struct {
	ID             uint    `json:"id"`
	Name           string  `json:"name"`
	Command        string  `json:"command"`
	Dir            string  `json:"dir"`
	LastExecStatus string  `json:"last_exec_status"`
	LastExecMsg    string  `json:"last_exec_msg"`
	LastCostTime   float64 `json:"last_cost_time"`
	LastExecTime   uint    `json:"last_exec_time"`
	NextExecTime   uint    `json:"next_exec_time"`
	TimeExpr       string  `json:"time_expr"`
	Status         string  `json:"status"`
	CreateUser     string  `json:"create_user"`
	CreateTime     uint    `json:"create_time"`
}

func (n *cron) list(ctx *gin.Context) {
	var listArgs proto.CrontabListArgs
	if err := ctx.ShouldBindJSON(&listArgs); err != nil {
		failed(ctx, 3001, "请求参数不合法")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", listArgs.NodeID).Error
	if err != nil {
		failed(ctx, 3002, "节点不存在")
		return
	}
	users := rbacService.getUsers(ctx)
	reply := proto.CrontabListReply{
		Total: 0,
		List:  make([]*model.Crontab, 0),
	}
	err = mrpc.Call(fn.Address, "CrontabServe.List", context.TODO(), listArgs, &reply)
	if err != nil {
		failed(ctx, 3003, "查询失败")
		return
	}
	r := &listReply{
		Total: reply.Total,
		List:  make([]*listReplyItem, 0),
	}
	if len(reply.List) > 0 {
		for _, i := range reply.List {
			r.List = append(r.List, &listReplyItem{
				ID:             i.ID,
				Name:           i.Name,
				Command:        i.Command,
				Dir:            i.Dir,
				LastExecStatus: i.LastExecStatus,
				LastExecMsg:    i.LastExecMsg,
				LastCostTime:   i.LastCostTime,
				LastExecTime:   i.LastExecTime,
				NextExecTime:   i.NextExecTime,
				TimeExpr:       i.TimeExpr,
				Status:         i.Status,
				CreateUser:     rbacService.getUserName(&users, i.CreateUserID),
				CreateTime:     i.CreateTime,
			})
		}
	}
	success(ctx, "查询成功", r)
}

func (n *cron) addCrontab(ctx *gin.Context) {
	var addArgs proto.CrontabArgs
	if err := ctx.ShouldBindJSON(&addArgs); err != nil {
		failed(ctx, 3004, "请求参数不合法")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", addArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 3005, "节点不存在或不可用")
		return
	}
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	addArgs.UserID = user.ID
	reply := model.Crontab{}
	err = mrpc.Call(fn.Address, "CrontabServe.Add", context.TODO(), addArgs, &reply)
	if err != nil {
		failed(ctx, 3006, "添加失败")
		return
	}
	msg := fmt.Sprintf(model.ContentCrontabAdd, time.Now().Format(proto.TimeLayout), user.RealName, fn.Address, reply.Name)
	model.Task().Create(&model.NodeLog{
		UserID:     user.ID,
		Action:     model.ActionAdd,
		Object:     model.ObjectCrontab,
		ObjectID:   reply.ID,
		NodeID:     fn.ID,
		Content:    msg,
		CreateTime: uint(time.Now().Unix()),
	})
	WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(fn.ID), msg)
	success(ctx, "添加成功", reply)
}

func (n *cron) getCrontab(ctx *gin.Context) {
	var getArgs proto.CrontabGetArgs
	if err := ctx.ShouldBindJSON(&getArgs); err != nil {
		failed(ctx, 3007, "请求参数不合法")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", getArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 3008, "节点不存在或不可用")
		return
	}
	reply := model.Crontab{}
	err = mrpc.Call(fn.Address, "CrontabServe.Get", context.TODO(), getArgs, &reply)
	if err != nil {
		failed(ctx, 3009, "查询失败")
		return
	}
	success(ctx, "查询成功", reply)
}

func (n *cron) editCrontab(ctx *gin.Context) {
	var editArgs proto.CrontabArgs
	if err := ctx.ShouldBindJSON(&editArgs); err != nil {
		failed(ctx, 3010, "请求参数不合法")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", editArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 3011, "节点不存在或不可用")
		return
	}
	if editArgs.Crontab.ID == 0 {
		failed(ctx, 3012, "定时任务ID不允许为空")
		return
	}
	var reply model.Crontab
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	editArgs.UserID = user.ID
	err = mrpc.Call(fn.Address, "CrontabServe.Edit", context.TODO(), editArgs, &reply)
	if err != nil {
		failed(ctx, 3014, "修改失败")
		return
	}
	msg := fmt.Sprintf(model.ContentCrontabEdit, time.Now().Format(proto.TimeLayout), user.RealName, fn.Address, reply.Name)
	model.Task().Create(&model.NodeLog{
		UserID:     user.ID,
		Action:     model.ActionEdit,
		Object:     model.ObjectCrontab,
		ObjectID:   reply.ID,
		NodeID:     fn.ID,
		Content:    msg,
		CreateTime: uint(time.Now().Unix()),
	})
	WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(fn.ID), msg)
	success(ctx, "修改成功", reply)
}

func (n *cron) auditCrontab(ctx *gin.Context) {
	var auditArgs proto.CrontabActionArgs
	if err := ctx.ShouldBindJSON(&auditArgs); err != nil {
		failed(ctx, 3015, "请求参数不合法")
		return
	}
	if len(auditArgs.CrontabIDS) == 0 {
		failed(ctx, 3016, "还未选择定时任务")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", auditArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 3017, "节点不存在或不可用")
		return
	}
	reply := make([]*model.Crontab, 0)
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	auditArgs.UserID = user.ID
	err = mrpc.Call(fn.Address, "CrontabServe.Audit", context.TODO(), auditArgs, &reply)
	if err != nil {
		failed(ctx, 3018, "审核失败")
		return
	}
	var msg string
	for _, i := range reply {
		msg = fmt.Sprintf(model.ContentCrontabAudit, time.Now().Format(proto.TimeLayout), user.RealName, fn.Address, i.Name)
		model.Task().Create(&model.NodeLog{
			UserID:     user.ID,
			Action:     model.ActionAudit,
			Object:     model.ObjectCrontab,
			ObjectID:   i.ID,
			NodeID:     fn.ID,
			Content:    msg,
			CreateTime: uint(time.Now().Unix()),
		})
		WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(fn.ID), msg)
	}
	success(ctx, "审核成功", reply)
}

func (n *cron) startCrontab(ctx *gin.Context) {
	var startArgs proto.CrontabActionArgs
	if err := ctx.ShouldBindJSON(&startArgs); err != nil {
		failed(ctx, 3019, "请求参数不合法")
		return
	}
	if len(startArgs.CrontabIDS) == 0 {
		failed(ctx, 3020, "还未选择定时任务")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", startArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 3021, "节点不存在或不可用")
		return
	}
	reply := make([]*model.Crontab, 0)
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	startArgs.UserID = user.ID
	err = mrpc.Call(fn.Address, "CrontabServe.Start", context.TODO(), startArgs, &reply)
	if err != nil {
		failed(ctx, 3022, "操作失败")
		return
	}
	var msg string
	for _, i := range reply {
		msg = fmt.Sprintf(model.ContentCrontabStart, time.Now().Format(proto.TimeLayout), user.RealName, fn.Address, i.Name)
		model.Task().Create(&model.NodeLog{
			UserID:     user.ID,
			Action:     model.ActionStart,
			Object:     model.ObjectCrontab,
			ObjectID:   i.ID,
			NodeID:     fn.ID,
			Content:    msg,
			CreateTime: uint(time.Now().Unix()),
		})
		WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(fn.ID), msg)
	}
	success(ctx, "操作成功", reply)
}

func (n *cron) stopCrontab(ctx *gin.Context) {
	var startArgs proto.CrontabActionArgs
	if err := ctx.ShouldBindJSON(&startArgs); err != nil {
		failed(ctx, 3023, "请求参数不合法")
		return
	}
	if len(startArgs.CrontabIDS) == 0 {
		failed(ctx, 3024, "还未选择定时任务")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", startArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 3025, "节点不存在或不可用")
		return
	}
	reply := make([]*model.Crontab, 0)
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	startArgs.UserID = user.ID
	err = mrpc.Call(fn.Address, "CrontabServe.Stop", context.TODO(), startArgs, &reply)
	if err != nil {
		failed(ctx, 3026, "操作失败")
		return
	}
	var msg string
	for _, i := range reply {
		msg = fmt.Sprintf(model.ContentCrontabStop, time.Now().Format(proto.TimeLayout), user.RealName, fn.Address, i.Name)
		model.Task().Create(&model.NodeLog{
			UserID:     user.ID,
			Action:     model.ActionStop,
			Object:     model.ObjectCrontab,
			ObjectID:   i.ID,
			NodeID:     fn.ID,
			Content:    msg,
			CreateTime: uint(time.Now().Unix()),
		})
		WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(fn.ID), msg)
	}
	success(ctx, "操作成功", reply)
}

func (n *cron) execCrontab(ctx *gin.Context) {
	var startArgs proto.CrontabActionArgs
	if err := ctx.ShouldBindJSON(&startArgs); err != nil {
		failed(ctx, 3027, "请求参数不合法")
		return
	}
	if len(startArgs.CrontabIDS) == 0 {
		failed(ctx, 3028, "还未选择定时任务")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", startArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 3029, "节点不存在或不可用")
		return
	}
	reply := make([]*model.Crontab, 0)
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	startArgs.UserID = user.ID
	err = mrpc.Call(fn.Address, "CrontabServe.Exec", context.TODO(), startArgs, &reply)
	if err != nil {
		failed(ctx, 3030, "操作失败")
		return
	}
	var msg string
	for _, i := range reply {
		msg = fmt.Sprintf(model.ContentCrontabExec, time.Now().Format(proto.TimeLayout), user.RealName, fn.Address, i.Name)
		model.Task().Create(&model.NodeLog{
			UserID:     user.ID,
			Action:     model.ActionExec,
			Object:     model.ObjectCrontab,
			ObjectID:   i.ID,
			NodeID:     fn.ID,
			Content:    msg,
			CreateTime: uint(time.Now().Unix()),
		})
		WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(fn.ID), msg)
	}
	success(ctx, "操作成功", reply)
}

func (n *cron) killCrontab(ctx *gin.Context) {
	var startArgs proto.CrontabActionArgs
	if err := ctx.ShouldBindJSON(&startArgs); err != nil {
		failed(ctx, 3031, "请求参数不合法")
		return
	}
	if len(startArgs.CrontabIDS) == 0 {
		failed(ctx, 3032, "还未选择定时任务")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", startArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 3033, "节点不存在或不可用")
		return
	}
	reply := make([]*model.Crontab, 0)
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	startArgs.UserID = user.ID
	err = mrpc.Call(fn.Address, "CrontabServe.Kill", context.TODO(), startArgs, &reply)
	if err != nil {
		failed(ctx, 3034, "操作失败")
		return
	}
	var msg string
	for _, i := range reply {
		msg = fmt.Sprintf(model.ContentCrontabKill, time.Now().Format(proto.TimeLayout), user.RealName, fn.Address, i.Name)
		model.Task().Create(&model.NodeLog{
			UserID:     user.ID,
			Action:     model.ActionKill,
			Object:     model.ObjectCrontab,
			ObjectID:   i.ID,
			NodeID:     fn.ID,
			Content:    msg,
			CreateTime: uint(time.Now().Unix()),
		})
		WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(fn.ID), msg)
	}
	success(ctx, "操作成功", reply)
}

func (n *cron) delCrontab(ctx *gin.Context) {
	var startArgs proto.CrontabActionArgs
	if err := ctx.ShouldBindJSON(&startArgs); err != nil {
		failed(ctx, 3035, "请求参数不合法")
		return
	}
	if len(startArgs.CrontabIDS) == 0 {
		failed(ctx, 3036, "还未选择定时任务")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", startArgs.NodeID).Error
	if err != nil || fn.Status != model.NodeStatusOk {
		failed(ctx, 3037, "节点不存在或不可用")
		return
	}
	reply := make([]*model.Crontab, 0)
	user := rbacService.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	startArgs.UserID = user.ID
	err = mrpc.Call(fn.Address, "CrontabServe.Del", context.TODO(), startArgs, &reply)
	if err != nil {
		failed(ctx, 3038, "操作失败")
		return
	}
	var msg string
	for _, i := range reply {
		msg = fmt.Sprintf(model.ContentCrontabDel, time.Now().Format(proto.TimeLayout), user.RealName, fn.Address, i.Name)
		model.Task().Create(&model.NodeLog{
			UserID:     user.ID,
			Action:     model.ActionDel,
			Object:     model.ObjectCrontab,
			ObjectID:   i.ID,
			NodeID:     fn.ID,
			Content:    msg,
			CreateTime: uint(time.Now().Unix()),
		})
		WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(fn.ID), msg)
	}
	success(ctx, "操作成功", reply)
}

type crontabLogListReply struct {
	Total int64                      `json:"total"`
	List  []*crontabLogListReplyItem `json:"list"`
}

type crontabLogListReplyItem struct {
	ID        uint    `json:"id"`
	Once      uint    `json:"once"`
	StartTime uint    `json:"start_time"`
	EndTime   uint    `json:"end_time"`
	CostTime  float64 `json:"cost_time"`
	Status    string  `json:"status"`
	Result    string  `json:"result"`
	ExecUser  string  `json:"exec_user"`
}

func (n *cron) log(ctx *gin.Context) {
	var listArgs proto.CrontabLogListArgs
	if err := ctx.ShouldBindJSON(&listArgs); err != nil {
		failed(ctx, 3039, "请求参数不合法")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", listArgs.NodeID).Error
	if err != nil {
		failed(ctx, 3040, "节点不存在")
		return
	}
	users := rbacService.getUsers(ctx)
	reply := proto.CrontabLogListReply{
		Total: 0,
		List:  make([]*model.CrontabLog, 0),
	}
	err = mrpc.Call(fn.Address, "CrontabServe.Log", context.TODO(), listArgs, &reply)
	if err != nil {
		log.Println(err.Error())
		failed(ctx, 3041, "查询失败")
		return
	}
	r := &crontabLogListReply{
		Total: reply.Total,
		List:  make([]*crontabLogListReplyItem, 0),
	}
	if len(reply.List) > 0 {
		for _, i := range reply.List {
			r.List = append(r.List, &crontabLogListReplyItem{
				ID:        i.ID,
				Once:      i.Once,
				StartTime: i.StartTime,
				EndTime:   i.EndTime,
				CostTime:  i.CostTime,
				Status:    i.Status,
				Result:    i.Result,
				ExecUser:  rbacService.getUserName(&users, i.ExecUserID),
			})
		}
	}
	success(ctx, "查询成功", r)
}

func (n *cron) clean(ctx *gin.Context) {
	var cleanArgs proto.CrontabGetArgs
	if err := ctx.ShouldBindJSON(&cleanArgs); err != nil {
		failed(ctx, 3042, "请求参数不合法")
		return
	}
	var fn model.Node
	err := model.Task().First(&fn, "id=?", cleanArgs.NodeID).Error
	if err != nil {
		failed(ctx, 3043, "节点不存在")
		return
	}
	var reply int64
	err = mrpc.Call(fn.Address, "CrontabServe.Clean", context.TODO(), cleanArgs, &reply)
	if err != nil {
		failed(ctx, 3044, "查询失败")
		return
	}
	success(ctx, "清理完毕", reply)
}
