package server

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/bt-smart/btutil/result"
	"github.com/gin-gonic/gin"
	"github.com/realheyu/magpie/internal/admin"
	"github.com/realheyu/magpie/internal/configapi"
	"github.com/realheyu/magpie/internal/security"
	"github.com/realheyu/magpie/internal/store"
	"github.com/realheyu/magpie/web"
)

func AdminRouter(store *store.Store, sessions *security.SessionManager, ginRequestLog bool) *gin.Engine {
	r := newGinEngine(ginRequestLog)
	admin.NewHandler(store, sessions).RegisterRoutes(r)
	serveAdminSPA(r)
	return r
}

func ConfigAPIRouter(store *store.Store, ginRequestLog bool) *gin.Engine {
	r := newGinEngine(ginRequestLog)
	configapi.NewHandler(store).RegisterRoutes(r)
	return r
}

func newGinEngine(ginRequestLog bool) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	if ginRequestLog {
		r.Use(gin.Logger())
	}
	return r
}

func serveAdminSPA(r *gin.Engine) {
	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(dist))
	r.NoRoute(func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if strings.HasPrefix(path, "api/") {
			c.JSON(http.StatusNotFound, result.FailWithMsg("接口不存在"))
			return
		}
		if path == "" {
			path = "index.html"
		}
		if file, err := dist.Open(path); err == nil {
			_ = file.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		c.Request.URL.Path = "/index.html"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}
