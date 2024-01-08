package service

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"gorm.io/gorm"
	"task/model"
	"task/pkg/cas"
	"task/pkg/proto"
	"time"
)

const (
	TypeWeb uint = 1
)

var rbacService *rbac

type rbac struct{}

func (r *rbac) addInterface(ctx *gin.Context) {
	var addArgs proto.InterfaceAddArgs
	if err := ctx.ShouldBindJSON(&addArgs); err != nil {
		failed(ctx, 2001, "请求参数不合法")
		return
	}
	var i model.RbacInterface
	err := model.Task().First(&i, "route=?", addArgs.Route).Error
	if err == nil {
		failed(ctx, 2002, "接口已存在")
		return
	}
	err = model.Task().First(&i, "name=? and category_id=?", addArgs.Name, addArgs.CategoryId).Error
	if err == nil {
		failed(ctx, 2003, "同分类下接口名不允许重复")
		return
	}
	now := uint(time.Now().Unix())
	i = model.RbacInterface{
		Name:       addArgs.Name,
		Route:      addArgs.Route,
		Desc:       addArgs.Desc,
		Sort:       addArgs.Sort,
		CategoryID: addArgs.CategoryId,
		CreateTime: now,
		UpdateTime: now,
	}
	err = model.Task().Create(&i).Error
	if err != nil {
		failed(ctx, 2004, "添加失败")
		return
	}
	success(ctx, "添加成功", i.ID)
}

func (r *rbac) editInterface(ctx *gin.Context) {
	var editArgs proto.InterfaceEditArgs
	if err := ctx.ShouldBindJSON(&editArgs); err != nil {
		failed(ctx, 2005, "请求参数不合法")
		return
	}
	var i model.RbacInterface
	err := model.Task().First(&i, "id=?", editArgs.ID).Error
	if err != nil {
		failed(ctx, 2006, "接口不存在")
		return
	}
	var temp model.RbacInterface
	err = model.Task().First(&temp, "route=? and id<>?", editArgs.Route, editArgs.ID).Error
	if err == nil {
		failed(ctx, 2007, "接口地址已添加")
		return
	}
	err = model.Task().First(&temp, "category_id=? and name=? and id<>?", editArgs.CategoryId, editArgs.Name, editArgs.ID).Error
	if err == nil {
		failed(ctx, 2008, "接口名称不允许重复")
		return
	}
	i.Name = editArgs.Name
	i.Route = editArgs.Route
	i.Desc = editArgs.Desc
	i.Sort = editArgs.Sort
	i.CategoryID = editArgs.CategoryId
	i.UpdateTime = uint(time.Now().Unix())
	err = model.Task().Save(&i).Error
	if err != nil {
		failed(ctx, 2009, "修改失败")
		return
	}
	success(ctx, "修改成功", i.ID)
}

func (r *rbac) delInterface(ctx *gin.Context) {
	var delArgs proto.IdsArgs
	if err := ctx.ShouldBindJSON(&delArgs); err != nil {
		failed(ctx, 2010, "请求参数不合法")
		return
	}
	var interfaces []*model.RbacInterface
	err := model.Task().Where("id in (?)", delArgs.IDS).Find(&interfaces).Error
	if err == nil {
		for _, i := range interfaces {
			model.Task().Where("interface_id=?", i.ID).Delete(&model.RbacPermissionInterface{})
			model.Task().Delete(i)
		}
	}
	success(ctx, "删除成功", nil)
}

func (r *rbac) categoryList(ctx *gin.Context) {
	tree, _ := r.categoryInterfaceTree()
	success(ctx, "查询成功", tree)
}

func (r *rbac) addCategory(ctx *gin.Context) {
	var addArgs proto.CategoryAddArgs
	if err := ctx.ShouldBindJSON(&addArgs); err != nil {
		failed(ctx, 2011, "请求参数不合法")
		return
	}
	var category model.RbacInterfaceCategory
	err := model.Task().Where("name=? and parent_id=?", addArgs.Name, addArgs.ParentID).First(&category).Error
	if err == nil {
		failed(ctx, 2012, "分类名已存在")
		return
	}
	var level uint = 1
	if addArgs.ParentID > 0 {
		err = model.Task().First(&category, "id=?", addArgs.ParentID).Error
		if err != nil {
			failed(ctx, 2013, "父分类不存在")
			return
		}
		level = category.Level + 1
	}
	now := uint(time.Now().Unix())
	category = model.RbacInterfaceCategory{
		Name:       addArgs.Name,
		ParentID:   addArgs.ParentID,
		Level:      level,
		Sort:       addArgs.Sort,
		CreateTime: now,
		UpdateTime: now,
	}
	err = model.Task().Create(&category).Error
	if err != nil {
		failed(ctx, 2014, "添加失败")
		return
	}
	success(ctx, "添加成功", category.ID)
}

