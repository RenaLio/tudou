package types

import (
	"context"
)

type Invoker func(ctx context.Context, req *Request, cb MetricsCallback) (*Response, error)

type Plugin func(next Invoker) Invoker
