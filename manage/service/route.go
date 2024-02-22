package service

import (
	"github.com/gin-gonic/gin"
)

func setRoute(e *gin.Engine) {
	//e.StaticFS("disk", http.Dir("disk"))
	e.POST("logout", logout)
	e.GET("message", message)
	setRbacRoute(e)
	setNodeRoute(e)
	setCrontabRoute(e)
	setDaemonRoute(e)
	setConfigRoute(e)
}

func setRbacRoute(e *gin.Engine) {
	e.Group("/rbac", request(), auth()).
		POST("/interface/add", rbacService.addInterface).
		POST("/interface/edit", rbacService.editInterface).
		POST("/interface/del", rbacService.delInterface).
		POST("/categories", rbacService.categoryList).
		POST("/category/add", rbacService.addCategory).
		POST("/category/edit", rbacService.editCategory).
		POST("/category/del", rbacService.delCategory).
		POST("/permission", rbacService.permission).
		POST("/permission/add", rbacService.addPermission).
		POST("/permission/edit", rbacService.editPermission).
		POST("/permission/del", rbacService.delPermission).
		POST("/menus", rbacService.menuList).
		POST("/menu/add", rbacService.addMenu).
		POST("/menu/edit", rbacService.editMenu).
		POST("/menu/del", rbacService.delMenu).
		POST("/roles", rbacService.roleList).
		POST("/role/add", rbacService.addRole).
		POST("/role/edit", rbacService.editRole).
		POST("/role/del", rbacService.delRole).
		POST("/role/permissions", rbacService.rolePermissionList).
		POST("/role/permission/save", rbacService.saveRolePermissionList).
		POST("/users", rbacService.userList).
		POST("/user/refresh", rbacService.refreshUserList).
		POST("/user/role/set", rbacService.setUserRole).
		POST("/user/current", rbacService.currentUser).
		POST("/role/node/set", rbacService.setRoleNode)
}

func setNodeRoute(e *gin.Engine) {
	e.Group("/node", request(), auth()).
		POST("/list", nodeService.list).
		POST("/log/list", nodeService.logList)
}

func setCrontabRoute(e *gin.Engine) {
	e.Group("/crontab", request(), auth()).
		POST("/list", cronService.list).
		POST("/add", cronService.addCrontab).
		POST("/get", cronService.getCrontab).
		POST("/edit", cronService.editCrontab).
		POST("/audit", cronService.auditCrontab).
		POST("/start", cronService.startCrontab).
		POST("/stop", cronService.stopCrontab).
		POST("/exec", cronService.execCrontab).
		POST("/kill", cronService.killCrontab).
		POST("/del", cronService.delCrontab).
		POST("/log/list", cronService.log).
		POST("/log/clean", cronService.clean)
}

func setDaemonRoute(e *gin.Engine) {
	e.Group("/daemon", request(), auth()).
		POST("/list", daemonService.list).
		POST("/add", daemonService.addDaemon).
		POST("/get", daemonService.getDaemon).
		POST("/edit", daemonService.editDaemon).
		POST("/audit", daemonService.auditDaemon).
		POST("/start", daemonService.startDaemon).
		POST("/stop", daemonService.stopDaemon).
		POST("/del", daemonService.delDaemon).
		POST("/log/list", daemonService.log)
}

func setConfigRoute(e *gin.Engine) {
	e.Group("/config", request(), auth()).
		POST("/list", configService.list).
		POST("/add", configService.add).
		POST("/edit", configService.edit).
		POST("/del", configService.del)
}
