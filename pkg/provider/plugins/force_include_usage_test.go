package plugins

import (
	"context"
	"testing"

	"github.com/RenaLio/tudou/pkg/provider/types"
	"github.com/tidwall/gjson"
)

func TestForceIncludeUsageSetsStreamOptionForStreamingChatCompletions(t *testing.T) {
	req := &types.Request{
		Format:   types.FormatChatCompletion,
		IsStream: true,
		Payload:  []byte(`{"model":"gpt-4.1","stream":true}`),
	}

	invoked := false
	next := func(ctx context.Context, got *types.Request, cb types.MetricsCallback) (*types.Response, error) {
		invoked = true
		if !gjson.GetBytes(got.Payload, "stream_options.include_usage").Bool() {
			t.Fatalf("expected stream_options.include_usage=true, got payload=%s", string(got.Payload))
		}
		return &types.Response{}, nil
	}

	_, err := ForceIncludeUsage(next)(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("ForceIncludeUsage returned error: %v", err)
	}
	if !invoked {
		t.Fatal("expected next invoker to be called")
	}
}

func TestForceIncludeUsageOverridesExistingFalseValue(t *testing.T) {
	req := &types.Request{
		Format:   types.FormatChatCompletion,
		IsStream: true,
		Payload:  []byte(`{"model":"gpt-4.1","stream":true,"stream_options":{"include_usage":false}}`),
	}

	next := func(ctx context.Context, got *types.Request, cb types.MetricsCallback) (*types.Response, error) {
		if !gjson.GetBytes(got.Payload, "stream_options.include_usage").Bool() {
			t.Fatalf("expected stream_options.include_usage=true, got payload=%s", string(got.Payload))
		}
		return &types.Response{}, nil
	}

	_, err := ForceIncludeUsage(next)(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("ForceIncludeUsage returned error: %v", err)
	}
}

func TestForceIncludeUsageDoesNotModifyNonStreamingRequest(t *testing.T) {
	req := &types.Request{
		Format:   types.FormatChatCompletion,
		IsStream: false,
		Payload:  []byte(`{"model":"gpt-4.1"}`),
	}

	next := func(ctx context.Context, got *types.Request, cb types.MetricsCallback) (*types.Response, error) {
		if gjson.GetBytes(got.Payload, "stream_options.include_usage").Exists() {
			t.Fatalf("did not expect stream_options.include_usage, got payload=%s", string(got.Payload))
		}
		return &types.Response{}, nil
	}

	_, err := ForceIncludeUsage(next)(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("ForceIncludeUsage returned error: %v", err)
	}
}

func TestForceIncludeUsageDoesNotModifyClaudeMessages(t *testing.T) {
	req := &types.Request{
		Format:   types.FormatClaudeMessages,
		IsStream: true,
		Payload:  []byte(`{"model":"claude-sonnet-4","stream":true}`),
	}

	next := func(ctx context.Context, got *types.Request, cb types.MetricsCallback) (*types.Response, error) {
		if string(got.Payload) != string(req.Payload) {
			t.Fatalf("expected payload unchanged, got payload=%s", string(got.Payload))
		}
		return &types.Response{}, nil
	}

	_, err := ForceIncludeUsage(next)(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("ForceIncludeUsage returned error: %v", err)
	}
}
