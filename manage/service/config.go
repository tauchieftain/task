package service

import (
	"github.com/gin-gonic/gin"
	"task/model"
	"task/pkg/proto"
	"time"
)

var configService *conf

type conf struct{}

type configList struct {
	Total int64           `json:"total"`
	List  []*model.Config `json:"list"`
}

func (d *conf) list(ctx *gin.Context) {
	var listArgs proto.ConfigListArgs
	if err := ctx.ShouldBindJSON(&listArgs); err != nil {
		failed(ctx, 5000, "请求参数不合法")
		return
	}
	m := model.Task().Model(&model.Config{})
	if listArgs.Type != "" {
		m.Where("type=?", listArgs.Type)
	}
	if listArgs.Name != "" {
		m.Where("name like %?%", listArgs.Name)
	}
	l := &configList{
		Total: 0,
		List:  make([]*model.Config, 0),
	}
	m.Count(&l.Total)
	m.Order("create_time desc").Offset((listArgs.Page - 1) * listArgs.PageSize).Limit(listArgs.PageSize).Find(&l.List)
	success(ctx, "查询成功", l)
}

func (d *conf) add(ctx *gin.Context) {
	var addArgs proto.ConfigAddArgs
	if err := ctx.ShouldBindJSON(&addArgs); err != nil {
		failed(ctx, 5001, "请求参数不合法")
		return
	}
	now := uint(time.Now().Unix())
	c := model.Config{
		Name:       addArgs.Name,
		Type:       addArgs.Type,
		Value:      addArgs.Value,
		CreateTime: now,
		UpdateTime: now,
	}
	err := model.Task().Create(&c).Error
	if err != nil {
		failed(ctx, 5002, "添加失败")
		return
	}
	success(ctx, "添加成功", c.ID)
}

func (d *conf) edit(ctx *gin.Context) {
	var editArgs proto.ConfigEditArgs
	if err := ctx.ShouldBindJSON(&editArgs); err != nil {
		failed(ctx, 5003, "请求参数不合法")
		return
	}
	var c model.Config
	err := model.Task().First(&c, "id=?", editArgs.ID).Error
	if err != nil {
		failed(ctx, 5004, "数据不存在")
		return
	}
	c.Name = editArgs.Name
	c.Type = editArgs.Type
	c.Value = editArgs.Value
	c.UpdateTime = uint(time.Now().Unix())
	err = model.Task().Save(&c).Error
	if err != nil {
		failed(ctx, 5005, "修改失败")
		return
	}
	success(ctx, "修改成功", c.ID)
}

func (d *conf) del(ctx *gin.Context) {
	var delArgs proto.IdArgs
	if err := ctx.ShouldBindJSON(&delArgs); err != nil {
		failed(ctx, 5006, "请求参数不合法")
		return
	}
	model.Task().Delete(&model.Config{}, delArgs.ID)
	success(ctx, "删除成功", nil)
}
