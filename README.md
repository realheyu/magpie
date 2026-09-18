# Magpie

Magpie is a small Go config center. Version 1 keeps the model intentionally simple: one app owns one complete config document, and permissions are granted at the app level.

## Stack

- Backend: Go, Gin, GORM, MySQL
- Frontend: Vue 3, Vite, TypeScript, pnpm, Element Plus
- Responses: `github.com/bt-smart/btutil/result`
- Password/API key helpers: `github.com/bt-smart/btutil/crypto` and `github.com/bt-smart/btutil/strutil`
- Config file: TOML with `github.com/BurntSushi/toml`
- Logging: Zap with local file rotation by `gopkg.in/natefinch/lumberjack.v2`

## Configuration

Copy `config.example.toml` to `config.toml` and set local secrets there. `config.toml` is ignored by git. You can also point to a different file with `MAGPIE_CONFIG_FILE`.

```sh
cp config.example.toml config.toml
```

`config.example.toml` only contains placeholder values. Keep real MySQL addresses, passwords, session secrets, and bootstrap passwords in ignored `config.toml` or environment variables.

MySQL uses the standard one-line DSN format:

```toml
[mysql]
url = "magpie_user:password@tcp(127.0.0.1:3306)/magpie?charset=utf8mb4&parseTime=True&loc=UTC"
```

Environment variables still override TOML values when present, including `MAGPIE_MYSQL_URL`, `MAGPIE_LOG_LEVEL`, and `MAGPIE_LOG_FILE_ENABLED`.

Log configuration lives in `config.toml`:

```toml
[log]
level = "info"
console = true
ginRequest = false

[log.file]
enabled = true
filename = "logs/magpie.log"
maxSizeMb = 100
maxBackups = 10
maxAgeDays = 30
compress = true
```

Gin runs in release mode by default. For local route debugging, set `server.ginMode = "debug"` and optionally `log.ginRequest = true`.

## Development

Start the backend:

```sh
go run ./cmd/magpie
```

The binary starts two HTTP servers:

- Admin console and admin API: `http://localhost:8080`
- Config API: `http://localhost:8081`

Start the frontend dev server:

```sh
cd web
pnpm install
pnpm dev
```

Build frontend assets for embedding:

```sh
cd web
pnpm build
```

`web/dist` 是构建产物，不入库。全新 clone 后必须先执行上面的前端构建，再编译单二进制（`go:embed` 依赖 `web/dist` 存在）：

```sh
go build ./cmd/magpie
```

## Docker

Dockerfile uses layered multi-stage builds: frontend dependencies, frontend build, Go dependencies, Go build, and runtime image are separated. The runtime image defaults to Shanghai timezone.

```sh
docker build -t magpie:local .
```

The Alpine `apk` repository is switched to USTC mirror during build for domestic network stability.

Run with a local ignored config file and writable log directory:

```sh
docker run --rm \
  -p 8080:8080 \
  -p 8081:8081 \
  -e TZ=Asia/Shanghai \
  -v "$PWD/config.toml:/etc/magpie/config.toml:ro" \
  -v "$PWD/logs:/app/logs" \
  magpie:local
```

The image includes `/etc/magpie/config.example.toml` as a sanitized template. Keep real MySQL passwords and API secrets in mounted `config.toml` or environment variables.

## Config API

Read the full config content for an app:

```sh
curl -H 'Authorization: Bearer mgp_xxx' http://localhost:8081/v1/configs/app2-prod
```

Read metadata-wrapped JSON:

```sh
curl -H 'Authorization: Bearer mgp_xxx' 'http://localhost:8081/v1/configs/app2-prod?meta=true'
```

The config endpoint sets `ETag` and honors `If-None-Match` with `304 Not Modified`.

## Admin API Notes

- `GET /api/admin/apps/options` returns all current apps with lightweight fields for admin selectors.
- App deletion physically removes the current `apps` row and related grants, but keeps `app_revisions`; admins can restore a deleted app from a revision with `POST /api/admin/apps/{appName}/restore`.
- Non-admin users with `full` app permission may edit config content, but app-level metadata such as `status` and `sensitive` is preserved server-side.

## Go SDK

业务服务可以直接使用 Go SDK 拉取配置。SDK 默认请求 `?meta=true`，每次 `Load` 都会真实请求配置 API，不做本地缓存；返回值里会带上服务端的版本号和 `ETag`。

更完整的 SDK 使用说明见 `sdk/README.md`，可直接运行的正式连接示例见 `examples/go-sdk`。

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	magpiesdk "github.com/realheyu/magpie/sdk"
)

func main() {
	client, err := magpiesdk.New(magpiesdk.Options{
		Endpoint: "http://localhost:8081",
		AppName:  "app2-prod",
		APIKey:   "mgp_xxx",
	})
	if err != nil {
		log.Fatal(err)
	}

	snapshot, err := client.Load(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(snapshot.Content)
}
```

SDK 只返回原始配置内容和版本信息（`snapshot.Content`），解析成结构体由业务自己完成，例如 TOML 用 `github.com/BurntSushi/toml`：

```go
type AppConfig struct {
	Server struct {
		Port int `toml:"port"`
	} `toml:"server"`
}

snapshot, err := client.Load(context.Background())
if err != nil {
	log.Fatal(err)
}
var cfg AppConfig
if _, err := toml.Decode(snapshot.Content, &cfg); err != nil {
	log.Fatal(err)
}
fmt.Println(snapshot.Version, cfg.Server.Port)
```

也可以用轮询监听配置变化：

```go
err := client.Watch(context.Background(), 30*time.Second, func(snapshot magpiesdk.Snapshot) {
	fmt.Printf("配置更新到版本 %d\n", snapshot.Version)
})
```

仓库里也提供了一个可直接运行的示例程序：

```sh
MAGPIE_ENDPOINT=http://magpie.example.com:8081 \
MAGPIE_APP_NAME=app2-prod \
MAGPIE_API_KEY=mgp_xxx \
go run ./examples/go-sdk
```

默认只打印应用名、格式、版本、ETag、更新时间、配置字节数和 TOML 顶层 key，避免正式环境把配置明文打到终端。如果确认要查看明文，可以加：

```sh
go run ./examples/go-sdk -print-content
```

也可以完全使用命令行参数：

```sh
go run ./examples/go-sdk \
  -endpoint http://magpie.example.com:8081 \
  -app app2-prod \
  -api-key mgp_xxx
```
