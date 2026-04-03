package translator

import (
	"context"
	"fmt"
	"sync"

	"github.com/RenaLio/tudou/pkg/provider/types"
)

type Registry struct {
	mu        sync.RWMutex
	requests  map[types.Format]map[types.Format]RequestTransform
	responses map[types.Format]map[types.Format]ResponseTransform
}

func NewRegistry() *Registry {
	return &Registry{
		requests:  make(map[types.Format]map[types.Format]RequestTransform),
		responses: make(map[types.Format]map[types.Format]ResponseTransform),
	}
}

func (r *Registry) RegisterRequest(source, target types.Format, transform RequestTransform) {
	if transform == nil || source == target {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.requests[source]; !ok {
		r.requests[source] = make(map[types.Format]RequestTransform)
	}
	r.requests[source][target] = transform
}

func (r *Registry) RegisterResponse(source, target types.Format, transform ResponseTransform) {
	if transform == nil || source == target {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.responses[source]; !ok {
		r.responses[source] = make(map[types.Format]ResponseTransform)
	}
	r.responses[source][target] = transform
}

func (r *Registry) CanTransformRequest(source, target types.Format) bool {
	if source == "" || target == "" {
		return false
	}
	if source == target {
		return true
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	toMap, ok := r.requests[source]
	if !ok {
		return false
	}
	_, ok = toMap[target]
	return ok
}

func (r *Registry) CanTransformResponse(source, target types.Format) bool {
	if source == "" || target == "" {
		return false
	}
	if source == target {
		return true
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	toMap, ok := r.responses[source]
	if !ok {
		return false
	}
	_, ok = toMap[target]
	return ok
}

func (r *Registry) TransformRequest(ctx context.Context, input *types.Request, target types.Format) (*types.Request, error) {
	if input == nil {
		return nil, fmt.Errorf("request is nil")
	}
	if target == "" {
		return nil, fmt.Errorf("target format is required")
	}

	source := input.Format
	if source == target {
		return input, nil
	}

	transform, ok := r.getRequestTransform(source, target)
	if !ok {
		return nil, fmt.Errorf("unsupported request transform: %s -> %s", source, target)
	}

	return transform(ctx, input)
}

func (r *Registry) TransformResponse(ctx context.Context, req *types.Request, input *types.Response, target types.Format) (*types.Response, error) {
	if input == nil {
		return nil, fmt.Errorf("response is nil")
	}
	if target == "" {
		return nil, fmt.Errorf("target format is required")
	}

	source := input.Format
	if source == target {
		return input, nil
	}

	transform, ok := r.getResponseTransform(source, target)
	if !ok {
		return nil, fmt.Errorf("unsupported response transform: %s -> %s", source, target)
	}

	return transform(ctx, req, input)
}

func (r *Registry) getRequestTransform(source, target types.Format) (RequestTransform, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	toMap, ok := r.requests[source]
	if !ok {
		return nil, false
	}
	transform, ok := toMap[target]
	return transform, ok
}

func (r *Registry) getResponseTransform(source, target types.Format) (ResponseTransform, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	toMap, ok := r.responses[source]
	if !ok {
		return nil, false
	}
	transform, ok := toMap[target]
	return transform, ok
}

var (
	defaultRegistry     *Registry
	defaultRegistryOnce sync.Once
)

func DefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		reg := NewRegistry()
		defaultRegistry = reg
	})
	return defaultRegistry
}

func TransformRequest(ctx context.Context, input *types.Request, target types.Format) (*types.Request, error) {
	return DefaultRegistry().TransformRequest(ctx, input, target)
}

func TransformResponse(ctx context.Context, req *types.Request, input *types.Response, target types.Format) (*types.Response, error) {
	return DefaultRegistry().TransformResponse(ctx, req, input, target)
}

func CanTransform(source, target types.Format) bool {
	return DefaultRegistry().CanTransformRequest(source, target)
}

func CanTransformResponse(source, target types.Format) bool {
	return DefaultRegistry().CanTransformResponse(source, target)
}
