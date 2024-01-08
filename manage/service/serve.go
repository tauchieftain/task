package service

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"task/model"
	"task/pkg/proto"
	"time"
)

type Serve struct {
}

func NewServe() *Serve {
	return &Serve{}
}

func (s *Serve) Sync(request *proto.NodeSync, response *model.Node) error {
	err := model.Task().First(response, "address=?", request.Address).Error
	now := uint(time.Now().Unix())
	if err == nil {
		if response.Status == 0 {
			defer func() {
				msg := fmt.Sprintf(model.ContentNodeStatus, time.Now().Format(proto.TimeLayout), response.Address, model.NodeStatusAvailable)
				model.Task().Create(&model.NodeLog{
					UserID:     1,
					Action:     model.ActionEdit,
					Object:     model.ObjectNode,
					ObjectID:   response.ID,
					NodeID:     response.ID,
					Content:    msg,
					CreateTime: now,
				})
				WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(response.ID), msg)
			}()
		}
		response.Name = request.Node.Name
		response.Status = 1
		response.CrontabNum = request.Node.CrontabNum
		response.AuditCrontabNum = request.Node.AuditCrontabNum
		response.FailCrontabNum = request.Node.FailCrontabNum
		response.DaemonNum = request.Node.DaemonNum
		response.AuditDaemonNum = request.Node.AuditDaemonNum
		response.FailDaemonNum = request.Node.FailDaemonNum
		response.UpdateTime = now
		return model.Task().Save(response).Error
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Name = request.Node.Name
		response.Address = request.Address
		response.Status = 1
		response.CrontabNum = request.Node.CrontabNum
		response.AuditCrontabNum = request.Node.AuditCrontabNum
		response.FailCrontabNum = request.Node.FailCrontabNum
		response.DaemonNum = request.Node.DaemonNum
		response.AuditDaemonNum = request.Node.AuditDaemonNum
		response.FailDaemonNum = request.Node.FailDaemonNum
		response.CreateTime = now
		response.UpdateTime = now
		defer func(n *model.Node) {
			msg := fmt.Sprintf(model.ContentNodeDiscover, time.Unix(int64(n.CreateTime), 0).Format(proto.TimeLayout), n.Address)
			model.Task().Create(&model.NodeLog{
				UserID:     1,
				Action:     model.ActionAdd,
				Object:     model.ObjectNode,
				ObjectID:   n.ID,
				NodeID:     n.ID,
				Content:    msg,
				CreateTime: n.CreateTime,
			})
			WSCManage.pushWSMessage(rbacService.getNodeRoleIDS(n.ID), msg)
		}(response)
		return model.Task().Create(response).Error
	} else {
		return errors.New("db connect disable")
	}
}

func (s *Serve) Ping(request *proto.EmptyArgs, response *proto.EmptyReply) error {
	return nil
}

func (s *Serve) DingTalkNotice(request proto.DingTalkNoticeArgs, response *bool) error {
	var i = 0
	for _, addr := range request.Address {
		res, err := http.Post(addr, "application/json; charset=utf-8", strings.NewReader(request.Body))
		_ = res.Body.Close()
		if err != nil {
			Zap.Sugar().Errorln("DingTalkNotice Failed, " + err.Error())
			continue
		}
		i++
	}
	if i > 0 {
		*response = true
	}
	return nil
}
