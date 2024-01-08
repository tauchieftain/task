package proto

type InterfaceAddArgs struct {
	Name       string `json:"name"`
	Route      string `json:"route"`
	Desc       string `json:"desc"`
	Sort       uint   `json:"sort"`
	CategoryId uint   `json:"category_id"`
}

type InterfaceEditArgs struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Route      string `json:"route"`
	Desc       string `json:"desc"`
	Sort       uint   `json:"sort"`
	CategoryId uint   `json:"category_id"`
}

type CategoryAddArgs struct {
	Name     string `json:"name"`
	ParentID uint   `json:"parent_id"`
	Sort     uint   `json:"sort"`
}

type CategoryEditArgs struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	ParentID uint   `json:"parent_id"`
	Sort     uint   `json:"sort"`
}

type PermissionAddArgs struct {
	Name         string `json:"name"`
	MenuID       uint   `json:"menu_id"`
	Code         string `json:"code"`
	InterfaceIds []uint `json:"interface_ids"`
}

type PermissionEditArgs struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	MenuID       uint   `json:"menu_id"`
	Code         string `json:"code"`
	InterfaceIds []uint `json:"interface_ids"`
}

type MenuAddArgs struct {
	Name     string `json:"name"`
	Route    string `json:"route"`
	Icon     string `json:"icon"`
	ParentID uint   `json:"parent_id"`
	Type     uint   `json:"type"`
	Sort     uint   `json:"sort"`
	Status   uint   `json:"status"`
}

type MenuEditArgs struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Route    string `json:"route"`
	Icon     string `json:"icon"`
	ParentID uint   `json:"parent_id"`
	Type     uint   `json:"type"`
	Sort     uint   `json:"sort"`
	Status   uint   `json:"status"`
}

type RoleAddArgs struct {
	Name string `json:"name"`
}

type RoleEditArgs struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type RolePermissionsArgs struct {
	RoleID uint `json:"role_id"`
}

type RolePermissionSetArgs struct {
	RoleID      uint            `json:"role_id"`
	Permissions map[uint][]uint `json:"permissions"`
}

type UserRoleSetArgs struct {
	UserID uint `json:"user_id"`
	RoleID uint `json:"role_id"`
}

type RoleNodeAddArgs struct {
	RoleID  uint   `json:"role_id"`
	NodeIDS []uint `json:"node_ids"`
}
