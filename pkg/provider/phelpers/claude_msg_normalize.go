package phelpers

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/RenaLio/tudou/pkg/provider/common"
	"github.com/RenaLio/tudou/pkg/provider/plog"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

func NormalizeClaudeMsg(ctx context.Context, response *types.Response) (*types.Response, error) {
	if response == nil {
		return nil, errors.New("NormalizeClaudeMsg input response is nil")
	}
	if !response.IsStream {
		return response, nil
	}
	cp := common.CloneResponse(response)
	st := types.NewComplexEventStream(ctx, response.Stream)
	cp.Stream = st
	cp.IsStream = true
	cp.Format = types.FormatClaudeMessages
	//go normalizeClauseSSE(st)
	return cp, nil
}

func normalizeClauseSSE(stream *types.ComplexEventStream) {
	defer stream.CloseCh()
	for {
		select {
		case <-stream.Done():
			return
		default:
		}
		event, err := stream.Pull()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break // 正常结束，触发 defer close
			}
			plog.Error("pull error", err)
			stream.SetErr(err) // 记录底层错误给 Recv 拿去用
			return             // 异常结束，触发 defer close
		}
		chunk := event.Content
		chunk = bytes.TrimSpace(chunk)
		line := string(chunk)

		if len(line) == 0 {
			continue
		}

	}
}
