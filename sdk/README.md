# Magpie Go SDK 使用示例

业务服务可以通过 Go SDK 读取正式配置中心里的应用配置。SDK 每次调用都会真实请求配置 API，不做本地缓存。

## 安装

```sh
go get github.com/realheyu/magpie/sdk
```

## 连接正式配置中心

推荐把正式地址、应用名、API Key 放到环境变量或业务服务自己的配置文件里，不要写死在代码仓库中。

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	magpiesdk "github.com/realheyu/magpie/sdk"
)

func main() {
	client, err := magpiesdk.New(magpiesdk.Options{
		Endpoint: os.Getenv("MAGPIE_ENDPOINT"),
		AppName:  os.Getenv("MAGPIE_APP_NAME"),
		APIKey:   os.Getenv("MAGPIE_API_KEY"),
		Timeout:  5 * time.Second, // 单次 HTTP 请求超时，默认 5s
		MaxRetries: 2,             // 失败后的重试次数，默认 0 不重试；只重试网络错误和 408/429/5xx
	})
	if err != nil {
		log.Fatal(err)
	}

	// 注意：重试有线性退避（200ms、400ms…封顶 1s），含重试的总耗时上界约为 Timeout*(MaxRetries+1)+退避时间，
	// 调用方的 ctx 超时要留够余量。
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	snapshot, err := client.Load(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(snapshot.AppName, snapshot.Version)
	fmt.Println(snapshot.Content)
}
```

运行：

```sh
MAGPIE_ENDPOINT=http://magpie.example.com:6081 \
MAGPIE_APP_NAME=app2-prod \
MAGPIE_API_KEY=mgp_xxx \
go run ./examples/go-sdk
```

## 解析配置内容

SDK 只负责拉取原始内容（`snapshot.Content`），怎么解析成结构体由业务自己决定。TOML：

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
if _, err := toml.Decode(snapshot.Content, &cfg); err != nil { // github.com/BurntSushi/toml
	log.Fatal(err)
}
fmt.Println(snapshot.Version, cfg.Server.Port)
```

JSON 用标准库即可：

```go
var cfg AppConfig
if err := json.Unmarshal([]byte(snapshot.Content), &cfg); err != nil {
	log.Fatal(err)
}
```

## 监听变化

`Watch` 会按间隔轮询配置 API，并用本次监听循环里的上一版 `ETag/version` 判断是否触发回调；它不会把配置内容持久化到本地。轮询过程中遇到瞬时错误（网络抖动、服务端重启）会跳过当轮并在下个周期自动重试，只有 `ctx` 取消才会退出；首次 `Load` 失败仍会直接返回错误，方便启动时暴露配置地址或密钥问题。

```go
err := client.Watch(context.Background(), 30*time.Second, func(snapshot magpiesdk.Snapshot) {
	fmt.Printf("配置更新到版本 %d\n", snapshot.Version)
})
if err != nil {
	log.Fatal(err)
}
```
