package base

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"github.com/RenaLio/tudou/pkg/provider/constant"
	"github.com/RenaLio/tudou/pkg/provider/plog"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/tidwall/gjson"
)

func ChatCompletionsParse(raw []byte) (*types.StreamEvent, error) {
	//line := string(raw)
	//event := &types.StreamEvent{
	//	Content:  raw,
	//	Finished: false,
	//	Usage:    nil,
	//}
	//if !strings.HasPrefix(line, "data: ") {
	//	return nil,nil
	//}
	//dataStr := strings.TrimPrefix(line, "data: ")
	//if dataStr == "[DONE]" {
	//	event.Finished = true
	//	return event, nil
	//}
	//// 解析usage
	//usage := parseUsage([]byte(dataStr), "usage")
	return parseSSEEvent(raw, nil, nil, "usage")
}

func OpenAIResponsesParse(raw []byte) (*types.StreamEvent, error) {
	return parseSSEEvent(raw, nil, nil, "usage", "response.usage")
}

func ClaudeMessagesParse(raw []byte) (*types.StreamEvent, error) {
	return parseSSEEvent(raw, map[string]struct{}{
		"message_stop": {},
	}, nil, "usage", "message.usage")
}

type SSEResult struct {
	DataStr  string
	DataType string // event or data
	Finished bool
	Ok       bool
}

type ExtractSSEDataFunc func(raw []byte) *SSEResult

func parseSSEEvent(raw []byte, finishTypes map[string]struct{}, extractFunc ExtractSSEDataFunc, usagePaths ...string) (*types.StreamEvent, error) {
	if extractFunc == nil {
		extractFunc = extractSSEData
	}
	result := extractFunc(raw)
	if !result.Ok {
		return nil, nil
	}
	if result.DataType == "event" {
		return &types.StreamEvent{
			Content:  raw,
			Finished: result.Finished,
		}, nil
	}

	event := &types.StreamEvent{
		Content:  raw,
		Finished: result.Finished,
	}
	if event.Finished {
		return event, nil
	}

	data := []byte(result.DataStr)
	if usage := parseUsage(data, usagePaths...); usage != nil {
		event.Usage = usage
	}

	return event, nil
}

func extractSSEData(raw []byte) *SSEResult {
	raw = bytes.TrimRight(raw, "\n")
	line := string(raw)
	if strings.HasPrefix(line, "event:") {
		return new(SSEResult{
			DataType: "event",
			Ok:       true,
		})
	}
	if !strings.HasPrefix(line, "data:") {
		return new(SSEResult{
			Ok: false,
		})
	}
	dataStr := strings.TrimPrefix(line, "data: ")
	if dataStr == "[DONE]" {
		return new(SSEResult{
			DataStr:  dataStr,
			Finished: true,
			Ok:       true,
		})
	}

	return new(SSEResult{
		DataStr:  dataStr,
		Finished: false,
		Ok:       true,
	})
}

func parseUsage(data []byte, usagePaths ...string) *types.Usage {
	for _, usagePath := range usagePaths {
		usageNode := gjson.GetBytes(data, usagePath)
		if !usageNode.Exists() || usageNode.Type == gjson.Null {
			continue
		}
		return parseUsageNode(usageNode)
	}
	return nil
}

func parseUsageNode(usageNode gjson.Result) *types.Usage {
	usage := &types.Usage{}

	if v, ok := getIntByPaths(usageNode, "input_tokens", "prompt_tokens"); ok {
		usage.InputTokens = v
	}
	if v, ok := getIntByPaths(usageNode, "output_tokens", "completion_tokens"); ok {
		usage.OutputTokens = v
	}
	if v, ok := getIntByPaths(usageNode, "total_tokens"); ok {
		usage.TotalTokens = v
	}

	if v, ok := getIntByPaths(usageNode, "cache_creation_input_tokens"); ok {
		usage.CachedCreationInputTokens = v
	}
	if v, ok := getIntByPaths(usageNode, "cache_read_input_tokens"); ok {
		usage.CachedReadInputTokens = v
	}
	if inputNode, ok := getNodeByPaths(usageNode, "input_token_details", "prompt_tokens_details"); ok {
		if v, ok := getIntByPaths(inputNode, "cached_tokens"); ok {
			usage.CachedTokens = v
		}
	}
	if usage.CachedTokens == 0 {
		usage.CachedTokens = usage.CachedCreationInputTokens + usage.CachedReadInputTokens
	}
	if outputNode, ok := getNodeByPaths(usageNode, "output_token_details", "completion_tokens_details"); ok {
		if v, ok := getIntByPaths(outputNode, "reasoning_tokens"); ok {
			usage.ReasoningTokens = v
		}
	}
	if usage.TotalTokens == 0 {
		total := usage.InputTokens + usage.OutputTokens
		if total > 0 {
			usage.TotalTokens = total
		}
	}
	return usage
}

func getNodeByPaths(parent gjson.Result, paths ...string) (gjson.Result, bool) {
	for _, path := range paths {
		node := parent.Get(path)
		if node.Exists() && node.Type != gjson.Null {
			return node, true
		}
	}
	return gjson.Result{}, false
}

