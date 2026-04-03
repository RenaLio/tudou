package types

import (
	"context"
	"io"
	"sync"

	"github.com/RenaLio/tudou/pkg/provider/plog"
)

type ComplexEventStream struct {
	ch chan []byte

	ctx    context.Context
	cancel context.CancelFunc
	err    error

	upstream StandardStream

	closeOnce sync.Once
}

var _ StandardStream = (*ComplexEventStream)(nil)

func NewComplexEventStream(ctx context.Context, upstream StandardStream) *ComplexEventStream {
	ctx, cancel := context.WithCancel(ctx)
	return &ComplexEventStream{
		ch:       make(chan []byte, 10),
		ctx:      ctx,
		cancel:   cancel,
		upstream: upstream,
	}
}

func (e *ComplexEventStream) Recv() (*StreamEvent, error) {
	select {
	case <-e.ctx.Done():
		return nil, context.Canceled

	case payload, ok := <-e.ch:
		if !ok {
			// 通道已关，说明 goroutine 跑完了
			if e.err != nil {
				return nil, e.err // 如果底层报错了，把错误抛给网关
			}
			return nil, io.EOF // 正常结束
		}

		// 包装成标准的增量事件交给网关
		return &StreamEvent{
			Content: payload, // 注意：为了兼容你直接拼接的 SSE bytes，事件结构体可以增加一个 RawContent 字段
		}, nil
	}
}

func (e *ComplexEventStream) GetMetrics() *ResponseMetrics {
	return e.upstream.GetMetrics()
}

func (e *ComplexEventStream) Close() error {
	e.closeOnce.Do(func() {
		e.cancel()
	})
	return e.upstream.Close()
}

func (e *ComplexEventStream) Send(payload []byte) error {
	select {
	case <-e.ctx.Done():
		plog.Debug("send", string(payload))
		return context.Canceled
	case e.ch <- payload:
		return nil
	}
}

func (e *ComplexEventStream) SetErr(err error) {
	e.err = err
}

func (e *ComplexEventStream) Pull() (*StreamEvent, error) {
	return e.upstream.Recv()
}

func (e *ComplexEventStream) CloseCh() {
	close(e.ch)
}

func (e *ComplexEventStream) Done() <-chan struct{} {
	return e.ctx.Done()
}
