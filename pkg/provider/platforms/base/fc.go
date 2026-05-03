package base

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"github.com/RenaLio/tudou/pkg/provider/constant"
	"github.com/RenaLio/tudou/pkg/provider/plog"
	"github.com/RenaLio/tudou/pkg/provider/types"
	"github.com/tidwall/sjson"

	"github.com/tidwall/gjson"
)

type UsageParse func([]byte) *types.Usage

func ChatCompletionsParse(raw []byte) (*types.StreamEvent, error) {
	cloneRaw := bytes.TrimRight(raw, "\n")
	line := string(cloneRaw)
	// ping [Done] 之类的
	if !strings.HasPrefix(line, "data:") {
		return &types.StreamEvent{
			Content:  raw,
			Finished: false,
		}, nil
	}
	event := &types.StreamEvent{
		Content:  raw,
		Finished: false,
	}
	dataStr := strings.TrimPrefix(line, "data: ")
	if dataStr == "[DONE]" {
		event.Finished = true
		return event, nil
	}
	usage := ParseChatCompletionsUsage([]byte(dataStr))
	event.Usage = usage
	return event, nil
}

func ParseChatCompletionsUsage(data []byte) *types.Usage {
	usage := &types.Usage{}
	for _, usagePath := range []string{"usage"} {
		if usageNode := gjson.GetBytes(data, usagePath); usageNode.Exists() {
			usage.InputTokens = gjson.GetBytes(data, usagePath+".prompt_tokens").Int()
			usage.OutputTokens = gjson.GetBytes(data, usagePath+".completion_tokens").Int()
			usage.CachedReadInputTokens = gjson.GetBytes(data, usagePath+".prompt_tokens_details.cached_tokens").Int()
			usage.CachedTokens = usage.CachedCreationInputTokens + usage.CachedReadInputTokens
			usage.ReasoningTokens = gjson.GetBytes(data, usagePath+".completion_tokens_details.reasoning_tokens").Int()
			usage.TotalTokens = gjson.GetBytes(data, usagePath+".total_tokens").Int()
			if usage.TotalTokens <= 0 {
				usage.TotalTokens = usage.InputTokens + usage.OutputTokens
			}
		}
	}
	return usage
}

func OpenAIResponsesParse(raw []byte) (*types.StreamEvent, error) {
	cloneRaw := bytes.TrimRight(raw, "\n")
	line := string(cloneRaw)
	if strings.HasPrefix(line, "event:") {
		return &types.StreamEvent{
			Content:  raw,
			Finished: false,
		}, nil
	}
	// ping 之类的
	if !strings.HasPrefix(line, "data:") {
		return &types.StreamEvent{
			Content:  raw,
			Finished: false,
		}, nil
	}
	event := &types.StreamEvent{
		Content:  raw,
		Finished: false,
	}
	dataStr := strings.TrimPrefix(line, "data: ")
	usage := ParseOpenAIResponsesUsage([]byte(dataStr))
	event.Usage = usage
	return event, nil
}

func ParseOpenAIResponsesUsage(data []byte) *types.Usage {
	usage := &types.Usage{}
	for _, usagePath := range []string{"usage", "response.usage"} {
		if usageNode := gjson.GetBytes(data, usagePath); usageNode.Exists() {
			usage.InputTokens = gjson.GetBytes(data, usagePath+".input_tokens").Int()
			usage.OutputTokens = gjson.GetBytes(data, usagePath+".output_tokens").Int()
			usage.CachedReadInputTokens = gjson.GetBytes(data, usagePath+".input_tokens_details.cached_tokens").Int()
			usage.CachedTokens = usage.CachedCreationInputTokens + usage.CachedReadInputTokens
			usage.ReasoningTokens = gjson.GetBytes(data, usagePath+".output_tokens_details.reasoning_tokens").Int()
			usage.TotalTokens = gjson.GetBytes(data, usagePath+".total_tokens").Int()
			if usage.TotalTokens <= 0 {
				usage.TotalTokens = usage.InputTokens + usage.OutputTokens
			}
		}
	}
	return usage
}

