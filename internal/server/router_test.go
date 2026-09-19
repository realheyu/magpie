package server

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/realheyu/magpie/web"
)

func newSPARouter(t *testing.T, webBasePath string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	serveAdminSPA(r, webBasePath)
	return r
}

func TestServeAdminSPAInjectsBasePath(t *testing.T) {
	cases := []struct {
		basePath string
		want     string
	}{
		{"/", `window.__MAGPIE_BASE__ = "/"`},
		{"/magpie/", `window.__MAGPIE_BASE__ = "/magpie/"`},
	}
	for _, c := range cases {
		r := newSPARouter(t, c.basePath)
		for _, path := range []string{"/", "/apps", "/users/1"} {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
			if w.Code != http.StatusOK {
				t.Fatalf("GET %s with base %q: status = %d, want 200", path, c.basePath, w.Code)
			}
			if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
				t.Fatalf("GET %s with base %q: content-type = %q, want text/html", path, c.basePath, ct)
			}
			if !strings.Contains(w.Body.String(), c.want) {
				t.Fatalf("GET %s with base %q: index.html 未注入部署前缀 %q", path, c.basePath, c.want)
			}
			// 变量名本身含占位符子串，这里检查带引号的占位值已被替换
			if strings.Contains(w.Body.String(), `"`+basePathMarker+`"`) {
				t.Fatalf("GET %s with base %q: 占位符 %q 未被替换", path, c.basePath, basePathMarker)
			}
		}
	}
}

func TestServeAdminSPAAPIPrefixNotFallback(t *testing.T) {
	r := newSPARouter(t, "/magpie/")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/not-exist", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown api status = %d, want 404", w.Code)
	}
}

func TestServeAdminSPAServesStaticAssets(t *testing.T) {
	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		t.Fatalf("open dist: %v", err)
	}
	asset, err := firstFile(dist, "assets")
	if err != nil {
		t.Skipf("dist 中没有可测试的静态资源: %v", err)
	}
	r := newSPARouter(t, "/magpie/")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/"+asset, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET /assets/%s status = %d, want 200", asset, w.Code)
	}
	if strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("GET /assets/%s 应返回静态文件而不是 index.html", asset)
	}
}

func firstFile(fsys fs.FS, dir string) (string, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() {
			return e.Name(), nil
		}
	}
	return "", fs.ErrNotExist
}
