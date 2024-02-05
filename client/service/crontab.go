package service

import (
	"bufio"
	"container/heap"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"task/client/config"
	"task/model"
	pkgcrontab "task/pkg/crontab"
	"task/pkg/helper"
	"task/pkg/mrpc"
	"task/pkg/proto"
	"time"
)

type item = pkgcrontab.PriorityItem

type crontab struct {
	jobs     map[uint]*crontabJob
	onceJobs map[string]*crontabJob
	queue    pkgcrontab.PriorityQueue
	mux      sync.RWMutex
	ready    chan *item
}

type crontabJob struct {
	id           uint
	crontab      *crontab
	once         bool
	uniqueId     string
	userId       uint
	timeExpr     string
	nextExecTime time.Time
	process      *crontabJobProcess
	value        *model.Crontab
}

type crontabJobProcess struct {
	id         uint32
	crontabJob *crontabJob
	ctx        context.Context
	cancel     context.CancelFunc
	execStatus string
	execMsg    string
	execResult string
}

func newCrontab() *crontab {
	return &crontab{
		jobs:     make(map[uint]*crontabJob),
		onceJobs: make(map[string]*crontabJob),
		queue:    make(pkgcrontab.PriorityQueue, 0, 100),
		ready:    make(chan *item, 100),
	}
}

func (c *crontab) start() {
	c.recovery()
	go c.run()
}

func (c *crontab) run() {
	go c.checkReady()
	for i := range c.ready {
		jobID := i.Value.(*crontabJob).id
		if j, ok := c.jobs[jobID]; ok {
			go j.exec()
		}
	}
}

func (c *crontab) checkReady() {
	ticker := time.NewTicker(200 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			if c.queue.Len() == 0 {
				continue
			}
			for {
				c.mux.Lock()
				i := c.getQueueItem()
				c.mux.Unlock()
				if i == nil {
					break
				}
				c.ready <- i
			}
		}
	}
}

func (c *crontab) recovery() {
	var crontabJobs []*model.Crontab
	err := model.Task().Where("status in (?)", []string{model.StatusTiming, model.StatusRunning}).Find(&crontabJobs).Error
	if err == nil {
		var j *crontabJob
		for _, v := range crontabJobs {
			j, err = c.addJob(&crontabJob{
				id:       v.ID,
				timeExpr: v.TimeExpr,
			})
			if err != nil {
				continue
			}
			model.Task().Model(&model.Crontab{}).Where("id=?", v.ID).Updates(map[string]interface{}{
				"status":         model.StatusTiming,
				"next_exec_time": uint(j.nextExecTime.Unix()),
			})
		}
	}
}

func (c *crontab) addJob(j *crontabJob) (*crontabJob, error) {
	c.mux.Lock()
	defer c.mux.Unlock()
	if _, ok := c.jobs[j.id]; !ok {
		j.crontab = c
		c.jobs[j.id] = j
	}
	nt, err := c.getNextExecTime(c.jobs[j.id].timeExpr)
	if err != nil {
		return nil, err
	}
	c.jobs[j.id].nextExecTime = nt
	heap.Push(&c.queue, &item{
		Priority: int(nt.Unix()),
		Value:    c.jobs[j.id],
	})
	return c.jobs[j.id], nil
}

func (c *crontab) getNextExecTime(expr string) (time.Time, error) {
	parse := pkgcrontab.NewParse(expr)
	now := time.Now()
	nt, err := parse.NextExecTime(now)
	if err != nil {
		return now, err
	}
	return nt, nil
}

func (c *crontab) getQueueItem() *item {
	l := c.queue.Len()
	if l == 0 {
		return nil
	}
	i := c.queue[0]
	if i.Priority > int(time.Now().Unix()) {
		return nil
	}
	heap.Remove(&c.queue, 0)
	return i
}

func (c *crontab) removeQueueItem(jobID uint) {
	for k, i := range c.queue {
		if i.Value.(*crontabJob).id == jobID {
			heap.Remove(&c.queue, k)
			break
		}
	}
}

func (c *crontab) addOnceJob(j *crontabJob) (*crontabJob, error) {
	c.mux.Lock()
	j.once = true
	j.crontab = c
	j.uniqueId = helper.UUID()
	if _, ok := c.onceJobs[j.uniqueId]; ok {
		c.mux.Unlock()
		return nil, errors.New("ID重复")
	}
	c.onceJobs[j.uniqueId] = j
	c.mux.Unlock()
	return j, nil
}