func (r *rbac) editCategory(ctx *gin.Context) {
	var editArgs proto.CategoryEditArgs
	if err := ctx.ShouldBindJSON(&editArgs); err != nil {
		failed(ctx, 2015, "请求参数不合法")
		return
	}
	var category model.RbacInterfaceCategory
	err := model.Task().First(&category, "id=?", editArgs.ID).Error
	if err != nil {
		failed(ctx, 2016, "分类不存在")
		return
	}
	var temp model.RbacInterfaceCategory
	err = model.Task().Where("name=? and parent_id=? and id<>", editArgs.Name, editArgs.ParentID, editArgs.ID).First(&temp).Error
	if err == nil {
		failed(ctx, 2017, "分类名已存在")
		return
	}
	var level uint = 1
	if editArgs.ParentID > 0 {
		err = model.Task().First(&temp, "id=?", editArgs.ParentID).Error
		if err != nil {
			failed(ctx, 2018, "父分类不存在")
			return
		}
		level = temp.Level + 1
	}
	var changeParentCategory bool
	if category.ParentID != editArgs.ParentID {
		changeParentCategory = true
	}
	category.Name = editArgs.Name
	category.ParentID = editArgs.ParentID
	category.Sort = editArgs.Sort
	category.Level = level
	category.UpdateTime = uint(time.Now().Unix())
	err = model.Task().Save(&category).Error
	if err != nil {
		failed(ctx, 2019, "修改失败")
		return
	}
	if changeParentCategory {
		r.recursionEditChildCategory(category.ID, category.Level)
	}
	success(ctx, "修改成功", nil)
}

func (r *rbac) delCategory(ctx *gin.Context) {
	var delArgs proto.IdArgs
	if err := ctx.ShouldBindJSON(&delArgs); err != nil {
		failed(ctx, 2020, "请求参数不合法")
		return
	}
	var category model.RbacInterfaceCategory
	err := model.Task().First(&category, "id=?", delArgs.ID).Error
	if err == nil {
		r.recursionDelCategory(category.ID)
	}
	success(ctx, "删除成功", nil)
}

type permissionDetail struct {
	model.RbacPermission
	Interfaces []*model.RbacInterface `json:"interfaces"`
}

func (r *rbac) permission(ctx *gin.Context) {
	var idArgs proto.IdArgs
	if err := ctx.ShouldBindJSON(&idArgs); err != nil {
		failed(ctx, 2021, "请求参数不合法")
		return
	}
	var p model.RbacPermission
	err := model.Task().First(&p, "id=?", idArgs.ID).Error
	if err != nil {
		failed(ctx, 2022, "权限不存在")
		return
	}
	d := permissionDetail{}
	d.ID = p.ID
	d.Name = p.Name
	d.MenuID = p.MenuID
	d.Code = p.Code
	d.CreateTime = p.CreateTime
	d.UpdateTime = p.UpdateTime
	var pi []*model.RbacPermissionInterface
	err = model.Task().Where("permission_id=?", p.ID).Find(&pi).Error
	if err == nil {
		var interfaceIds []uint
		for _, i := range pi {
			interfaceIds = append(interfaceIds, i.InterfaceID)
		}
		if len(interfaceIds) > 0 {
			model.Task().Where("id in (?)", interfaceIds).Find(&d.Interfaces)
		}
	}
	success(ctx, "查询成功", &d)
}

