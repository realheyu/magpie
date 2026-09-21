package configapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/bt-smart/btutil/result"
	"github.com/gin-gonic/gin"
	"github.com/realheyu/magpie/internal/domain"
	"github.com/realheyu/magpie/internal/security"
	"github.com/realheyu/magpie/internal/store"
	"gorm.io/gorm"
)

type Handler struct {
	store *store.Store
}

func NewHandler(store *store.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) { result.GinData(c, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) { result.GinData(c, gin.H{"status": "ok"}) })
	r.GET("/v1/configs/:appName", h.getConfig)
}

func (h *Handler) getConfig(c *gin.Context) {
	apiKey, ok := h.authenticate(c)
	if !ok {
		return
	}
	app, err := h.store.GetAppByName(c.Param("appName"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, result.FailWithMsg("应用不存在"))
		} else {
			c.JSON(http.StatusInternalServerError, result.FailWithMsg(err.Error()))
		}
		return
	}
	if app.Status != domain.StatusActive {
		c.JSON(http.StatusNotFound, result.FailWithMsg("应用不存在"))
		return
	}
	allowed, err := h.store.APIKeyCanAccessApp(apiKey.ID, app.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, result.FailWithMsg(err.Error()))
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, result.FailWithMsg("API 密钥无权读取该应用"))
		return
	}
	_ = h.store.TouchAPIKey(apiKey.ID)

	etag := buildETag(app.AppName, app.Version, app.Content)
	c.Header("ETag", etag)
	c.Header("X-Magpie-Version", strconv.FormatInt(app.Version, 10))
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
	if c.Query("meta") == "true" {
		result.GinData(c, configMetaResp{AppName: app.AppName, Format: app.Format, Content: app.Content, Sensitive: app.Sensitive, Version: app.Version, ETag: etag, UpdatedAt: app.UpdatedAt})
		return
	}
	c.Data(http.StatusOK, domain.ContentType(app.Format), []byte(app.Content))
}

func (h *Handler) authenticate(c *gin.Context) (*store.APIKey, bool) {
	token := security.BearerToken(c.GetHeader("Authorization"))
	if token == "" {
		c.JSON(http.StatusUnauthorized, result.FailWithMsg("缺少 API 密钥"))
		return nil, false
	}
	apiKey, err := h.store.FindActiveAPIKeyByHash(security.HashAPIKey(token))
	if err != nil {
		c.JSON(http.StatusUnauthorized, result.FailWithMsg("API 密钥无效"))
		return nil, false
	}
	return apiKey, true
}