func (c *crontab) kill(jobID uint) {
	c.mux.Lock()
	if j, ok := c.jobs[jobID]; ok {
		if j.process != nil {
			j.process.cancel()
		}
		delete(c.jobs, jobID)
		c.removeQueueItem(jobID)
	}
	for uniqueID, j := range c.onceJobs {
		if j.id == jobID {
			if j.process != nil {
				j.process.cancel()
			}
			delete(c.onceJobs, uniqueID)
		}
	}
	c.mux.Unlock()
}

func (c *crontab) killAll() {
	c.mux.Lock()
	if len(c.jobs) > 0 {
		for ID, j := range c.jobs {
			delete(c.jobs, ID)
			j.process.cancel()
		}
	}
	if len(c.onceJobs) > 0 {
		for uniqueID, j := range c.onceJobs {
			delete(c.onceJobs, uniqueID)
			j.process.cancel()
		}
	}
	c.mux.Unlock()
}

func (j *crontabJob) exec() {
	var err error
	if j.once {
		err = model.Task().Take(&j.value, "id=?", j.id).Error
	} else {
		err = model.Task().Take(&j.value, "id=? and status in (?)", j.id, []string{model.StatusTiming, model.StatusRunning}).Error
	}
	if err != nil {
		j.crontab.kill(j.id)
		Zap.Sugar().Errorf("Crontab Job %d is not Exist\n", j.id)
		return
	}
	originStatus := j.value.Status
	model.Task().Model(&j.value).Updates(map[string]interface{}{
		"status": model.StatusRunning,
	})
	sTime := time.Now()
	p := newCrontabJobProcess(j)
	j.process = p
	defer func() {
		if e := recover(); e != nil {
			Zap.Sugar().Errorf("[Crontab]%s exec panic %s \n", j.value.Name, e)
		}
		costTime, _ := strconv.ParseFloat(fmt.Sprintf("%.4f", time.Now().Sub(sTime).Seconds()), 64)
		userId := j.userId
		data := map[string]interface{}{
			"last_exec_time":   sTime.Unix(),
			"last_cost_time":   costTime,
			"last_exec_status": p.execStatus,
			"last_exec_msg":    p.execMsg,
			"status":           originStatus,
		}
		if !j.once {
			var nj *crontabJob
			nj, err = j.crontab.addJob(&crontabJob{
				id:       j.id,
				timeExpr: j.timeExpr,
			})
			if err != nil {
				Zap.Sugar().Errorf("[Crontab]%s add next job Failed, %s", j.value.Name, err.Error())
				data["status"] = model.StatusStopped
				data["next_exec_time"] = 0
			} else {
				data["next_exec_time"] = int(nj.nextExecTime.Unix())
			}
			userId = j.value.UpdateUserID
		}
		model.Task().Model(&j.value).Updates(data)
		model.Task().Create(&model.CrontabLog{
			CrontabID:  j.id,
			Status:     p.execStatus,
			Once:       uint(helper.BoolToInt(j.once)),
			StartTime:  uint(sTime.Unix()),
			EndTime:    uint(time.Now().Unix()),
			CostTime:   costTime,
			Result:     p.execResult,
			ExecUserID: userId,
			CreateTime: uint(time.Now().Unix()),
		})
		if p.execStatus == model.ExecStatusError {
			p.triggerError()
		}
	}()
	p.exec()
}

func newCrontabJobProcess(j *crontabJob) *crontabJobProcess {
	p := &crontabJobProcess{
		crontabJob: j,
	}
	p.ctx, p.cancel = context.WithCancel(context.Background())
	return p
}

func (p *crontabJobProcess) exec() {
	finishChan := make(chan struct{}, 1)
	if p.crontabJob.value.Timeout > 0 {
		time.AfterFunc(time.Duration(p.crontabJob.value.Timeout)*time.Second, func() {
			select {
			case <-finishChan:
				close(finishChan)
			default:
				p.triggerTimeout()
			}
		})
	}
	command := p.crontabJob.value.Command
	args := strings.Split(command, " ")
	cmd := p.getCmd(args[0], args[1:]...)
	var stdout, stderr io.ReadCloser
	var err error
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		p.execStatus = model.ExecStatusError
		p.execMsg = "标准输出初始化失败," + err.Error()
		return
	}
	defer func() {
		_ = stdout.Close()
	}()
	stderr, err = cmd.StderrPipe()
	if err != nil {
		p.execStatus = model.ExecStatusError
		p.execMsg = "标准错误初始化失败," + err.Error()
		return
	}
	defer func() {
		_ = stderr.Close()
	}()
	err = cmd.Start()
	if err != nil {
		p.execStatus = model.ExecStatusError
		p.execMsg = "进程启动失败," + err.Error()
		return
	}
	reader := bufio.NewReader(stdout)
	readerErr := bufio.NewReader(stderr)
	go func() {
		var line []byte
		for {
			line, _ = reader.ReadBytes('\n')
			if len(line) == 0 {
				break
			}
			p.execResult += fmt.Sprintf("%s\n", line)
		}
		for {
			line, _ = readerErr.ReadBytes('\n')
			if len(line) == 0 {
				break
			}
			p.execResult += fmt.Sprintf("%s\n", line)
		}
	}()
	err = cmd.Wait()
	finishChan <- struct{}{}
	if err != nil {
		p.execStatus = model.ExecStatusError
		p.execMsg = "执行失败," + err.Error()
		return
	}
	p.execStatus = model.ExecStatusSuccess
	p.execMsg = "执行成功"
}

