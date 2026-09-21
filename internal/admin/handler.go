package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/bt-smart/btutil/result"
	"github.com/gin-gonic/gin"
	"github.com/realheyu/magpie/internal/domain"
	"github.com/realheyu/magpie/internal/security"
	"github.com/realheyu/magpie/internal/store"
	"gorm.io/gorm"
)

const currentUserKey = "currentUser"

type Handler struct {
	store    *store.Store
	sessions *security.SessionManager
}

func NewHandler(store *store.Store, sessions *security.SessionManager) *Handler {
	return &Handler{store: store, sessions: sessions}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) { result.GinData(c, gin.H{"status": "ok"}) })

	api := r.Group("/api/admin")
	api.POST("/login", h.login)

	auth := api.Group("")
	auth.Use(h.authRequired())
	auth.POST("/logout", h.logout)
	auth.GET("/me", h.me)
	auth.GET("/apps", h.listApps)
	auth.POST("/apps", h.adminRequired(), h.createApp)
	auth.GET("/apps/:appName", h.getApp)
	auth.PUT("/apps/:appName", h.updateApp)
	auth.DELETE("/apps/:appName", h.adminRequired(), h.deleteApp)
	auth.GET("/apps/:appName/revisions", h.listRevisions)
	auth.POST("/apps/:appName/rollback", h.rollbackApp)
	auth.POST("/apps/:appName/restore", h.adminRequired(), h.restoreApp)

	adminOnly := auth.Group("")
	adminOnly.Use(h.adminRequired())
	adminOnly.GET("/users", h.listUsers)
	adminOnly.POST("/users", h.createUser)
	adminOnly.PUT("/users/:id", h.updateUser)
	adminOnly.DELETE("/users/:id", h.deleteUser)
	adminOnly.GET("/users/:id/permissions", h.getUserPermissions)
	adminOnly.PUT("/users/:id/permissions", h.setUserPermissions)
	adminOnly.GET("/api-keys", h.listAPIKeys)
	adminOnly.POST("/api-keys", h.createAPIKey)
	adminOnly.DELETE("/api-keys/:id", h.deleteAPIKey)
	adminOnly.PUT("/api-keys/:id/apps", h.updateAPIKeyApps)
	adminOnly.PUT("/api-keys/:id/status", h.updateAPIKeyStatus)
	adminOnly.GET("/audit-logs", h.listAuditLogs)
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if !bindJSON(c, &req) {
		return
	}
	user, err := h.store.FindUserByUsername(req.Username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// 数据库故障不该伪装成密码错误，否则排障方向会被带偏
		fail(c, http.StatusInternalServerError, "读取用户信息失败："+err.Error())
		return
	}
	if err != nil || user.Status != domain.StatusActive || !security.VerifyPassword(req.Password, user.PasswordHash, user.PasswordSalt) {
		fail(c, http.StatusUnauthorized, "用户名或密码错误")
		return
	}
	if err := h.store.TouchUserLogin(user.ID); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	token, expiresAt, err := h.sessions.Create(user.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	result.GinData(c, loginResponse{Token: token, ExpiresAt: expiresAt, User: toUserResp(user)})
}

func (h *Handler) logout(c *gin.Context) {
	if token := security.BearerToken(c.GetHeader("Authorization")); token != "" {
		h.sessions.Revoke(token)
	}
	result.GinOk(c)
}

func (h *Handler) me(c *gin.Context) {
	result.GinData(c, toUserResp(currentUser(c)))
}

func (h *Handler) listApps(c *gin.Context) {
	user := currentUser(c)
	page, pageSize := pagination(c)
	rows, total, err := h.store.ListAppsForUser(user, strings.TrimSpace(c.Query("query")), page, pageSize)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]appResp, 0, len(rows))
	for i := range rows {
		permission := domain.PermissionFull
		if user.Role != domain.RoleAdmin {
			permission = rows[i].Permission
			if !domain.ValidPermission(permission) {
				fail(c, http.StatusInternalServerError, "读取应用权限失败")
				return
			}
		}
		items = append(items, h.toAppResp(&rows[i].App, user, permission))
	}
	result.GinPage(c, items, total)
}

func (h *Handler) createApp(c *gin.Context) {
	var req saveAppRequest
	if !bindJSON(c, &req) {
		return
	}
	user := currentUser(c)
	app := &store.App{AppName: req.AppName, Description: req.Description, Format: req.Format, Content: req.Content, Sensitive: req.Sensitive, Status: req.Status}
	if err := h.store.CreateApp(app, user.ID, req.ChangeSummary); err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	result.GinData(c, h.toAppResp(app, user, domain.PermissionFull))
}

