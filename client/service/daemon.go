package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"task/client/config"
	"task/model"
	"task/pkg/helper"
	"task/pkg/mrpc"
	"task/pkg/proto"
	"time"
)

type daemon struct {
	jobs  map[uint]*daemonJob
	mux   sync.Mutex
	ready chan *daemonJob
}

type daemonJob struct {
	daemon  *daemon
	value   *model.Daemon
	ctx     context.Context
	cancel  context.CancelFunc
	sTime   time.Time
	logPath string
	logFile *os.File
	errMsg  string
}

func newDaemon() *daemon {
	return &daemon{
		jobs:  make(map[uint]*daemonJob),
		ready: make(chan *daemonJob, 100),
	}
}

func (d *daemon) start() {
	var dj []*model.Daemon
	err := model.Task().Where("status=? and end_time>?", model.StatusStopped, time.Now().Unix()-30).Order("id asc").Find(&dj).Error
	if err == nil {
		for _, j := range dj {
			d.addJob(&daemonJob{value: j})
		}
	}
	go d.run()
}

func (d *daemon) addJob(j *daemonJob) {
	j.daemon = d
	d.mux.Lock()
	d.jobs[j.value.ID] = j
	d.mux.Unlock()
	d.ready <- j
}

func (d *daemon) delJob(ID uint) {
	d.mux.Lock()
	if j, ok := d.jobs[ID]; ok {
		delete(d.jobs, ID)
		d.mux.Unlock()
		j.cancel()
	} else {
		d.mux.Unlock()
	}
}

func (d *daemon) delAllJob() {
	if len(d.jobs) == 0 {
		return
	}
	d.mux.Lock()
	for ID, j := range d.jobs {
		delete(d.jobs, ID)
		j.cancel()
	}
	d.mux.Unlock()
}

func (d *daemon) run() {
	for i := range d.ready {
		d.mux.Lock()
		if j, ok := d.jobs[i.value.ID]; ok {
			j.ctx, j.cancel = context.WithCancel(context.Background())
			go j.exec()
		}
		d.mux.Unlock()
	}
}

func (j *daemonJob) exec() {
	j.sTime = time.Now()
	err := model.Task().Model(j.value).Updates(map[string]interface{}{
		"start_time": uint(time.Now().Unix()),
		"status":     model.StatusRunning,
	}).Error
	if err != nil {
		Zap.Sugar().Errorf("%s update status failed during execution, err:%s \n", j.value.Name, err.Error())
		return
	}
	t := time.NewTicker(1 * time.Second)
	defer func() {
		t.Stop()
		if e := recover(); e != nil {
			Zap.Sugar().Errorf("%s exec panic %s \n", j.value.Name, e)
		}
		j.daemon.delJob(j.value.ID)
		model.Task().Model(j.value).Updates(map[string]interface{}{
			"status":        model.StatusStopped,
			"failed":        1,
			"failed_reason": j.errMsg,
			"end_time":      uint(time.Now().Unix()),
		})
		j.failedNotice()
	}()
	var retryNum uint = 0
	for {
		err = j.launch()
		if err == nil {
			j.errMsg = "脚本错误," + err.Error()
			return
		}
		Zap.Sugar().Errorf("%s exec failed, err:%s \n", j.value.Name, err.Error())
		retryNum++
		select {
		case <-j.ctx.Done():
			j.errMsg = "人工干预停止"
			return
		case <-t.C:
		}
		if retryNum > j.value.FailedRestartNum {
			j.errMsg = "未开启错误重试或已达到最大重试次数"
			return
		}
	}
}

func (j *daemonJob) launch() error {
	var (
		err            error
		stdout, stderr io.ReadCloser
	)
	command := j.value.Command
	args := strings.Split(command, " ")
	cmd := j.getCmd(args[0], args[1:]...)
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer func() {
		_ = stdout.Close()
	}()
	stderr, err = cmd.StderrPipe()
	if err != nil {
		return err
	}
	defer func() {
		_ = stderr.Close()
	}()
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = j.setLogFile()
	if err != nil {
		return err
	}
	defer func() {
		_ = j.logFile.Close()
	}()
	reader := bufio.NewReader(stdout)
	readerErr := bufio.NewReader(stderr)
	go func() {
		var line []byte
		for {
			line, err = reader.ReadBytes('\n')
			if err != nil || err == io.EOF {
				break
			}
			j.writeLog(line)
		}
		for {
			line, _ = readerErr.ReadBytes('\n')
			if err != nil || err == io.EOF {
				break
			}
			j.writeLog(line)
		}
	}()
	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

func (j *daemonJob) setLogFile() error {
	logPath := config.DaemonLogPath(j.value.ID, time.Now().Format(proto.LogPathTimeLayout))
	logFile, err := helper.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_RDWR)
	if err != nil {
		return err
	}
	j.logFile = logFile
	return nil
}

func (j *daemonJob) writeLog(b []byte) {
	logPath := config.DaemonLogPath(j.value.ID, time.Now().Format(proto.LogPathTimeLayout))
	if logPath != j.logPath {
		_ = j.logFile.Close()
		var err error
		j.logFile, err = helper.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_RDWR)
		if err != nil {
			Zap.Sugar().Errorf("%s write log failed, %s \n", j.value.Name, err.Error())
			return
		}
		j.logPath = logPath
	}
	_, _ = j.logFile.Write(b)
}

func (j *daemonJob) getCmd(name string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(j.ctx, name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Setsid = true
	if j.value.Dir != "" && helper.FileExist(j.value.Dir) {
		cmd.Dir = j.value.Dir
	}
	if len(j.value.Env) > 0 {
		cmd.Env = j.value.Env
	}
	if j.value.User != "" {
		user, err := user.Lookup(j.value.User)
		if err == nil {
			cmd.SysProcAttr = &syscall.SysProcAttr{}
			uid, _ := strconv.Atoi(user.Uid)
			gid, _ := strconv.Atoi(user.Gid)
			cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
		}
	}
	return cmd
}

func (j *daemonJob) failedNotice() {
	if len(j.value.FailedNotice) == 0 {
		return
	}
	for _, notice := range j.value.FailedNotice {
		switch notice {
		case model.DingTalkNotify:
			title := config.NodeAddr() + "告警：任务失败"
			content := fmt.Sprintf("> ###### 节点: %s 的任务出错报警：\n> ##### 任务ID：%d\n> ##### 任务名称：%s\n> ##### 报警时间：%s> ##### 失败原因:%s\n", config.NodeAddr(), int(j.value.ID), j.value.Name, time.Now().Format(proto.TimeLayout), j.errMsg)
			args := &proto.DingTalkNoticeArgs{
				Address: j.value.DingTalkAddr,
				Body: fmt.Sprintf(
					`{
						"msgtype": "markdown",
						"markdown": {
							"title": "%s",
							"text": "%s"
						}
					}`, title, content),
			}
			var reply bool
			err := mrpc.Call(config.ManageListenAddr(), "Serve.DingTalkNotice", context.TODO(), args, &reply)
			if err != nil {
				Zap.Sugar().Errorln("任务失败-钉钉通知失败," + err.Error())
			}
			break
		default:
			return
		}
	}
}