func (p *crontabJobProcess) getCmd(name string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(p.ctx, name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Setsid = true
	if p.crontabJob.value.Dir != "" && helper.FileExist(p.crontabJob.value.Dir) {
		cmd.Dir = p.crontabJob.value.Dir
	}
	if len(p.crontabJob.value.Env) > 0 {
		cmd.Env = p.crontabJob.value.Env
	}
	if p.crontabJob.value.User != "" {
		user, err := user.Lookup(p.crontabJob.value.User)
		if err == nil {
			cmd.SysProcAttr = &syscall.SysProcAttr{}
			uid, _ := strconv.Atoi(user.Uid)
			gid, _ := strconv.Atoi(user.Gid)
			cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
		}
	}
	return cmd
}

func (p *crontabJobProcess) triggerTimeout() {
	var reply bool
	for _, trigger := range p.crontabJob.value.TimeoutTrigger {
		switch trigger {
		case model.ForceKill:
			p.crontabJob.crontab.kill(p.crontabJob.id)
			model.Task().Model(&p.crontabJob.value).Updates(map[string]interface{}{
				"status": model.StatusStopped,
			})
			break
		case model.DingTalkNotify:
			title := config.NodeAddr() + "告警：任务超时"
			content := fmt.Sprintf("> ###### 节点: %s 的脚本超时报警：\n> ##### 任务ID：%d\n> ##### 任务名称：%s\n> ##### 超时时间: %d 秒\n> ##### 报警时间：%s", config.NodeAddr(), int(p.crontabJob.id), p.crontabJob.value.Name, p.crontabJob.value.Timeout, time.Now().Format(proto.TimeLayout))
			args := &proto.DingTalkNoticeArgs{
				Address: p.crontabJob.value.DingTalkAddr,
				Body: fmt.Sprintf(
					`{
						"msgtype": "markdown",
						"markdown": {
							"title": "%s",
							"text": "%s"
						}
					}`, title, content),
			}
			err := mrpc.Call(config.ManageListenAddr(), "Serve.DingTalkNotice", context.TODO(), args, &reply)
			if err != nil {
				Zap.Sugar().Errorln("执行超时-钉钉通知失败," + err.Error())
			}
			break
		default:
			return
		}
	}
}

func (p *crontabJobProcess) triggerError() {
	var reply bool
	for _, trigger := range p.crontabJob.value.ErrorTrigger {
		switch trigger {
		case model.ForceKill:
			p.crontabJob.crontab.kill(p.crontabJob.id)
			model.Task().Model(&p.crontabJob.value).Updates(map[string]interface{}{
				"status": model.StatusStopped,
			})
			break
		case model.DingTalkNotify:
			title := config.NodeAddr() + "告警：任务出错"
			content := fmt.Sprintf("> ###### 节点: %s 的任务出错报警：\n> ##### 任务ID：%d\n> ##### 任务名称：%s\n> ##### 报警时间：%s> ##### 失败原因:%s\n", config.NodeAddr(), int(p.crontabJob.id), p.crontabJob.value.Name, time.Now().Format(proto.TimeLayout), p.execMsg)
			args := &proto.DingTalkNoticeArgs{
				Address: p.crontabJob.value.DingTalkAddr,
				Body: fmt.Sprintf(
					`{
						"msgtype": "markdown",
						"markdown": {
							"title": "%s",
							"text": "%s"
						}
					}`, title, content),
			}
			err := mrpc.Call(config.ManageListenAddr(), "Serve.DingTalkNotice", context.TODO(), args, &reply)
			if err != nil {
				Zap.Sugar().Errorln("任务出错-钉钉通知失败," + err.Error())
			}
			break
		default:
			return
		}
	}
}