func (h *Handler) getApp(c *gin.Context) {
	if c.Param("appName") == "options" {
		h.listAppOptions(c)
		return
	}
	user := currentUser(c)
	app, permission, ok := h.loadVisibleApp(c)
	if !ok {
		return
	}
	result.GinData(c, h.toAppResp(app, user, permission))
}

func (h *Handler) updateApp(c *gin.Context) {
	user := currentUser(c)
	app, permission, ok := h.loadVisibleApp(c)
	if !ok {
		return
	}
	if user.Role != domain.RoleAdmin && !domain.CanEdit(permission) {
		fail(c, http.StatusForbidden, "没有编辑该应用的权限")
		return
	}
	var req saveAppRequest
	if !bindJSON(c, &req) {
		return
	}
	if user.Role != domain.RoleAdmin {
		req.Sensitive = app.Sensitive
		req.Status = app.Status
	}
	updated, err := h.store.UpdateApp(app.AppName, req.Description, req.Format, req.Content, req.Sensitive, req.Status, user.ID, req.ChangeSummary)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	result.GinData(c, h.toAppResp(updated, user, domain.PermissionFull))
}

func (h *Handler) deleteApp(c *gin.Context) {
	if err := h.store.DeleteApp(c.Param("appName"), currentUser(c).ID); err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	result.GinOk(c)
}

func (h *Handler) listRevisions(c *gin.Context) {
	user := currentUser(c)
	appName := c.Param("appName")
	app, err := h.store.GetAppByName(appName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) && user.Role == domain.RoleAdmin {
			h.listRevisionPage(c, appName, false)
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "应用不存在")
		} else {
			fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	permission := domain.PermissionFull
	if user.Role != domain.RoleAdmin {
		if app.Status != domain.StatusActive {
			fail(c, http.StatusNotFound, "应用不存在")
			return
		}
		var permissionErr error
		permission, permissionErr = h.store.GetUserAppPermission(user.ID, app.ID)
		if permissionErr != nil || !domain.ValidPermission(permission) {
			fail(c, http.StatusForbidden, "没有查看该应用的权限")
			return
		}
	}
	h.listRevisionPage(c, app.AppName, user.Role != domain.RoleAdmin && permission == domain.PermissionMasked)
}

func (h *Handler) listRevisionPage(c *gin.Context, appName string, maskSensitive bool) {
	page, pageSize := pagination(c)
	revisions, total, err := h.store.ListAppRevisions(appName, page, pageSize)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]revisionResp, 0, len(revisions))
	for _, revision := range revisions {
		content := revision.Content
		if maskSensitive && revision.Sensitive {
			content = domain.MaskContent(content)
		}
		items = append(items, revisionResp{ID: revision.ID, AppName: revision.AppName, Version: revision.Version, Format: revision.Format, Content: content, Sensitive: revision.Sensitive, ChangeSummary: revision.ChangeSummary, CreatedByUserID: revision.CreatedByUserID, CreatedAt: revision.CreatedAt})
	}
	result.GinPage(c, items, total)
}

func (h *Handler) rollbackApp(c *gin.Context) {
	user := currentUser(c)
	_, permission, ok := h.loadVisibleApp(c)
	if !ok {
		return
	}
	if user.Role != domain.RoleAdmin && !domain.CanEdit(permission) {
		fail(c, http.StatusForbidden, "没有回滚该应用的权限")
		return
	}
	var req rollbackRequest
	if !bindJSON(c, &req) {
		return
	}
	app, err := h.store.RollbackApp(c.Param("appName"), req.Version, user.ID, user.Role != domain.RoleAdmin)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	result.GinData(c, h.toAppResp(app, user, domain.PermissionFull))
}

func (h *Handler) restoreApp(c *gin.Context) {
	var req restoreAppRequest
	if !bindJSON(c, &req) {
		return
	}
	user := currentUser(c)
	app, err := h.store.RestoreAppFromRevision(c.Param("appName"), req.Version, user.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "应用版本不存在")
			return
		}
		fail(c, http.StatusOK, err.Error())
		return
	}
	result.GinData(c, h.toAppResp(app, user, domain.PermissionFull))
}

func (h *Handler) listUsers(c *gin.Context) {
	page, pageSize := pagination(c)
	users, total, err := h.store.ListUsers(strings.TrimSpace(c.Query("query")), page, pageSize)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]userResp, 0, len(users))
	for i := range users {
		items = append(items, toUserResp(&users[i]))
	}
	result.GinPage(c, items, total)
}