func (r *rbac) addPermission(ctx *gin.Context) {
	var addArgs proto.PermissionAddArgs
	if err := ctx.ShouldBindJSON(&addArgs); err != nil {
		failed(ctx, 2023, "请求参数不合法")
		return
	}
	var menu model.RbacMenu
	err := model.Task().First(&menu, "id=?", addArgs.MenuID).Error
	if err != nil {
		failed(ctx, 2024, "对应菜单不存在")
		return
	}
	var p model.RbacPermission
	err = model.Task().First(&p, "menu_id=? and name=?", addArgs.MenuID, addArgs.Name).Error
	if err == nil {
		failed(ctx, 2025, "同菜单下权限名称不允许重复")
		return
	}
	var now = uint(time.Now().Unix())
	p = model.RbacPermission{}
	p.Name = addArgs.Name
	p.MenuID = addArgs.MenuID
	p.Code = addArgs.Code
	p.CreateTime = now
	p.UpdateTime = now
	err = model.Task().Create(&p).Error
	if err != nil {
		failed(ctx, 2026, "添加失败")
		return
	}
	if len(addArgs.InterfaceIds) > 0 {
		var pi []*model.RbacPermissionInterface
		for _, id := range addArgs.InterfaceIds {
			pi = append(pi, &model.RbacPermissionInterface{
				PermissionID: p.ID,
				InterfaceID:  id,
				CreateTime:   now,
			})
		}
		model.Task().Create(&pi)
	}
	success(ctx, "添加成功", p.ID)
}

func (r *rbac) editPermission(ctx *gin.Context) {
	var editArgs proto.PermissionEditArgs
	if err := ctx.ShouldBindJSON(&editArgs); err != nil {
		failed(ctx, 2027, "请求参数不合法")
		return
	}
	var p model.RbacPermission
	err := model.Task().First(&p, "id=?", editArgs.ID).Error
	if err != nil {
		failed(ctx, 2028, "权限不存在")
		return
	}
	var temp model.RbacPermission
	err = model.Task().First(&temp, "name=? and menu_id=? and id<>?", editArgs.Name, editArgs.MenuID, editArgs.ID).Error
	if err == nil {
		failed(ctx, 2029, "同菜单下权限名不允许重复")
		return
	}
	var now = uint(time.Now().Unix())
	p.Name = editArgs.Name
	p.MenuID = editArgs.MenuID
	p.Code = editArgs.Code
	p.UpdateTime = now
	err = model.Task().Save(&p).Error
	if err != nil {
		failed(ctx, 2030, "修改失败")
		return
	}
	model.Task().Delete(&model.RbacPermissionInterface{}, "permission_id=?", p.ID)
	if len(editArgs.InterfaceIds) > 0 {
		var pi []*model.RbacPermissionInterface
		for _, id := range editArgs.InterfaceIds {
			pi = append(pi, &model.RbacPermissionInterface{
				PermissionID: p.ID,
				InterfaceID:  id,
				CreateTime:   now,
			})
		}
		model.Task().Create(&pi)
	}
	success(ctx, "修改成功", p.ID)
}

func (r *rbac) delPermission(ctx *gin.Context) {
	var idArgs proto.IdArgs
	if err := ctx.ShouldBindJSON(&idArgs); err != nil {
		failed(ctx, 2031, "请求参数不合法")
		return
	}
	var p model.RbacPermission
	err := model.Task().First(&p, "id=?", idArgs.ID).Error
	if err == nil {
		model.Task().Where("permission_id=?", idArgs.ID).Delete(&model.RbacPermissionInterface{})
		model.Task().Delete(&p)
	}
	success(ctx, "删除成功", nil)
}

func (r *rbac) menuList(ctx *gin.Context) {
	tree, _ := r.menuTree(0, 0)
	success(ctx, "查询成功", tree)
}

func (r *rbac) addMenu(ctx *gin.Context) {
	var addArgs proto.MenuAddArgs
	if err := ctx.ShouldBindJSON(&addArgs); err != nil {
		failed(ctx, 2032, "请求参数不合法")
		return
	}
	var menu model.RbacMenu
	err := model.Task().First(&menu, "name=? and parent_id=?", addArgs.Name, addArgs.ParentID).Error
	if err == nil {
		failed(ctx, 2033, "同层级菜单名不允许重复")
		return
	}
	var level uint = 1
	if addArgs.ParentID > 0 {
		err = model.Task().First(&menu, "id=?", addArgs.ParentID).Error
		if err != nil {
			failed(ctx, 2034, "父菜单不存在")
			return
		}
		level = menu.Level + 1
	}
	now := uint(time.Now().Unix())
	menu = model.RbacMenu{}
	menu.Name = addArgs.Name
	menu.Route = addArgs.Route
	menu.Icon = addArgs.Icon
	menu.ParentID = addArgs.ParentID
	menu.Level = level
	menu.Type = addArgs.Type
	menu.Sort = addArgs.Sort
	menu.Status = addArgs.Status
	menu.CreateTime = now
	menu.UpdateTime = now
	err = model.Task().Create(&menu).Error
	if err != nil {
		fmt.Println(err.Error())
		failed(ctx, 2035, "添加失败")
		return
	}
	success(ctx, "添加成功", menu.ID)
}

