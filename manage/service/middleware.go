package service

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"io"
	"os"
	"task/pkg/helper"
	"task/pkg/proto"
	"time"
)

func request() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		traceId := helper.UUID()
		startTime := time.Now()
		parentTraceId := ctx.GetHeader("X-Ca-TraceId")
		ctx.Set("traceId", traceId)
		ctx.Set("startTime", startTime)
		ctx.Set("parentId", parentTraceId)
		body, _ := ctx.GetRawData()
		ctx.Set("params", string(body))
		ctx.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		ctx.Next()
		endTime := time.Now()
		Zap.With([]zap.Field{
			zap.String("clientIp", ctx.ClientIP()),
			zap.String("host", ctx.Request.Host),
			zap.String("url", ctx.Request.RequestURI),
			zap.String("params", ctx.GetString("params")),
			zap.Int("processId", os.Getpid()),
			zap.String("start", startTime.Format(proto.TimeMicroLayout)),
			zap.String("end", endTime.Format(proto.TimeMicroLayout)),
			zap.Float64("duration", float64(endTime.Sub(startTime).Nanoseconds()/1e4)/100.0),
			zap.String("traceId", traceId),
			zap.String("parentId", parentTraceId),
			zap.Int("errCode", ctx.GetInt("errCode")),
			zap.String("errMsg", ctx.GetString("errMsg")),
		}...).Info("")
	}
}

func auth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, err := casService(ctx).CheckToken(ctx)
		if err != nil {
			failed(ctx, 1000, "Forbidden Access!!")
			return
		}
		ctx.Set("user", user)
		if user.IsAdmin == 0 && !rbacService.isAdmin(user.ID) && ctx.Request.RequestURI != "/rbac/user/current" {
			userPermissions, _ := rbacService.userPermissions(user.ID, TypeWeb)
			if userPermissions == nil {
				failed(ctx, 1000, "Forbidden Access!!")
				return
			}
			interfacePermissions, _ := rbacService.getPermissionsByInterfaceRoute(ctx.Request.RequestURI)
			if interfacePermissions == nil {
				failed(ctx, 1000, "Forbidden Access!!")
				return
			}
			canAccess := false
			for _, ip := range interfacePermissions {
				for _, up := range userPermissions {
					if ip.ID == up.ID {
						canAccess = true
						break
					}
				}
			}
			if !canAccess {
				failed(ctx, 1000, "Forbidden Access!!")
				return
			}
		}
		ctx.Next()
	}
}
