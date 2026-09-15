// Package magpiesdk provides a small Go client for reading Magpie app configs.
package magpiesdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

const defaultTimeout = 5 * time.Second

type Options struct {
	Endpoint   string
	AppName    string
	APIKey     string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type Client struct {
	endpoint   string
	appName    string
	apiKey     string
	httpClient *http.Client
}

type Snapshot struct {
	AppName   string    `json:"appName"`
	Format    string    `json:"format"`
	Content   string    `json:"content"`
	Sensitive bool      `json:"sensitive"`
	Version   int64     `json:"version"`
	ETag      string    `json:"etag"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ServerError struct {
	StatusCode int
	Code       int
	Msg        string
}

func (e *ServerError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return fmt.Sprintf("magpie request failed: http %d, code %d", e.StatusCode, e.Code)
}

func New(options Options) (*Client, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(options.Endpoint), "/")
	if endpoint == "" {
		return nil, errors.New("endpoint is required")
	}
	if _, err := url.ParseRequestURI(endpoint); err != nil {
		return nil, fmt.Errorf("invalid endpoint: %w", err)
	}
	appName := strings.TrimSpace(options.AppName)
	if appName == "" {
		return nil, errors.New("appName is required")
	}
	apiKey := strings.TrimSpace(options.APIKey)
	if apiKey == "" {
		return nil, errors.New("apiKey is required")
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		timeout := options.Timeout
		if timeout <= 0 {
			timeout = defaultTimeout
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	return &Client{endpoint: endpoint, appName: appName, apiKey: apiKey, httpClient: httpClient}, nil
}

func (c *Client) Load(ctx context.Context) (Snapshot, error) {
	requestURL, err := c.configURL(true)
	if err != nil {
		return Snapshot{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return Snapshot{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Snapshot{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Snapshot{}, parseServerError(resp)
	}

	var payload resultPayload[Snapshot]
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Snapshot{}, err
	}
	if payload.Code != 0 {
		return Snapshot{}, &ServerError{StatusCode: resp.StatusCode, Code: payload.Code, Msg: payload.Msg}
	}
	snapshot := payload.Data
	if snapshot.ETag == "" {
		snapshot.ETag = resp.Header.Get("ETag")
	}
	if snapshot.Version == 0 {
		if version, err := strconv.ParseInt(resp.Header.Get("X-Magpie-Version"), 10, 64); err == nil {
			snapshot.Version = version
		}
	}
	return snapshot, nil
}

func (c *Client) LoadString(ctx context.Context) (string, error) {
	snapshot, err := c.Load(ctx)
	if err != nil {
		return "", err
	}
	return snapshot.Content, nil
}

func (c *Client) LoadTOML(ctx context.Context, out any) (Snapshot, error) {
	if out == nil {
		return Snapshot{}, errors.New("out is required")
	}
	snapshot, err := c.Load(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	if _, err := toml.Decode(snapshot.Content, out); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func (c *Client) Watch(ctx context.Context, interval time.Duration, onChange func(Snapshot)) error {
	if onChange == nil {
		return errors.New("onChange is required")
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}

	snapshot, err := c.Load(ctx)
	if err != nil {
		return err
	}
	onChange(snapshot)
	lastETag := snapshot.ETag
	lastVersion := snapshot.Version

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			snapshot, err := c.Load(ctx)
			if err != nil {
				// 拉取失败按瞬时故障处理，跳过本轮、下个周期继续重试，
				// 避免一次网络抖动就让监听协程退出；只有 ctx 取消才结束 Watch。
				if ctx.Err() != nil {
					return ctx.Err()
				}
				continue
			}
			if snapshotChanged(snapshot, lastETag, lastVersion) {
				onChange(snapshot)
				lastETag = snapshot.ETag
				lastVersion = snapshot.Version
			}
		}
	}
}

func (c *Client) configURL(meta bool) (string, error) {
	parsed, err := url.Parse(c.endpoint)
	if err != nil {
		return "", err
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/v1/configs/" + url.PathEscape(c.appName)
	if meta {
		query := parsed.Query()
		query.Set("meta", "true")
		parsed.RawQuery = query.Encode()
	}
	return parsed.String(), nil
}

type resultPayload[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

func snapshotChanged(snapshot Snapshot, lastETag string, lastVersion int64) bool {
	if lastETag != "" && snapshot.ETag != "" {
		return snapshot.ETag != lastETag
	}
	if lastVersion != 0 && snapshot.Version != 0 {
		return snapshot.Version != lastVersion
	}
	return true
}

func parseServerError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var payload resultPayload[json.RawMessage]
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&payload); err == nil {
		return &ServerError{StatusCode: resp.StatusCode, Code: payload.Code, Msg: payload.Msg}
	}
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = resp.Status
	}
	return &ServerError{StatusCode: resp.StatusCode, Msg: msg}
}