func (h *Handler) createUser(c *gin.Context) {
	var req createUserRequest
	if !bindJSON(c, &req) {
		return
	}
	user, err := h.store.CreateUser(req.Username, req.DisplayName, req.Password, req.Role, req.Status, currentUser(c).ID)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	result.GinData(c, toUserResp(user))
}

func (h *Handler) updateUser(c *gin.Context) {
	id, ok := paramUint(c, "id")
	if !ok {
		return
	}
	var req updateUserRequest
	if !bindJSON(c, &req) {
		return
	}
	user, err := h.store.UpdateUser(id, req.DisplayName, req.Password, req.Role, req.Status, currentUser(c).ID)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	result.GinData(c, toUserResp(user))
}

func (h *Handler) deleteUser(c *gin.Context) {
	id, ok := paramUint(c, "id")
	if !ok {
		return
	}
	if err := h.store.DeleteUser(id, currentUser(c).ID); err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	result.GinOk(c)
}

func (h *Handler) getUserPermissions(c *gin.Context) {
	id, ok := paramUint(c, "id")
	if !ok {
		return
	}
	permissions, apps, err := h.store.ListUserPermissions(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	appNames := make(map[uint64]string, len(apps))
	for _, app := range apps {
		appNames[app.ID] = app.AppName
	}
	items := make([]permissionEntry, 0, len(permissions))
	for _, permission := range permissions {
		items = append(items, permissionEntry{AppName: appNames[permission.AppID], Permission: permission.Permission})
	}
	result.GinData(c, permissionsResponse{Permissions: items})
}

func (h *Handler) setUserPermissions(c *gin.Context) {
	id, ok := paramUint(c, "id")
	if !ok {
		return
	}
	var req permissionsRequest
	if !bindJSON(c, &req) {
		return
	}
	entries := make(map[string]string, len(req.Permissions))
	for _, permission := range req.Permissions {
		entries[permission.AppName] = permission.Permission
	}
	if err := h.store.ReplaceUserPermissions(id, entries, currentUser(c).ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "用户不存在")
			return
		}
		fail(c, http.StatusOK, err.Error())
		return
	}
	result.GinOk(c)
}

func (h *Handler) listAPIKeys(c *gin.Context) {
	page, pageSize := pagination(c)
	keys, total, err := h.store.ListAPIKeys(page, pageSize)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]apiKeyResp, 0, len(keys))
	for i := range keys {
		item, err := h.toAPIKeyResp(&keys[i])
		if err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		items = append(items, item)
	}
	result.GinPage(c, items, total)
}

func (h *Handler) createAPIKey(c *gin.Context) {
	var req createAPIKeyRequest
	if !bindJSON(c, &req) {
		return
	}
	plainKey, err := security.NewAPIKey()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	apiKey, err := h.store.CreateAPIKey(req.Name, security.HashAPIKey(plainKey), security.APIKeyPreview(plainKey), req.AppNames, req.ExpiresAt, currentUser(c).ID)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	item, err := h.toAPIKeyResp(apiKey)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	result.GinData(c, createAPIKeyResponse{APIKey: plainKey, Item: item})
}

func (h *Handler) deleteAPIKey(c *gin.Context) {
	id, ok := paramUint(c, "id")
	if !ok {
		return
	}
	if err := h.store.DeleteAPIKey(id, currentUser(c).ID); err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	result.GinOk(c)
}

func (h *Handler) updateAPIKeyApps(c *gin.Context) {
	id, ok := paramUint(c, "id")
	if !ok {
		return
	}
	var req updateAPIKeyAppsRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.store.ReplaceAPIKeyApps(id, req.AppNames, currentUser(c).ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "API 密钥不存在")
			return
		}
		fail(c, http.StatusOK, err.Error())
		return
	}
	result.GinOk(c)
}

func (h *Handler) updateAPIKeyStatus(c *gin.Context) {
	id, ok := paramUint(c, "id")
	if !ok {
		return
	}
	var req updateAPIKeyStatusRequest
	if !bindJSON(c, &req) {
		return
	}
	apiKey, err := h.store.UpdateAPIKeyStatus(id, req.Status, currentUser(c).ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "API 密钥不存在")
			return
		}
		fail(c, http.StatusOK, err.Error())
		return
	}
	item, err := h.toAPIKeyResp(apiKey)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	result.GinData(c, item)
}

func (h *Handler) listAppOptions(c *gin.Context) {
	if currentUser(c).Role != domain.RoleAdmin {
		fail(c, http.StatusForbidden, "需要管理员权限")
		return
	}
	apps, err := h.store.ListAppOptions(strings.TrimSpace(c.Query("query")))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]appOptionResp, 0, len(apps))
	for i := range apps {
		items = append(items, toAppOptionResp(&apps[i]))
	}
	result.GinData(c, items)
}