func (r *rbac) editMenu(ctx *gin.Context) {
	var editArgs proto.MenuEditArgs
	if err := ctx.ShouldBindJSON(&editArgs); err != nil {
		failed(ctx, 2036, "请求参数不合法")
		return
	}
	var menu model.RbacMenu
	err := model.Task().First(&menu, "id=?", editArgs.ID).Error
	if err != nil {
		failed(ctx, 2037, "菜单不存在")
		return
	}
	var temp model.RbacMenu
	err = model.Task().First(&temp, "name=? and parent_id=? and id<>?", editArgs.Name, editArgs.ParentID, editArgs.ID).Error
	if err == nil {
		failed(ctx, 2038, "同层级下菜单名不允许重复")
		return
	}
	var level uint = 1
	if editArgs.ParentID > 0 {
		err = model.Task().First(&temp, "id=?", editArgs.ParentID).Error
		if err != nil {
			failed(ctx, 2039, "父菜单不存在")
			return
		}
		level = temp.Level + 1
	}
	var changeParentCategory bool
	if menu.ParentID != editArgs.ParentID {
		changeParentCategory = true
	}
	menu.Name = editArgs.Name
	menu.Route = editArgs.Route
	menu.Icon = editArgs.Icon
	menu.ParentID = editArgs.ParentID
	menu.Type = editArgs.Type
	menu.Level = level
	menu.Sort = editArgs.Sort
	menu.Status = editArgs.Status
	menu.UpdateTime = uint(time.Now().Unix())
	err = model.Task().Save(&menu).Error
	if err != nil {
		failed(ctx, 2040, "修改失败")
		return
	}
	if changeParentCategory {
		r.recursionEditChildMenu(menu.ID, menu.Level)
	}
	success(ctx, "修改成功", nil)
}

func (r *rbac) delMenu(ctx *gin.Context) {
	var idArgs proto.IdArgs
	if err := ctx.ShouldBindJSON(&idArgs); err != nil {
		failed(ctx, 2041, "请求参数不合法")
		return
	}
	var menu model.RbacMenu
	err := model.Task().First(&menu, "id=?", idArgs.ID).Error
	if err == nil {
		r.recursionDelMenu(menu.ID)
	}
	success(ctx, "删除成功", idArgs.ID)
}

type roleListItem struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	NodeIDS    []uint `json:"node_ids"`
	CreateTime uint   `json:"create_time"`
	UpdateTime uint   `json:"update_time"`
}

func (r *rbac) roleList(ctx *gin.Context) {
	rl := make([]*roleListItem, 0)
	var roles []*model.RbacRole
	err := model.Task().Find(&roles).Error
	if err == nil {
		for _, role := range roles {
			i := &roleListItem{
				ID:         role.ID,
				Name:       role.Name,
				CreateTime: role.CreateTime,
				UpdateTime: role.UpdateTime,
			}
			if role.ID != proto.RoleAdmin {
				i.NodeIDS = r.getRoleNodeIDS(role.ID)
			}
			rl = append(rl, i)
		}
	}
	success(ctx, "查询成功", rl)
}

func (r *rbac) addRole(ctx *gin.Context) {
	var addArgs proto.RoleAddArgs
	if err := ctx.ShouldBindJSON(&addArgs); err != nil {
		failed(ctx, 2043, "请求参数不合法")
		return
	}
	var role model.RbacRole
	err := model.Task().First(&role, "name=?", addArgs.Name).Error
	if err == nil {
		failed(ctx, 2044, "角色名已存在")
		return
	}
	now := uint(time.Now().Unix())
	role = model.RbacRole{}
	role.Name = addArgs.Name
	role.CreateTime = now
	role.UpdateTime = now
	err = model.Task().Create(&role).Error
	if err != nil {
		failed(ctx, 2045, "添加失败")
		return
	}
	success(ctx, "添加成功", role.ID)
}

func (r *rbac) editRole(ctx *gin.Context) {
	var editArgs proto.RoleEditArgs
	if err := ctx.ShouldBindJSON(&editArgs); err != nil {
		failed(ctx, 2046, "请求参数不合法")
		return
	}
	var role model.RbacRole
	err := model.Task().First(&role, "id=?", editArgs.ID).Error
	if err != nil {
		failed(ctx, 2047, "角色不存在")
		return
	}
	role.Name = editArgs.Name
	role.UpdateTime = uint(time.Now().Unix())
	err = model.Task().Save(&role).Error
	if err != nil {
		failed(ctx, 2048, "修改失败")
		return
	}
	success(ctx, "修改成功", role.ID)
}

