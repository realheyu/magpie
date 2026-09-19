package server

import (
	"bytes"
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

// basePathMarker 是 index.html 里预留的部署前缀占位符，
// 构建产物中保持原样，运行时由后端替换成实际的 webBasePath。
const basePathMarker = "__MAGPIE_BASE__"

func AdminRouter(store *store.Store, sessions *security.SessionManager, ginRequestLog bool, webBasePath string) *gin.Engine {
	r := newGinEngine(ginRequestLog)
	admin.NewHandler(store, sessions).RegisterRoutes(r)
	serveAdminSPA(r, webBasePath)
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

func serveAdminSPA(r *gin.Engine, webBasePath string) {
	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		panic(err)
	}
	indexHTML := mustLoadIndexHTML(dist, webBasePath)
	fileServer := http.FileServer(http.FS(dist))
	r.NoRoute(func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if strings.HasPrefix(path, "api/") {
			c.JSON(http.StatusNotFound, result.FailWithMsg("接口不存在"))
			return
		}
		if path != "" {
			if file, err := dist.Open(path); err == nil {
				info, statErr := file.Stat()
				_ = file.Close()
				// 目录不放行给 FileServer，避免暴露嵌入目录的文件列表，统一回退到 SPA 入口
				if statErr == nil && !info.IsDir() {
					fileServer.ServeHTTP(c.Writer, c.Request)
					return
				}
			}
		}
		// 首页和 SPA 路由回退统一返回注入了部署前缀的 index.html
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
}

func mustLoadIndexHTML(dist fs.FS, webBasePath string) []byte {
	raw, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		panic(err)
	}
	// 连同引号整体替换：占位符是全局变量名 __MAGPIE_BASE__ 的子串，裸替换会破坏变量名
	return bytes.Replace(raw, []byte(`"`+basePathMarker+`"`), []byte(`"`+webBasePath+`"`), 1)
}
