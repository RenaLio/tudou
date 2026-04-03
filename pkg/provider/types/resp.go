package types

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"
)

type Response struct {
	// HTTP 状态码
	StatusCode int
	// 响应数据格式
	Format Format
	// 提供者
	Provider string
	// 是否是流式响应
	IsStream bool
	// 原始数据
	RawData []byte
	// HTTP 头部
	Header http.Header
	// 流式响应
	Stream StandardStream
	Err    error
}

type StandardStream interface {
	Recv() (*StreamEvent, error)  // 每次调用读取一个标准的增量事件
	Close() error                 // 必须提供的关闭方法，用于释放上游连接
	GetMetrics() *ResponseMetrics // 冗余设计：支持主动拉取指标
}

type StreamEvent struct {
	Content  []byte
	Finished bool
	Usage    *Usage
}

// StreamParseFunc
// 定义解析策略函数：不会收到空内容，收到的数据一定是有信息的.
// 从之解析收集usage等数据，可以一定程度上处理数据转换，如字段映射reasoning -> reasoning_content.
type StreamParseFunc func(rawLine []byte) (*StreamEvent, error)

// BaseStreamIterator 是 StandardStream 接口的标准实现
type BaseStreamIterator struct {
	// 底层依赖
	body io.ReadCloser
	//scanner *bufio.Scanner
	reader  *bufio.Reader
	parseFn StreamParseFunc // 由具体厂商 SDK 注入的解析逻辑

	// 状态与回调
	callback MetricsCallback
	metrics  *ResponseMetrics

	// 内部控制变量
	startTime    time.Time
	isFirstToken bool
	closeOnce    sync.Once // 确保 Close 逻辑只执行一次
}

// 确保 BaseStreamIterator 实现了接口 (编译期检查)
var _ StandardStream = (*BaseStreamIterator)(nil)

// NewStreamIterator 创建一个新的标准流迭代器
func NewStreamIterator(
	body io.ReadCloser,
	req *Request,
	baseMetrics *ResponseMetrics,
	parseFn StreamParseFunc,
	callback MetricsCallback,
	startTime time.Time,
) *BaseStreamIterator {
	it := &BaseStreamIterator{
		body:         body,
		reader:       bufio.NewReader(body),
		parseFn:      parseFn,
		metrics:      baseMetrics,
		startTime:    startTime,
		isFirstToken: true,
	}
	it.callback = callback
	return it
}

// Recv 实现 StandardStream 接口：接收并解析下一个事件。如果遇到错误或 EOF，则返回 nil。大多数情况下，数据是原样返回的(取决于解析策略)
func (it *BaseStreamIterator) Recv() (*StreamEvent, error) {
	// 使用无限循环，直到读出有效事件、遇到错误或遇到 EOF
	for {
		//使用 ReadBytes 逐行读取，它不受 64KB 最大 Token 限制
		line, err := it.reader.ReadBytes('\n')

		// 如果读取到了实际内容(包括换行以及实际数据)
		if len(line) > 0 {
			// 行内容为空格或制表符，原样返回
			if len(bytes.TrimSpace(line)) == 0 {
				event := &StreamEvent{
					Content:  line,
					Finished: false,
				}
				return event, nil
			}
			event, parseErr := it.parseFn(line)
			if parseErr != nil {
				return nil, parseErr // 解析失败，抛出异常终止流
			}

			// 如果解析器认为这一行是有效信息（非 nil）
			if event != nil {
				if event.Usage != nil {
					mergeUsage(&it.metrics.Usage, event.Usage)
				}

				// 拦截首字时间 (TTFT)
				if it.isFirstToken && len(event.Content) > 0 {
					it.metrics.TTFT = time.Since(it.startTime)
					it.isFirstToken = false
				}

				// 成功拿到事件，返回给上层
				return event, nil
			}
		}

		// 检查底层读取错误 (包括正常的结束 io.EOF)
		// ⚠️ 注意：这步必须放在处理完 line 数据之后。
		// 因为 ReadBytes 在读到最后一行如果没有 \n，会同时返回 (最后一部分数据, io.EOF)
		if err != nil {
			if err == io.EOF {
				return nil, io.EOF // 正常读取完毕
			}
			// 网络断开、超时或其他底层错误
			return nil, err
		}

		// 如果 event == nil 且 err == nil，说明是空行或注释，继续下一轮 for 循环读取
	}
}

func (it *BaseStreamIterator) Close() error {
	var closeErr error

	// 保证并发安全，且无论上层调用几次 Close，逻辑只执行一次
	it.closeOnce.Do(func() {
		// 1. 计算最终指标
		it.metrics.TotalTime = time.Since(it.startTime)
		// 避免 TotalTime 比 TTFT 还小（并发精度极个别情况）导致负数
		if it.metrics.TotalTime > it.metrics.TTFT {
			it.metrics.TransferTime = it.metrics.TotalTime - it.metrics.TTFT
		} else {
			it.metrics.TransferTime = 0
		}

		// 2. 触发回调，将指标喂给负载均衡器
		if it.callback != nil {
			it.callback(it.metrics)
		}

		// 3. 关闭底层的 TCP 连接读取器
		if it.body != nil {
			closeErr = it.body.Close()
		}
	})

	return closeErr
}

// GetMetrics 冗余设计：支持 Handler 层主动获取
func (it *BaseStreamIterator) GetMetrics() *ResponseMetrics {
	return it.metrics
}

func mergeUsage(dst *Usage, src *Usage) {
	if dst == nil || src == nil {
		return
	}

	dst.InputTokens = maxInt64(dst.InputTokens, src.InputTokens)
	dst.OutputTokens = maxInt64(dst.OutputTokens, src.OutputTokens)
	dst.TotalTokens = maxInt64(dst.TotalTokens, src.TotalTokens)
	dst.CachedTokens = maxInt64(dst.CachedTokens, src.CachedTokens)
	dst.CachedCreationInputTokens = maxInt64(dst.CachedCreationInputTokens, src.CachedCreationInputTokens)
	dst.CachedReadInputTokens = maxInt64(dst.CachedReadInputTokens, src.CachedReadInputTokens)
	dst.ReasoningTokens = maxInt64(dst.ReasoningTokens, src.ReasoningTokens)

	if dst.TotalTokens == 0 {
		total := dst.InputTokens + dst.OutputTokens
		if total > 0 {
			dst.TotalTokens = total
		}
	}
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
