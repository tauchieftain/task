package service

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"go.uber.org/zap"
	gormlogger "gorm.io/gorm/logger"
	"net/http"
	"task/manage/config"
	"task/model"
	"task/pkg/cas"
	"task/pkg/logger"
	"task/pkg/mrpc"
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
	WSCManage = newWSCManage()
}

func Start() {
	go rpcServe()
	httpServe()
}

func rpcServe() {
	mrpc.ListenAndServer(config.RpcListenAddr(), NewServe())
	Zap.Sugar().Errorln("Rpc Serve is stopped!!!")
}

func httpServe() {
	gin.SetMode(gin.ReleaseMode)
	if config.IsDebug() {
		gin.SetMode(gin.DebugMode)
	}
	e := gin.New()
	e.Use(gin.Recovery())
	setRoute(e)
	_ = e.Run(config.HttpListenAddr())
	Zap.Sugar().Errorln("Http Serve is stopped!!!")
}

func success(ctx *gin.Context, msg string, data interface{}) {
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  msg,
		"data": data,
	})
}

func failed(ctx *gin.Context, code int, msg string) {
	ctx.Set("errCode", code)
	ctx.Set("errMsg", msg)
	ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
		"code": code,
		"msg":  msg,
	})
}

func casService(ctx *gin.Context) *cas.CAS {
	addr := config.CasAddress()
	appId := config.CasAppId()
	token := ctx.GetHeader("token")
	if token == "" {
		body := make(map[string]interface{})
		err := ctx.BindJSON(&body)
		if err == nil {
			if tk, ok := body["token"]; ok {
				token = tk.(string)
			}
		}
	}
	return cas.New(addr, appId, token)
}

func logout(ctx *gin.Context) {
	casService(ctx).Logout()
	success(ctx, "注销成功", nil)
}

func cache(db int) *redis.Client {
	return config.GetRedis(db)
}
