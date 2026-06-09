package plugins

import (
	"context"

	"github.com/RenaLio/tudou/pkg/provider/types"
	"github.com/tidwall/sjson"
)

// ForceIncludeUsage enforces stream_options.include_usage=true for
// streaming Chat Completions requests. Other formats, including Claude
// Messages, are passed through unchanged.
func ForceIncludeUsage(next types.Invoker) types.Invoker {
	return func(ctx context.Context, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
		if req != nil && req.IsStream && req.Format == types.FormatChatCompletion && len(req.Payload) > 0 {
			if payload, err := sjson.SetBytes(req.Payload, "stream_options.include_usage", true); err == nil {
				req.Payload = payload
			}
		}
		return next(ctx, req, cb)
	}
}