func ClaudeMessagesParse(raw []byte) (*types.StreamEvent, error) {
	cloneRaw := bytes.TrimRight(raw, "\n")
	line := string(cloneRaw)
	if strings.HasPrefix(line, "event:") {
		return &types.StreamEvent{
			Content:  raw,
			Finished: false,
		}, nil
	}
	// ping之类的
	if !strings.HasPrefix(line, "data:") {
		return &types.StreamEvent{
			Content:  raw,
			Finished: false,
		}, nil
	}
	event := &types.StreamEvent{
		Content:  raw,
		Finished: false,
	}
	// normalize claude  message
	// normalize start message
	dataStr := strings.TrimPrefix(line, "data: ")
	if gjson.Get(dataStr, "type").String() == "message_start" && !gjson.Get(dataStr, "message.usage.input_tokens").Exists() {
		data, _ := sjson.Set(dataStr, "message.usage.input_tokens", 0)
		event.Content = []byte("data: " + data + "\n")
	}

	if gjson.Get(dataStr, "type").String() == "content_block_start" && gjson.Get(dataStr, "content_block.type").String() == "thinking" && !gjson.Get(dataStr, "content_block.thinking").Exists() {
		data, _ := sjson.Set(dataStr, "content_block.thinking", "")
		event.Content = []byte("data: " + data + "\n")
	}
	usage := ParseClaudeUsage([]byte(dataStr))
	event.Usage = usage

	return event, nil
}

func ParseClaudeUsage(data []byte) *types.Usage {
	usage := &types.Usage{}
	for _, usagePath := range []string{"message.usage", "usage"} {
		if usageNode := gjson.GetBytes(data, usagePath); usageNode.Exists() {
			usage.InputTokens = gjson.GetBytes(data, usagePath+".input_tokens").Int()
			usage.OutputTokens = gjson.GetBytes(data, usagePath+".output_tokens").Int()
			usage.CachedCreationInputTokens = gjson.GetBytes(data, usagePath+".cache_creation_input_tokens").Int()
			usage.CachedReadInputTokens = gjson.GetBytes(data, usagePath+".cache_read_input_tokens").Int()
			usage.InputTokens = usage.InputTokens + usage.CachedCreationInputTokens + usage.CachedReadInputTokens
			usage.CachedTokens = usage.CachedCreationInputTokens + usage.CachedReadInputTokens
			usage.TotalTokens = usage.InputTokens + usage.OutputTokens
		}
	}
	return usage
}

func (c *Client) ChatCompletion(ctx context.Context, reqUrl string, originReq *types.Request, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
	return c.executeJSONRequest(ctx, reqUrl, originReq, req, cb, ChatCompletionsParse, ParseChatCompletionsUsage)
}

func (c *Client) Responses(ctx context.Context, reqUrl string, originReq *types.Request, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
	return c.executeJSONRequest(ctx, reqUrl, originReq, req, cb, OpenAIResponsesParse, ParseOpenAIResponsesUsage)
}

func (c *Client) ClaudeMessages(ctx context.Context, reqUrl string, originReq *types.Request, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
	return c.executeJSONRequest(ctx, reqUrl, originReq, req, cb, ClaudeMessagesParse, ParseClaudeUsage)
}

func (c *Client) executeJSONRequest(
	ctx context.Context,
	reqURL string,
	originReq *types.Request,
	req *types.Request,
	cb types.MetricsCallback,
	streamParse types.StreamParseFunc,
	usageParse UsageParse,
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
	metrics.Format = originReq.Format
	metrics.IsStream = req.IsStream
	metrics.StatusCode = httpResp.StatusCode
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
		data, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, err
		}
		//plog.Debug("data:", string(data))
		if exceptedStatus {
			plog.Debug("unexpected status code:", httpResp.StatusCode, string(data))
		}

		if !firstByte.IsZero() {
			metrics.TTFT = metrics.TTFB
		}
		if !firstByte.IsZero() {
			metrics.TransferTime = time.Since(firstByte)
		}
		metrics.TotalTime = time.Since(start)

		usage := usageParse(data)
		if usage != nil {
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
