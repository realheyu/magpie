package magpiesdk_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	magpiesdk "github.com/realheyu/magpie/sdk"
)

func ExampleClient_Load() {
	client, err := magpiesdk.New(magpiesdk.Options{
		Endpoint:   "http://magpie.example.com:8081",
		AppName:    "app2-prod",
		APIKey:     "mgp_xxx",
		HTTPClient: exampleHTTPClient("[server]\nport = 8080\n"),
	})
	if err != nil {
		panic(err)
	}

	snapshot, err := client.Load(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Println(snapshot.AppName)
	fmt.Println(snapshot.Version)
	fmt.Print(snapshot.Content)

	// Output:
	// app2-prod
	// 7
	// [server]
	// port = 8080
}

func ExampleClient_LoadTOML() {
	type AppConfig struct {
		Server struct {
			Port int `toml:"port"`
		} `toml:"server"`
	}

	client, err := magpiesdk.New(magpiesdk.Options{
		Endpoint:   "http://magpie.example.com:8081",
		AppName:    "app2-prod",
		APIKey:     "mgp_xxx",
		HTTPClient: exampleHTTPClient("[server]\nport = 8080\n"),
	})
	if err != nil {
		panic(err)
	}

	var cfg AppConfig
	snapshot, err := client.LoadTOML(context.Background(), &cfg)
	if err != nil {
		panic(err)
	}

	fmt.Println(snapshot.Version)
	fmt.Println(cfg.Server.Port)

	// Output:
	// 7
	// 8080
}

func exampleHTTPClient(content string) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer mgp_xxx" {
			return newExampleResponse(http.StatusUnauthorized, nil, map[string]any{"code": 1, "msg": "API 密钥无效"}), nil
		}
		if r.URL.Path != "/v1/configs/app2-prod" || r.URL.Query().Get("meta") != "true" {
			return newExampleResponse(http.StatusNotFound, nil, map[string]any{"code": 1, "msg": "应用不存在"}), nil
		}

		header := http.Header{}
		header.Set("ETag", `"app2-prod-7-demo"`)
		return newExampleResponse(http.StatusOK, header, map[string]any{
			"code": 0,
			"msg":  "",
			"data": map[string]any{
				"appName":   "app2-prod",
				"format":    "toml",
				"content":   content,
				"sensitive": false,
				"version":   7,
				"etag":      `"app2-prod-7-demo"`,
				"updatedAt": time.Unix(1700000000, 0).UTC().Format(time.RFC3339),
			},
		}), nil
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newExampleResponse(statusCode int, header http.Header, body any) *http.Response {
	if header == nil {
		header = http.Header{}
	}
	buf := bytes.NewBuffer(nil)
	_ = json.NewEncoder(buf).Encode(body)
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Header:     header,
		Body:       io.NopCloser(buf),
	}
}
