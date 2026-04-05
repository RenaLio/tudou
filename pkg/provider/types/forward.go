package types

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/goccy/go-json"
)

type ForwardRequest struct {
	Method  string            `json:"method"`  // GET/POST/PUT...
	Path    string            `json:"path"`    // /v1/order
	Query   map[string]string `json:"query"`   // ?a=1&b=2
	Headers map[string]string `json:"headers"` // 请求头
	Body    json.RawMessage   `json:"body"`    // 任意 JSON body
}

type ForwardFunc func(ctx context.Context, req *ForwardRequest) (*http.Response, error)

func ForwardHTTP(ctx context.Context, httpC *http.Client, baseURL string, reqData *ForwardRequest) (*http.Response, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("解析BaseURL失败: %w", err)
	}
	u.Path = path.Join(u.Path, reqData.Path)

	// 处理 Query 参数
	if len(reqData.Query) > 0 {
		q := u.Query()
		for key, value := range reqData.Query {
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
	}

	// 构造请求 Body
	var bodyReader io.Reader
	if len(reqData.Body) > 0 {
		bodyReader = bytes.NewReader(reqData.Body)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, reqData.Method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置 Headers
	for key, value := range reqData.Headers {
		req.Header.Set(key, value)
	}

	// 执行请求
	resp, err := httpC.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行HTTP请求失败: %w", err)
	}

	return resp, nil
}
