package model

const (
	ObjectNode          string = "Node"
	ObjectCrontab       string = "Crontab"
	ObjectDaemon        string = "Daemon"
	ActionAdd           string = "Add"
	ActionEdit          string = "Edit"
	ActionDel           string = "Del"
	ActionAudit         string = "Audit"
	ActionStart         string = "Start"
	ActionStop          string = "Stop"
	ActionExec          string = "Exec"
	ActionKill          string = "Kill"
	ContentNodeDiscover string = "%v, 发现了新的节点 %v"
	ContentNodeStatus   string = "%v, 节点 %v 的状态变为 %v"
	ContentCrontabAdd   string = "%v, 用户 %v 在节点 %v 上添加了定时任务 %v"
	ContentCrontabEdit  string = "%v, 用户 %v 在节点 %v 上修改了定时任务 %v"
	ContentCrontabDel   string = "%v, 用户 %v 在节点 %v 上删除了定时任务 %v"
	ContentCrontabAudit string = "%v, 用户 %v 在节点 %v 上审核通过了定时任务 %v"
	ContentCrontabStart string = "%v, 用户 %v 在节点 %v 上开启了定时任务 %v"
	ContentCrontabStop  string = "%v, 用户 %v 在节点 %v 上停止了定时任务 %v"
	ContentCrontabExec  string = "%v, 用户 %v 在节点 %v 上手动执行了定时任务 %v"
	ContentCrontabKill  string = "%v, 用户 %v 在节点 %v 上强杀了定时任务 %v"
	ContentDaemonAdd    string = "%v, 用户 %v 在节点 %v 上添加了常驻任务 %v"
	ContentDaemonEdit   string = "%v, 用户 %v 在节点 %v 上修改了常驻任务 %v"
	ContentDaemonDel    string = "%v, 用户 %v 在节点 %v 上删除了常驻任务 %v"
	ContentDaemonAudit  string = "%v, 用户 %v 在节点 %v 上审核通过了常驻任务 %v"
	ContentDaemonStart  string = "%v, 用户 %v 在节点 %v 上开启了常驻任务 %v"
	ContentDaemonStop   string = "%v, 用户 %v 在节点 %v 上停止了常驻任务 %v"
)

type NodeLog struct {
	ID         uint   `json:"id" gorm:"primaryKey;comment:主键ID"`
	UserID     uint   `json:"user_id" gorm:"index;commit:操作人ID"`
	Action     string `json:"action" gorm:"commit:操作方法"`
	Object     string `json:"object" gorm:"commit:操作对象"`
	ObjectID   uint   `json:"object_id" gorm:"commit:操作对象ID"`
	NodeID     uint   `json:"node_id" gorm:"commit:节点ID"`
	Content    string `json:"content" gorm:"type:varchar(256);commit:操作内容"`
	CreateTime uint   `json:"create_time" gorm:"comment:创建时间"`
}

func (NodeLog) TableName() string {
	return "t_node_log"
}
