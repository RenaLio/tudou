package plugins

import (
	"context"
	"reflect"
	"testing"

	"github.com/RenaLio/tudou/pkg/provider/types"
)

func TestApplyPluginsWrapsInDeclaredOrder(t *testing.T) {
	var trace []string

	base := func(ctx context.Context, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
		trace = append(trace, "base")
		return &types.Response{}, nil
	}

	p1 := func(next types.Invoker) types.Invoker {
		return func(ctx context.Context, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
			trace = append(trace, "p1-before")
			resp, err := next(ctx, req, cb)
			trace = append(trace, "p1-after")
			return resp, err
		}
	}
	p2 := func(next types.Invoker) types.Invoker {
		return func(ctx context.Context, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
			trace = append(trace, "p2-before")
			resp, err := next(ctx, req, cb)
			trace = append(trace, "p2-after")
			return resp, err
		}
	}

	invoker := ApplyPlugins(base, p1, p2)
	if _, err := invoker(context.Background(), &types.Request{}, nil); err != nil {
		t.Fatalf("invoker returned error: %v", err)
	}

	want := []string{"p1-before", "p2-before", "base", "p2-after", "p1-after"}
	if !reflect.DeepEqual(trace, want) {
		t.Fatalf("unexpected trace: got=%v want=%v", trace, want)
	}
}

func TestApplyPluginsWithoutPluginsReturnsBaseInvoker(t *testing.T) {
	called := false
	base := func(ctx context.Context, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
		called = true
		return &types.Response{}, nil
	}

	invoker := ApplyPlugins(base)
	if _, err := invoker(context.Background(), &types.Request{}, nil); err != nil {
		t.Fatalf("invoker returned error: %v", err)
	}
	if !called {
		t.Fatal("expected base invoker to be called")
	}
}
