package service

import (
	"bufio"
	"errors"
	"gorm.io/gorm"
	"os"
	"regexp"
	"task/client/config"
	"task/model"
	"task/pkg/helper"
	"task/pkg/proto"
	"time"
)

type Serve struct{}

func newServe() *Serve {
	return &Serve{}
}

func (s *Serve) Ping(request *proto.EmptyArgs, response *proto.EmptyReply) error {
	return nil
}

type CrontabServe struct {
	crontab *crontab
}

func newCrontabServe(c *crontab) *CrontabServe {
	return &CrontabServe{
		crontab: c,
	}
}

func (cs *CrontabServe) List(request *proto.CrontabListArgs, response *proto.CrontabListReply) error {
	m := model.Task().Model(&model.Crontab{})
	if request.Keyword != "" {
		txt := "%" + request.Keyword + "%"
		m = m.Where("(name like ? or command like ?)", txt, txt)
	}
	m.Count(&response.Total)
	err := m.Order("update_time desc").Offset((request.Page - 1) * request.PageSize).Limit(request.PageSize).Find(&response.List).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

func (cs *CrontabServe) Add(request proto.CrontabArgs, response *model.Crontab) error {
	now := uint(time.Now().Unix())
	request.Crontab.Status = model.StatusUnaudited
	request.Crontab.CreateUserID = request.UserID
	request.Crontab.UpdateUserID = request.UserID
	request.Crontab.CreateTime = now
	request.Crontab.UpdateTime = now
	defer func() {
		response = &request.Crontab
	}()
	return model.Task().Create(&request.Crontab).Error
}

func (cs *CrontabServe) Get(request proto.CrontabGetArgs, reply *model.Crontab) error {
	return model.Task().First(reply, request.CrontabID).Error
}

func (cs *CrontabServe) Edit(request proto.CrontabArgs, response *model.Crontab) error {
	cs.crontab.kill(request.Crontab.ID)
	defer model.Task().First(response, request.Crontab.ID)
	return model.Task().Model(&model.Crontab{}).Where("id=?", request.Crontab.ID).Updates(map[string]interface{}{
		"name":            request.Crontab.Name,
		"status":          model.StatusUnaudited,
		"next_exec_time":  0,
		"time_expr":       request.Crontab.TimeExpr,
		"timeout":         request.Crontab.Timeout,
		"timeout_trigger": request.Crontab.TimeoutTrigger,
		"error_trigger":   request.Crontab.ErrorTrigger,
		"ding_talk_addr":  request.Crontab.DingTalkAddr,
		"update_user_id":  request.UserID,
		"update_time":     uint(time.Now().Unix()),
	}).Error
}

func (cs *CrontabServe) Audit(request proto.CrontabActionArgs, reply *[]*model.Crontab) error {
	err := model.Task().Model(&model.Crontab{}).Where("id in (?)", request.CrontabIDS).Where("status=?", model.StatusUnaudited).Find(reply).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var IDS []uint
	for k, i := range *reply {
		(*reply)[k].Status = model.StatusOk
		IDS = append(IDS, i.ID)
	}
	return model.Task().Model(&model.Crontab{}).Where("id in (?)", IDS).Update("status", model.StatusOk).Error
}

func (cs *CrontabServe) Start(request proto.CrontabActionArgs, response *[]*model.Crontab) error {
	m := model.Task().Where("id in (?) and status in (?)", request.CrontabIDS, []string{model.StatusOk, model.StatusStopped})
	err := m.Find(response).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var j *crontabJob
	for _, v := range *response {
		j, err = cs.crontab.addJob(&crontabJob{
			id:       v.ID,
			timeExpr: v.TimeExpr,
		})
		if err != nil {
			continue
		}
		model.Task().Model(v).Updates(map[string]interface{}{
			"status":         model.StatusTiming,
			"next_exec_time": uint(j.nextExecTime.Unix()),
		})
	}
	return nil
}

func (cs *CrontabServe) Stop(request proto.CrontabActionArgs, response *[]*model.Crontab) error {
	err := model.Task().Model(&model.Crontab{}).Where("id in (?) and status in (?)", request.CrontabIDS, []string{model.StatusTiming, model.StatusRunning}).Find(response).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var IDS []uint
	for k, v := range *response {
		(*response)[k].Status = model.StatusStopped
		(*response)[k].NextExecTime = 0
		cs.crontab.kill(v.ID)
		IDS = append(IDS, v.ID)
	}
	return model.Task().Model(&model.Crontab{}).Where("id in (?)", IDS).Updates(map[string]interface{}{
		"status":         model.StatusStopped,
		"next_exec_time": 0,
	}).Error
}

func (cs *CrontabServe) Del(request proto.CrontabActionArgs, response *[]*model.Crontab) error {
	m := model.Task().Where("id in (?)", request.CrontabIDS)
	err := m.Find(response).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var IDS []uint
	for _, j := range *response {
		cs.crontab.kill(j.ID)
		IDS = append(IDS, j.ID)
	}
	return model.Task().Where("id in (?)", IDS).Delete(&model.Crontab{}).Error
}

func (cs *CrontabServe) Exec(request proto.CrontabActionArgs, response *[]*model.Crontab) error {
	m := model.Task().Where("id in (?)", request.CrontabIDS)
	err := m.Find(response).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var j *crontabJob
	for _, v := range *response {
		j, err = cs.crontab.addOnceJob(&crontabJob{
			id:     v.ID,
			userId: request.UserID,
		})
		if err == nil {
			go j.exec()
		}
	}
	return nil
}

func (cs *CrontabServe) Kill(request proto.CrontabActionArgs, response *[]*model.Crontab) error {
	m := model.Task().Where("id in (?)", request.CrontabIDS)
	err := m.Find(response).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	for _, j := range *response {
		cs.crontab.kill(j.ID)
	}
	return nil
}

func (cs *CrontabServe) Log(request *proto.CrontabLogListArgs, response *proto.CrontabLogListReply) error {
	m := model.Task().Model(&model.CrontabLog{}).Where("crontab_id=?", request.CrontabID)
	if request.StartTime > 0 {
		m.Where("start_time >= ?", request.StartTime)
	}
	if request.EndTime > 0 {
		m.Where("start_time <= ?", request.EndTime)
	}
	if request.Status != "" {
		m.Where("status = ?", request.Status)
	}
	m.Count(&response.Total)
	err := m.Order("id desc").Offset((request.Page - 1) * request.PageSize).Limit(request.PageSize).Find(&response.List).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

type DaemonServe struct {
	daemon *daemon
}

func newDaemonServe(d *daemon) *DaemonServe {
	return &DaemonServe{
		daemon: d,
	}
}

func (ds *DaemonServe) List(request proto.DaemonListArgs, response *proto.DaemonListReply) error {
	m := model.Task().Model(&model.Daemon{})
	if request.Keyword != "" {
		txt := "%" + request.Keyword + "%"
		m = m.Where("(name like ? or command like ?)", txt, txt)
	}
	m.Count(&response.Total)
	err := m.Order("update_time desc").Offset((request.Page - 1) * request.PageSize).Limit(request.PageSize).Find(&response.List).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

func (ds *DaemonServe) Add(request proto.DaemonArgs, response *model.Daemon) error {
	now := uint(time.Now().Unix())
	request.Daemon.Status = model.StatusUnaudited
	request.Daemon.CreateUserID = request.UserID
	request.Daemon.UpdateUserID = request.UserID
	request.Daemon.CreateTime = now
	request.Daemon.UpdateTime = now
	defer func() {
		response = &request.Daemon
	}()
	return model.Task().Create(&request.Daemon).Error
}

func (ds *DaemonServe) Get(request proto.DaemonGetArgs, reply *model.Daemon) error {
	return model.Task().First(reply, request.DaemonID).Error
}

func (ds *DaemonServe) Edit(request proto.DaemonArgs, response *model.Daemon) error {
	ds.daemon.delJob(request.Daemon.ID)
	defer model.Task().First(response, request.Daemon.ID)
	return model.Task().Model(&model.Daemon{}).Where("id=?", request.Daemon.ID).Updates(map[string]interface{}{
		"name":               request.Daemon.Name,
		"status":             model.StatusUnaudited,
		"start_time":         0,
		"end_time":           0,
		"failed_restart_num": request.Daemon.FailedRestartNum,
		"failed":             0,
		"failed_reason":      "",
		"failed_notice":      request.Daemon.FailedNotice,
		"ding_talk_addr":     request.Daemon.DingTalkAddr,
		"update_user_id":     request.UserID,
		"update_time":        uint(time.Now().Unix()),
	}).Error
}

func (ds *DaemonServe) Audit(request proto.DaemonActionArgs, reply *[]*model.Daemon) error {
	err := model.Task().Where("id in (?)", request.DaemonIDS).Where("status=?", model.StatusUnaudited).Find(reply).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var IDS []uint
	for k, i := range *reply {
		(*reply)[k].Status = model.StatusOk
		IDS = append(IDS, i.ID)
	}
	return model.Task().Model(&model.Daemon{}).Where("id in (?)", IDS).Update("status", model.StatusOk).Error
}

func (ds *DaemonServe) Start(request proto.DaemonActionArgs, response *[]*model.Daemon) error {
	m := model.Task().Where("id in (?) and status in (?)", request.DaemonIDS, []string{model.StatusOk, model.StatusStopped})
	err := m.Find(response).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	for _, v := range *response {
		ds.daemon.addJob(&daemonJob{
			value: v,
		})
	}
	return nil
}

func (ds *DaemonServe) Stop(request proto.DaemonActionArgs, response *[]*model.Daemon) error {
	err := model.Task().Where("id in (?) and status = ?", request.DaemonIDS, model.StatusRunning).Find(response).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var IDS []uint
	for k, v := range *response {
		(*response)[k].Status = model.StatusStopped
		ds.daemon.delJob(v.ID)
		IDS = append(IDS, v.ID)
	}
	return model.Task().Model(&model.Daemon{}).Where("id in (?)", IDS).Update("status", model.StatusStopped).Error
}

func (ds *DaemonServe) Del(request proto.DaemonActionArgs, response *[]*model.Daemon) error {
	m := model.Task().Where("id in (?)", request.DaemonIDS)
	err := m.Find(response).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var IDS []uint
	for _, j := range *response {
		ds.daemon.delJob(j.ID)
		IDS = append(IDS, j.ID)
	}
	return model.Task().Where("id in (?)", IDS).Delete(&model.Daemon{}).Error
}

func (ds *DaemonServe) Log(request proto.DaemonLogArgs, response *proto.DaemonLogReply) error {
	logPath := config.DaemonLogPath(request.DaemonID, request.Date)
	if !helper.FileExist(logPath) {
		return errors.New("日志文件不存在")
	}
	f, err := os.Open(logPath)
	if err != nil {
		return errors.New("无权限访问日志文件")
	}
	defer func() {
		_ = f.Close()
	}()
	response.Offset = request.Offset
	_, _ = f.Seek(int64(request.Offset), 0)
	reader := bufio.NewReader(f)
	var reg *regexp.Regexp
	if request.Keyword != "" {
		reg, err = regexp.Compile(request.Keyword)
		if err != nil {
			return errors.New("关键字不合法")
		}
	}
	for {
		line, _ := reader.ReadBytes('\n')
		if len(line) == 0 {
			break
		}
		response.Offset += uint(len(line))
		if reg == nil || reg.Match(line) {
			response.Content = append(response.Content, string(line))
		}
		if len(response.Content) == int(request.Size) {
			break
		}
	}
	return nil
}
