package magpiesdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoadSendsAPIKeyAndDoesNotCache(t *testing.T) {
	requestCount := 0
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer mgp_test" {
			t.Fatalf("authorization header mismatch: %q", r.Header.Get("Authorization"))
		}
		if r.URL.Scheme != "http" || r.URL.Host != "magpie.test" || r.URL.Path != "/v1/configs/app2-prod" || r.URL.Query().Get("meta") != "true" {
			t.Fatalf("unexpected request url: %s", r.URL.String())
		}
		if got := r.Header.Get("If-None-Match"); got != "" {
			t.Fatalf("sdk should not send conditional cache header, got %q", got)
		}
		requestCount++
		header := http.Header{}
		header.Set("ETag", fmt.Sprintf(`"app2-prod-%d-abc"`, requestCount))
		return newTestResponse(http.StatusOK, header, resultPayload[Snapshot]{
			Code: 0,
			Data: Snapshot{AppName: "app2-prod", Format: "toml", Content: fmt.Sprintf("[server]\nport = %d\n", requestCount), Version: int64(requestCount), UpdatedAt: time.Unix(10, 0).UTC()},
		}), nil
	})}

	client, err := New(Options{Endpoint: "http://magpie.test", AppName: "app2-prod", APIKey: "mgp_test", HTTPClient: httpClient})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	first, err := client.Load(context.Background())
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	if first.Content == "" || first.ETag == "" {
		t.Fatalf("unexpected first snapshot: %+v", first)
	}
	second, err := client.Load(context.Background())
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("expected two real requests, got %d", requestCount)
	}
	if second.Version != 2 || second.Content == first.Content {
		t.Fatalf("expected second load to fetch fresh content: first=%+v second=%+v", first, second)
	}
}

func TestLoadParsesServerError(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return newTestResponse(http.StatusForbidden, nil, resultPayload[json.RawMessage]{Code: 1, Msg: "API 密钥无权读取该应用"}), nil
	})}

	client, err := New(Options{Endpoint: "http://magpie.test", AppName: "app2-prod", APIKey: "mgp_test", HTTPClient: httpClient})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	_, err = client.Load(context.Background())
	var serverErr *ServerError
	if !errors.As(err, &serverErr) {
		t.Fatalf("expected ServerError, got %T: %v", err, err)
	}
	if serverErr.StatusCode != http.StatusForbidden || serverErr.Msg != "API 密钥无权读取该应用" {
		t.Fatalf("unexpected server error: %+v", serverErr)
	}
}

func TestLoadRetriesTransientFailures(t *testing.T) {
	var requests atomic.Int32
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		// 前两次返回 503 模拟服务端瞬时故障，第三次成功
		if requests.Add(1) <= 2 {
			return newTestResponse(http.StatusServiceUnavailable, nil, nil), nil
		}
		header := http.Header{}
		header.Set("ETag", `"app2-prod-1-abc"`)
		return newTestResponse(http.StatusOK, header, resultPayload[Snapshot]{
			Code: 0,
			Data: Snapshot{AppName: "app2-prod", Format: "toml", Content: "[server]\nport = 8080\n", Version: 1, ETag: `"app2-prod-1-abc"`},
		}), nil
	})}

	client, err := New(Options{Endpoint: "http://magpie.test", AppName: "app2-prod", APIKey: "mgp_test", MaxRetries: 2, HTTPClient: httpClient})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	snapshot, err := client.Load(context.Background())
	if err != nil {
		t.Fatalf("expected retry to recover, got: %v", err)
	}
	if snapshot.Version != 1 || requests.Load() != 3 {
		t.Fatalf("expected success on 3rd request, requests=%d snapshot=%+v", requests.Load(), snapshot)
	}
}

func TestLoadRetriesExhaustedReturnsLast(t *testing.T) {
	var requests atomic.Int32
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		return newTestResponse(http.StatusBadGateway, nil, nil), nil
	})}

	client, err := New(Options{Endpoint: "http://magpie.test", AppName: "app2-prod", APIKey: "mgp_test", MaxRetries: 2, HTTPClient: httpClient})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	_, err = client.Load(context.Background())
	var serverErr *ServerError
	if !errors.As(err, &serverErr) || serverErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected wrapped 502 ServerError, got: %v", err)
	}
	if requests.Load() != 3 {
		t.Fatalf("expected 1 initial + 2 retries, got %d requests", requests.Load())
	}
}

func TestLoadDoesNotRetryPermanentErrors(t *testing.T) {
	var requests atomic.Int32
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		return newTestResponse(http.StatusForbidden, nil, resultPayload[json.RawMessage]{Code: 1, Msg: "API 密钥无权读取该应用"}), nil
	})}

	client, err := New(Options{Endpoint: "http://magpie.test", AppName: "app2-prod", APIKey: "mgp_test", MaxRetries: 3, HTTPClient: httpClient})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	_, err = client.Load(context.Background())
	var serverErr *ServerError
	if !errors.As(err, &serverErr) || serverErr.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 ServerError, got: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("permanent error should not be retried, got %d requests", requests.Load())
	}
}

func TestNewValidatesOptions(t *testing.T) {
	if _, err := New(Options{}); err == nil {
		t.Fatal("expected empty endpoint to fail")
	}
	if _, err := New(Options{Endpoint: "http://127.0.0.1:6081", APIKey: "mgp"}); err == nil {
		t.Fatal("expected empty appName to fail")
	}
	if _, err := New(Options{Endpoint: "http://127.0.0.1:6081", AppName: "app"}); err == nil {
		t.Fatal("expected empty apiKey to fail")
	}
}

func TestWatchSurvivesTransientErrors(t *testing.T) {
	var requests atomic.Int32
	changes := 0
	etag := `"app2-prod-1-abc"`
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		// 第 1 次是初始 Load；第 2 次模拟网络抖动；之后恢复正常且内容不变
		if requests.Add(1) == 2 {
			return nil, errors.New("transient network error")
		}
		header := http.Header{}
		header.Set("ETag", etag)
		return newTestResponse(http.StatusOK, header, resultPayload[Snapshot]{
			Code: 0,
			Data: Snapshot{AppName: "app2-prod", Format: "toml", Content: "[server]\nport = 8080\n", Version: 1, ETag: etag},
		}), nil
	})}

	client, err := New(Options{Endpoint: "http://magpie.test", AppName: "app2-prod", APIKey: "mgp_test", HTTPClient: httpClient})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for requests.Load() < 3 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()
	err = client.Watch(ctx, 5*time.Millisecond, func(Snapshot) { changes++ })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if requests.Load() < 3 {
		t.Fatalf("watch should keep polling after a transient error, requests=%d", requests.Load())
	}
	if changes != 1 {
		t.Fatalf("expected only the initial snapshot callback, got %d", changes)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newTestResponse(statusCode int, header http.Header, body any) *http.Response {
	if header == nil {
		header = http.Header{}
	}
	var reader io.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		buf := bytes.NewBuffer(nil)
		_ = json.NewEncoder(buf).Encode(body)
		reader = buf
	}
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Header:     header,
		Body:       io.NopCloser(reader),
	}
}