func (r *rbac) delRole(ctx *gin.Context) {
	var idArgs proto.IdArgs
	if err := ctx.ShouldBindJSON(&idArgs); err != nil {
		failed(ctx, 2049, "请求参数不合法")
		return
	}
	var role model.RbacRole
	err := model.Task().First(&role, "id=?", idArgs.ID).Error
	if err == nil {
		err = model.Task().Delete(&role).Error
		if err != nil {
			failed(ctx, 2050, "删除失败")
			return
		}
	}
	success(ctx, "删除成功", nil)
}

func (r *rbac) rolePermissionList(ctx *gin.Context) {
	var listArgs proto.RolePermissionsArgs
	if err := ctx.ShouldBindJSON(&listArgs); err != nil {
		failed(ctx, 2051, "请求参数不合法")
		return
	}
	var rp []*model.RbacRolePermission
	err := model.Task().Find(&rp, "role_id=?", listArgs.RoleID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		failed(ctx, 2052, "查询失败")
		return
	}
	var ids []uint
	for _, i := range rp {
		ids = append(ids, i.PermissionID)
	}
	var ps []*model.RbacPermission
	err = model.Task().Find(&ps, "id in (?)", ids).Error
	if err != nil {
		failed(ctx, 2053, "查询失败")
		return
	}
	success(ctx, "查询成功", ps)
}

func (r *rbac) saveRolePermissionList(ctx *gin.Context) {
	var setArgs proto.RolePermissionSetArgs
	if err := ctx.ShouldBindJSON(&setArgs); err != nil {
		failed(ctx, 2054, "请求参数不合法")
		return
	}
	var role model.RbacRole
	err := model.Task().First(&role, "id=?", setArgs.RoleID).Error
	if err != nil {
		failed(ctx, 2055, "角色不存在")
		return
	}
	var rp []*model.RbacRolePermission
	model.Task().Delete(&rp, "role_id=?", setArgs.RoleID)
	if len(setArgs.Permissions) > 0 {
		rp = rp[:0]
		now := uint(time.Now().Unix())
		for k, v := range setArgs.Permissions {
			for _, vv := range v {
				rp = append(rp, &model.RbacRolePermission{
					RoleID:       setArgs.RoleID,
					PermissionID: vv,
					MenuId:       k,
					CreateTime:   now,
				})
			}
		}
		model.Task().Create(&rp)
	}
	success(ctx, "保存成功", nil)
}

type userList struct {
	Total uint           `json:"total"`
	List  []userListItem `json:"list"`
}

type userListItem struct {
	ID       uint   `json:"id"`
	Phone    string `json:"phone"`
	RealName string `json:"real_name"`
	Status   uint   `json:"status"`
	IsAdmin  uint   `json:"is_admin"`
	RoleID   uint   `json:"role_id"`
	RoleName string `json:"role_name"`
	AddTime  string `json:"add_time"`
}

func (r *rbac) userList(ctx *gin.Context) {
	result := userList{
		Total: 0,
		List:  make([]userListItem, 0),
	}
	users := r.getUsers(ctx)
	if users != nil {
		result.Total = uint(len(users))
		for _, user := range users {
			item := userListItem{
				ID:       user.ID,
				Phone:    user.Phone,
				RealName: user.RealName,
				Status:   user.Status,
				IsAdmin:  user.IsAdmin,
				AddTime:  user.AddTime,
			}
			var userRole model.RbacRoleUser
			err := model.Task().Where("user_id=?", user.ID).First(&userRole).Error
			if err == nil {
				item.RoleID = userRole.RoleID
				var role model.RbacRole
				model.Task().Where("id=?", userRole.RoleID).First(&role)
				item.RoleName = role.Name
			}
			result.List = append(result.List, item)
		}
	}
	success(ctx, "查询成功", result)
}

func (r *rbac) refreshUserList(ctx *gin.Context) {
	cacheKey := "task_app_users"
	cache(0).Del(cacheKey)
	success(ctx, "刷新成功", nil)
}

