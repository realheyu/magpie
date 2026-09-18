package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	magpiesdk "github.com/realheyu/magpie/sdk"
)

func main() {
	endpoint := flag.String("endpoint", env("MAGPIE_ENDPOINT", ""), "Magpie 配置 API 地址，例如 http://magpie.example.com:8081")
	appName := flag.String("app", env("MAGPIE_APP_NAME", ""), "应用名，例如 app2-prod")
	apiKey := flag.String("api-key", env("MAGPIE_API_KEY", ""), "API Key，也可以使用 MAGPIE_API_KEY 环境变量")
	timeout := flag.Duration("timeout", 5*time.Second, "请求超时时间")
	decodeTOML := flag.Bool("decode-toml", true, "按 TOML 解析配置并打印顶层 key")
	printContent := flag.Bool("print-content", false, "打印配置明文，正式环境谨慎开启")
	flag.Parse()

	if *endpoint == "" || *appName == "" || *apiKey == "" {
		fmt.Fprintln(os.Stderr, "缺少参数：endpoint、app、api-key 都不能为空")
		fmt.Fprintln(os.Stderr, "示例：MAGPIE_ENDPOINT=http://magpie.example.com:8081 MAGPIE_APP_NAME=app2-prod MAGPIE_API_KEY=mgp_xxx go run ./examples/go-sdk")
		os.Exit(2)
	}

	client, err := magpiesdk.New(magpiesdk.Options{
		Endpoint: strings.TrimRight(*endpoint, "/"),
		AppName:  *appName,
		APIKey:   *apiKey,
		Timeout:  *timeout,
	})
	if err != nil {
		log.Fatalf("创建 SDK 客户端失败：%v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	snapshot, err := client.Load(ctx)
	if err != nil {
		log.Fatalf("拉取配置失败：%v", err)
	}
	printSnapshot(snapshot)

	if *decodeTOML && snapshot.Format == "toml" {
		// SDK 只给原始内容，业务自己解析成结构体
		var cfg map[string]any
		if _, err := toml.Decode(snapshot.Content, &cfg); err != nil {
			log.Fatalf("解析 TOML 失败：%v", err)
		}
		fmt.Printf("TOML 顶层 key：%s\n", strings.Join(sortedKeys(cfg), ", "))
	}
	if *printContent {
		fmt.Println("\n配置内容：")
		fmt.Println(snapshot.Content)
	}
}

func printSnapshot(snapshot magpiesdk.Snapshot) {
	fmt.Printf("应用：%s\n", snapshot.AppName)
	fmt.Printf("格式：%s\n", snapshot.Format)
	fmt.Printf("版本：%d\n", snapshot.Version)
	fmt.Printf("ETag：%s\n", snapshot.ETag)
	fmt.Printf("更新时间：%s\n", snapshot.UpdatedAt.Format(time.RFC3339))
	fmt.Printf("配置字节数：%d\n", len(snapshot.Content))
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
