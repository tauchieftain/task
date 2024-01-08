package service

import (
	"context"
	"go.uber.org/zap"
	gormlogger "gorm.io/gorm/logger"
	"log"
	"os"
	"os/signal"
	"syscall"
	"task/client/config"
	"task/model"
	"task/pkg/logger"
	"task/pkg/mrpc"
	"task/pkg/proto"
	"time"
)

var Zap *zap.Logger

func init() {
	l := config.GetLogger()
	logLevel := gormlogger.Warn
	if config.IsDebug() {
		logLevel = gormlogger.Info
	}
	Zap = l.Logger
	gormLogger := logger.NewZapGorm(l, &logger.ZapGormConfig{SlowThreshold: time.Second, LogLevel: logLevel})
	model.InitDB(config.GetDbConfig(), map[string]interface{}{
		"logger": gormLogger,
	})
}

func Start() {
	migrate()
	heartBeat()
	c := newCrontab()
	c.start()
	d := newDaemon()
	d.start()
	go handleSignal(c, d)
	mrpc.ListenAndServer(config.RpcListenAddr(), newServe(), newCrontabServe(c), newDaemonServe(d))
}

func migrate() {
	err := model.Task().AutoMigrate(&model.Crontab{}, &model.CrontabLog{}, &model.Daemon{})
	if err != nil {
		log.Fatalf("Auto Migrate Failed")
	}
}

func handleSignal(c *crontab, d *daemon) {
	s := make(chan os.Signal)
	signal.Notify(s, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-s
	c.killAll()
	d.delAllJob()
}

func heartBeat() {
	addr := config.NodeAddr()
	nodeSync := &proto.NodeSync{
		Address: addr,
		Node: &model.Node{
			Name:            config.NodeName(),
			Address:         addr,
			CrontabNum:      0,
			AuditCrontabNum: 0,
			FailCrontabNum:  0,
			DaemonNum:       0,
			AuditDaemonNum:  0,
			FailDaemonNum:   0,
		},
	}
	var cs []*model.Crontab
	err := model.Task().Order("create_time").Find(&cs).Error
	if err == nil {
		for _, v := range cs {
			nodeSync.Node.CrontabNum += 1
			if v.Status == model.StatusUnaudited {
				nodeSync.Node.AuditCrontabNum += 1
			} else if v.LastExecStatus == model.ExecStatusError && (v.Status == model.StatusTiming || v.Status == model.StatusRunning) {
				nodeSync.Node.FailCrontabNum += 1
			}
		}
	}
	var ds []*model.Daemon
	err = model.Task().Order("create_time").Find(&ds).Error
	if err == nil {
		for _, v := range ds {
			nodeSync.Node.DaemonNum += 1
			if v.Status == model.StatusUnaudited {
				nodeSync.Node.AuditDaemonNum += 1
			} else if v.Failed == 1 {
				nodeSync.Node.FailDaemonNum += 1
			}
		}
	}
	err = mrpc.Call(config.ManageListenAddr(), "Serve.Sync", context.TODO(), nodeSync, nodeSync.Node)
	if err != nil {
		Zap.Sugar().Errorln("节点心跳服务异常," + err.Error())
	}
	time.AfterFunc(time.Duration(config.HeartBeatInterval())*time.Second, heartBeat)
}