func (r *rbac) setUserRole(ctx *gin.Context) {
	var setArgs proto.UserRoleSetArgs
	if err := ctx.ShouldBindJSON(&setArgs); err != nil {
		failed(ctx, 2057, "请求参数不合法")
		return
	}
	var ur model.RbacRoleUser
	model.Task().Delete(&ur, "user_id=?", setArgs.UserID)
	ur = model.RbacRoleUser{}
	ur.UserID = setArgs.UserID
	ur.RoleID = setArgs.RoleID
	ur.CreateTime = uint(time.Now().Unix())
	err := model.Task().Create(&ur).Error
	if err != nil {
		failed(ctx, 2058, "设置失败")
		return
	}
	success(ctx, "设置成功", nil)
}

type cUser struct {
	cas.User
	Menu []*mTree `json:"menu"`
}

func (r *rbac) currentUser(ctx *gin.Context) {
	user := r.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	cu := &cUser{}
	cu.ID = user.ID
	cu.Phone = user.Phone
	cu.RealName = user.RealName
	cu.Job = user.Job
	cu.JobNumber = user.JobNumber
	cu.HiredDate = user.HiredDate
	cu.WorkType = user.WorkType
	cu.Status = user.Status
	cu.WorkPlace = user.WorkPlace
	cu.IsAdmin = user.IsAdmin
	cu.AddTime = user.AddTime
	tree, _ := r.menuTree(TypeWeb, 1)
	if user.IsAdmin == 1 {
		cu.Menu = tree
	} else {
		var userRole model.RbacRoleUser
		model.Task().Where("user_id=?", user.ID).First(&userRole)
		if userRole.RoleID == proto.RoleAdmin {
			cu.Menu = tree
			cu.IsAdmin = 1
		} else {
			var rolePermissions []*model.RbacRolePermission
			err := model.Task().Where("role_id=?", userRole.RoleID).Find(&rolePermissions).Error
			if err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					failed(ctx, 1001, "Service Internal Error!!")
					return
				}
				failed(ctx, 1000, "Forbidden Access!!")
				return
			}
			var ids []uint
			for _, rp := range rolePermissions {
				ids = append(ids, rp.PermissionID)
			}
			cu.Menu = r.recursionValidMenuTree(tree, ids)
		}
	}
	success(ctx, "查询成功", cu)
}

func (r *rbac) isAdmin(userId uint) bool {
	var roleUser model.RbacRoleUser
	err := model.Task().Where("user_id=?", userId).First(&roleUser).Error
	if err == nil && roleUser.RoleID == proto.RoleAdmin {
		return true
	}
	return false
}

func (r *rbac) rolePermissions(roleId uint, t uint, s uint) ([]*model.RbacPermission, error) {
	var rolePermissions []*model.RbacRolePermission
	err := model.Task().Where("role_id=?", roleId).Find(&rolePermissions).Error
	if err != nil {
		return nil, err
	}
	var permissions []*model.RbacPermission
	for _, rp := range rolePermissions {
		var menu model.RbacMenu
		err = model.Task().Where("id=?", rp.MenuId).First(&menu).Error
		if err != nil {
			return nil, err
		}
		if menu.Status == 0 || (menu.Type != 0 && t != menu.Type) || (s == 1 && menu.Status == 0) {
			continue
		}
		var permission model.RbacPermission
		err = model.Task().Where("id=?", rp.PermissionID).First(&permission).Error
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, &permission)
	}
	return permissions, nil
}

func (r *rbac) userPermissions(userId uint, t uint) ([]*model.RbacPermission, error) {
	var userRole model.RbacRoleUser
	err := model.Task().Where("user_id=?", userId).First(&userRole).Error
	if err != nil {
		return nil, err
	}
	return r.rolePermissions(userRole.RoleID, t, 1)
}