func (h *Handler) listAuditLogs(c *gin.Context) {
	page, pageSize := pagination(c)
	logs, total, err := h.store.ListAuditLogs(strings.TrimSpace(c.Query("query")), page, pageSize)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]auditLogResp, 0, len(logs))
	for i := range logs {
		items = append(items, toAuditLogResp(&logs[i]))
	}
	result.GinPage(c, items, total)
}

func (h *Handler) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := security.BearerToken(c.GetHeader("Authorization"))
		userID, ok := h.sessions.Validate(token)
		if !ok {
			fail(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}
		user, err := h.store.FindUserByID(userID)
		if err != nil || user.Status != domain.StatusActive {
			fail(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}
		c.Set(currentUserKey, user)
		c.Next()
	}
}

func (h *Handler) adminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if currentUser(c).Role != domain.RoleAdmin {
			fail(c, http.StatusForbidden, "需要管理员权限")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *Handler) loadVisibleApp(c *gin.Context) (*store.App, string, bool) {
	app, err := h.store.GetAppByName(c.Param("appName"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "应用不存在")
		} else {
			fail(c, http.StatusInternalServerError, err.Error())
		}
		return nil, "", false
	}
	user := currentUser(c)
	if user.Role == domain.RoleAdmin {
		return app, domain.PermissionFull, true
	}
	if app.Status != domain.StatusActive {
		fail(c, http.StatusNotFound, "应用不存在")
		return nil, "", false
	}
	permission, err := h.store.GetUserAppPermission(user.ID, app.ID)
	if err != nil || !domain.ValidPermission(permission) {
		fail(c, http.StatusForbidden, "没有查看该应用的权限")
		return nil, "", false
	}
	return app, permission, true
}

func (h *Handler) toAppResp(app *store.App, user *store.User, permission string) appResp {
	content := app.Content
	if user.Role != domain.RoleAdmin && app.Sensitive && permission == domain.PermissionMasked {
		content = domain.MaskContent(content)
	}
	return appResp{ID: app.ID, AppName: app.AppName, Description: app.Description, Format: app.Format, Content: content, Sensitive: app.Sensitive, Version: app.Version, Status: app.Status, Permission: permission, CreatedAt: app.CreatedAt, UpdatedAt: app.UpdatedAt}
}

func (h *Handler) toAPIKeyResp(apiKey *store.APIKey) (apiKeyResp, error) {
	apps, err := h.store.ListAPIKeyApps(apiKey.ID)
	if err != nil {
		return apiKeyResp{}, err
	}
	appNames := make([]string, 0, len(apps))
	for _, app := range apps {
		appNames = append(appNames, app.AppName)
	}
	return apiKeyResp{ID: apiKey.ID, Name: apiKey.Name, KeyPreview: apiKey.KeyPreview, Status: apiKey.Status, AppNames: appNames, ExpiresAt: apiKey.ExpiresAt, LastUsedAt: apiKey.LastUsedAt, CreatedAt: apiKey.CreatedAt, UpdatedAt: apiKey.UpdatedAt}, nil
}

func toUserResp(user *store.User) userResp {
	return userResp{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Role: user.Role, Status: user.Status, LastLoginAt: user.LastLoginAt, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt}
}

func toAppOptionResp(app *store.App) appOptionResp {
	return appOptionResp{AppName: app.AppName, Description: app.Description, Format: app.Format, Sensitive: app.Sensitive, Version: app.Version, Status: app.Status}
}

func toAuditLogResp(log *store.AuditLog) auditLogResp {
	return auditLogResp{ID: log.ID, ActorType: log.ActorType, ActorID: log.ActorID, Action: log.Action, ResourceType: log.ResourceType, ResourceID: log.ResourceID, Metadata: log.Metadata, CreatedAt: log.CreatedAt}
}

func currentUser(c *gin.Context) *store.User {
	user, _ := c.Get(currentUserKey)
	return user.(*store.User)
}

func bindJSON(c *gin.Context, out any) bool {
	if err := c.ShouldBindJSON(out); err != nil {
		fail(c, http.StatusOK, "请求参数错误")
		return false
	}
	return true
}

func pagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	return page, pageSize
}

func paramUint(c *gin.Context, key string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(key), 10, 64)
	if err != nil {
		fail(c, http.StatusOK, "参数格式错误："+key)
		return 0, false
	}
	return id, true
}

func fail(c *gin.Context, status int, msg string) {
	c.JSON(status, result.FailWithMsg(msg))
}