func getIntByPaths(parent gjson.Result, paths ...string) (int64, bool) {
	for _, path := range paths {
		node := parent.Get(path)
		if node.Exists() {
			return node.Int(), true
		}
	}
	return 0, false
}

func (c *Client) ChatCompletion(ctx context.Context, reqUrl string, originReq *types.Request, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
	return c.executeJSONRequest(ctx, reqUrl, originReq, req, cb, ChatCompletionsParse, "usage")
}

func (c *Client) Responses(ctx context.Context, reqUrl string, originReq *types.Request, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
	return c.executeJSONRequest(ctx, reqUrl, originReq, req, cb, OpenAIResponsesParse, "usage", "response.usage")
}

func (c *Client) ClaudeMessages(ctx context.Context, reqUrl string, originReq *types.Request, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
	return c.executeJSONRequest(ctx, reqUrl, originReq, req, cb, ClaudeMessagesParse, "usage", "message.usage")
}

func (c *Client) executeJSONRequest(
	ctx context.Context,
	reqURL string,
	originReq *types.Request,
	req *types.Request,
	cb types.MetricsCallback,
	streamParse types.StreamParseFunc,
	usagePaths ...string,
) (*types.Response, error) {
	plog.Debug("executeJSONRequest", "reqURL", reqURL)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewBuffer(req.Payload))
	if err != nil {
		return nil, err
	}
	for k, v := range req.Headers {
		if len(v) == 0 {
			continue
		}
		request.Header.Set(k, v[0])
	}

	var (
		start     time.Time
		dnsStart  time.Time
		dnsDone   time.Time
		tcpStart  time.Time
		tcpDone   time.Time
		tlsStart  time.Time
		tlsDone   time.Time
		firstByte time.Time
	)
	trace := &httptrace.ClientTrace{
		DNSStart:             func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:              func(_ httptrace.DNSDoneInfo) { dnsDone = time.Now() },
		ConnectStart:         func(network, addr string) { tcpStart = time.Now() },
		ConnectDone:          func(network, addr string, err error) { tcpDone = time.Now() },
		TLSHandshakeStart:    func() { tlsStart = time.Now() },
		TLSHandshakeDone:     func(tls.ConnectionState, error) { tlsDone = time.Now() },
		GotFirstResponseByte: func() { firstByte = time.Now() },
	}
	request = request.WithContext(httptrace.WithClientTrace(request.Context(), trace))
	start = time.Now()
	httpResp, err := c.httpC.Do(request)
	if err != nil {
		return nil, err
	}

	metrics := new(types.ResponseMetrics)
	metrics.Provider = c.Identifier()
	metrics.Model = req.Model
	metrics.Format = req.Format
	metrics.IsStream = req.IsStream
	metrics.Extra = make(map[string]any)
	metrics.Extra[constant.RequestFormatKey] = req.Format
	var exceptedStatus bool
	if httpResp.StatusCode >= http.StatusOK && httpResp.StatusCode < http.StatusMultipleChoices {
		metrics.Status = 1
	} else {
		metrics.Status = 2
		exceptedStatus = true
	}
	if req.IsStream && !strings.Contains(httpResp.Header.Get("Content-Type"), "text/event-stream") {
		exceptedStatus = true
		metrics.Status = 2
	}

	if !dnsStart.IsZero() && !dnsDone.IsZero() {
		metrics.DNSTime = dnsDone.Sub(dnsStart)
	}
	if !tcpStart.IsZero() && !tcpDone.IsZero() {
		metrics.TCPTime = tcpDone.Sub(tcpStart)
	}
	if !tlsStart.IsZero() && !tlsDone.IsZero() {
		metrics.TLSTime = tlsDone.Sub(tlsStart)
	}
	if !firstByte.IsZero() {
		metrics.TTFB = firstByte.Sub(start)
	}

	if (!req.IsStream) || exceptedStatus {
		defer func() {
			_ = httpResp.Body.Close()
		}()
		plog.Debug("respStatusCode:", httpResp.StatusCode)
		plog.Debug("content-type:", httpResp.Header.Get("Content-Type"))
		data, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, err
		}
		plog.Debug("data:", string(data))
		if exceptedStatus {
			return nil, fmt.Errorf("unexpected status code: %d %s", httpResp.StatusCode, string(data))
		}

		if !firstByte.IsZero() {
			metrics.TTFT = metrics.TTFB
		}
		if !firstByte.IsZero() {
			metrics.TransferTime = time.Since(firstByte)
		}
		metrics.TotalTime = time.Since(start)

		if usage := parseUsage(data, usagePaths...); usage != nil {
			metrics.Usage = *usage
		}
		if cb != nil {
			cb(metrics)
		}

		resp := &types.Response{
			StatusCode: httpResp.StatusCode,
			Provider:   c.Identifier(),
			IsStream:   false,
			Format:     req.Format,
			RawData:    data,
			Header:     httpResp.Header,
		}

		return resp, nil
	}

	// 流式响应
	stream := types.NewStreamIterator(httpResp.Body, req, metrics, streamParse, cb, start)

	resp := &types.Response{
		StatusCode: httpResp.StatusCode,
		Provider:   c.Identifier(),
		IsStream:   true,
		Format:     req.Format,
		Header:     httpResp.Header,
		Stream:     stream,
	}

	return resp, nil
}