func (r *rbac) getPermissionsByInterfaceRoute(route string) ([]*model.RbacPermission, error) {
	var i model.RbacInterface
	err := model.Task().Where("route=?", route).First(&i).Error
	if err != nil {
		return nil, err
	}
	var ip []*model.RbacPermissionInterface
	err = model.Task().Where("interface_id=?", i.ID).Find(&ip).Error
	if err != nil {
		return nil, err
	}
	var permissionIds []uint
	for _, v := range ip {
		permissionIds = append(permissionIds, v.PermissionID)
	}
	var permissions []*model.RbacPermission
	err = model.Task().Where("id in (?)", permissionIds).Find(&permissions).Error
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

type CITree struct {
	model.RbacInterfaceCategory
	Interfaces []*model.RbacInterface `json:"interfaces"`
	Children   []*CITree              `json:"children"`
}

func (r *rbac) categoryInterfaceTree() ([]*CITree, error) {
	var categories []*model.RbacInterfaceCategory
	err := model.Task().Order("sort").Find(&categories).Error
	if err != nil {
		return nil, err
	}
	return r.recursionCITree(categories, 0), nil
}

func (r *rbac) recursionCITree(data []*model.RbacInterfaceCategory, pid uint) []*CITree {
	var tree []*CITree
	for _, v := range data {
		if v.ParentID == pid {
			ct := &CITree{}
			ct.ID = v.ID
			ct.Name = v.Name
			ct.ParentID = v.ParentID
			ct.Level = v.Level
			ct.Sort = v.Sort
			ct.CreateTime = v.CreateTime
			ct.UpdateTime = v.UpdateTime
			model.Task().Where("category_id=?", v.ID).Order("sort").Find(&ct.Interfaces)
			ct.Children = r.recursionCITree(data, ct.ID)
			tree = append(tree, ct)
		}
	}
	return tree
}

func (r *rbac) recursionEditChildCategory(pid uint, level uint) {
	var categories []*model.RbacInterfaceCategory
	err := model.Task().Where("parent_id=?", pid).Find(&categories).Error
	if err == nil {
		for _, category := range categories {
			category.Level = level + 1
			category.UpdateTime = uint(time.Now().Unix())
			model.Task().Save(category)
			r.recursionEditChildCategory(category.ID, category.Level)
		}
	}
}

func (r *rbac) recursionDelCategory(cid uint) {
	var categories []*model.RbacInterfaceCategory
	err := model.Task().Where("parent_id=?", cid).Find(&categories).Error
	if err == nil {
		for _, category := range categories {
			r.recursionDelCategory(category.ID)
		}
	}
	var interfaces []*model.RbacInterface
	model.Task().Where("category_id=?", cid).Delete(&interfaces)
	model.Task().Delete(&model.RbacInterfaceCategory{}, cid)
}

type mTree struct {
	model.RbacMenu
	Permissions []*model.RbacPermission
	Children    []*mTree
}

func (r *rbac) menuTree(t uint, s uint) ([]*mTree, error) {
	var menus []*model.RbacMenu
	err := model.Task().Order("parent_id,sort").Find(&menus).Error
	if err != nil {
		return nil, err
	}
	return r.recursionMenuTree(menus, 0, t, s), nil
}

func (r *rbac) recursionMenuTree(data []*model.RbacMenu, pid uint, t uint, s uint) []*mTree {
	var tree []*mTree
	for _, v := range data {
		if v.ParentID == pid && (v.Type == 0 || t == 0 || v.Type == t) && (s == 0 || v.Status == 1) {
			mt := &mTree{}
			mt.ID = v.ID
			mt.Name = v.Name
			mt.Route = v.Route
			mt.Icon = v.Icon
			mt.ParentID = v.ParentID
			mt.Level = v.Level
			mt.Type = v.Type
			mt.Sort = v.Sort
			mt.Status = v.Status
			mt.CreateTime = v.CreateTime
			mt.UpdateTime = v.UpdateTime
			model.Task().Find(&mt.Permissions, "menu_id=?", v.ID)
			cData := make([]*model.RbacMenu, 0)
			model.Task().Where("parent_id=?", v.ID).Order("sort").Find(&cData)
			mt.Children = r.recursionMenuTree(cData, mt.ID, t, s)
			tree = append(tree, mt)
		}
	}
	return tree
}

func (r *rbac) recursionValidMenuTree(tree []*mTree, IDS []uint) []*mTree {
	var mt []*mTree
	if len(tree) == 0 {
		return mt
	}
	for _, t := range tree {
		if t.Status == 0 {
			continue
		}
		if len(t.Permissions) > 0 {
			ps := make([]*model.RbacPermission, 0)
			for k, p := range t.Permissions {
				for _, id := range IDS {
					if p.ID == id {
						ps = append(ps, t.Permissions[k])
						break
					}
				}
			}
			t.Permissions = ps
		}
		if len(t.Children) > 0 {
			t.Children = r.recursionValidMenuTree(t.Children, IDS)
		}
		if len(t.Children) > 0 || len(t.Permissions) > 0 {
			mt = append(mt, t)
		}
	}
	return mt
}

func (r *rbac) recursionEditChildMenu(pid uint, level uint) {
	var menus []*model.RbacMenu
	err := model.Task().Where("parent_id=?", pid).Find(&menus).Error
	if err == nil {
		for _, menu := range menus {
			menu.Level = level + 1
			menu.UpdateTime = uint(time.Now().Unix())
			model.Task().Save(menu)
			r.recursionEditChildMenu(menu.ID, menu.Level)
		}
	}
}

func (r *rbac) recursionDelMenu(mid uint) {
	var ms []*model.RbacMenu
	err := model.Task().Where("parent_id=?", mid).Find(&ms).Error
	if err == nil {
		for _, m := range ms {
			r.recursionDelMenu(m.ID)
		}
	}
	var ps []*model.RbacPermission
	err = model.Task().Find(&ps, "menu_id=?", mid).Error
	if err == nil {
		for _, p := range ps {
			model.Task().Delete(&[]*model.RbacPermissionInterface{}, "permission_id=?", p.ID)
		}
		model.Task().Delete(&[]*model.RbacPermission{}, "menu_id=?", mid)
	}
}

func (r *rbac) currentUserInfo(ctx *gin.Context) *cas.User {
	u, _ := ctx.Get("user")
	user, ok := u.(*cas.User)
	if !ok || user.ID == 0 {
		return nil
	}
	return user
}

func (r *rbac) currentUserRoleId(user *cas.User) uint {
	if user.IsAdmin == 1 {
		return proto.RoleAdmin
	}
	userRole := model.RbacRoleUser{}
	model.Task().First(&userRole, "user_id=?", user.ID)
	return userRole.RoleID
}

func (r *rbac) setRoleNode(ctx *gin.Context) {
	var addArgs proto.RoleNodeAddArgs
	if err := ctx.ShouldBindJSON(&addArgs); err != nil {
		failed(ctx, 2059, "请求参数不合法")
		return
	}
	user := r.currentUserInfo(ctx)
	if user == nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	if user.IsAdmin != 1 && !r.isAdmin(user.ID) && len(addArgs.NodeIDS) > 0 {
		var rn model.RbacRoleNode
		model.Task().Delete(&rn, "role_id=?", addArgs.RoleID)
		var rns []*model.RbacRoleNode
		now := uint(time.Now().Unix())
		for _, nodeId := range addArgs.NodeIDS {
			rns = append(rns, &model.RbacRoleNode{
				RoleID:     addArgs.RoleID,
				NodeID:     nodeId,
				CreateTime: now,
			})
		}
		err := model.Task().Create(&rns).Error
		if err != nil {
			failed(ctx, 2060, "设置失败")
			return
		}
	}
	success(ctx, "设置成功", nil)
}

func (r *rbac) getUserNodeIDS(userID uint) []uint {
	var userRole model.RbacRoleUser
	err := model.Task().First(&userRole, "user_id=?", userID).Error
	if err != nil {
		return make([]uint, 0)
	}
	return r.getRoleNodeIDS(userRole.RoleID)
}

func (r *rbac) getRoleNodeIDS(roleID uint) []uint {
	nodeIDS := make([]uint, 0)
	var roleNodes []*model.RbacRoleNode
	err := model.Task().Where("role_id=?", roleID).Find(&roleNodes).Error
	if err == nil {
		for _, rn := range roleNodes {
			nodeIDS = append(nodeIDS, rn.NodeID)
		}
	}
	return nodeIDS
}

func (r *rbac) getNodeRoleIDS(nodeID uint) []uint {
	roleIDS := make([]uint, 0)
	var roleNodes []*model.RbacRoleNode
	err := model.Task().Where("node_id=?", nodeID).Find(&roleNodes).Error
	if err == nil {
		for _, rn := range roleNodes {
			roleIDS = append(roleIDS, rn.RoleID)
		}
	}
	return roleIDS
}

func (r *rbac) getUsers(ctx *gin.Context) []cas.User {
	cacheKey := "task_app_users"
	bt, err := cache(0).Get(cacheKey).Bytes()
	var users []cas.User
	if err == nil {
		err = json.Unmarshal(bt, &users)
		if err != nil {
			return nil
		}
		return users
	}
	ul, err := casService(ctx).AppUsers()
	if err != nil {
		return nil
	}
	users = ul.List
	data, err := json.Marshal(ul.List)
	cache(0).Set(cacheKey, data, 15*time.Minute)
	return ul.List
}

func (r *rbac) getUserName(users *[]cas.User, userId uint) string {
	if userId == proto.UserAdmin {
		return "管理员"
	}
	for _, user := range *users {
		if user.ID == userId {
			return user.RealName
		}
	}
	return ""
}
