package model

type RbacRoleNode struct {
	ID         uint `json:"id" gorm:"primaryKey;comment:主键ID"`
	RoleID     uint `json:"role_id" gorm:"comment:角色ID"`
	NodeID     uint `json:"node_id" gorm:"comment:节点ID"`
	CreateTime uint `json:"create_time" gorm:"comment:创建时间"`
}

func (RbacRoleNode) TableName() string {
	return "t_rbac_role_node"
}
