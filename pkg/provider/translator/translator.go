package translator

import (
	"context"

	"github.com/RenaLio/tudou/pkg/provider/types"
)

type RequestTransform func(ctx context.Context, input *types.Request) (*types.Request, error)

type ResponseTransform func(ctx context.Context, req *types.Request, input *types.Response) (*types.Response, error)
