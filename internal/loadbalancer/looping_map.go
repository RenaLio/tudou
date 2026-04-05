package loadbalancer

import "sync"

type LoopingMap[K comparable, V any] struct {
	data     map[K]V
	keys     []K
	head     int
	capacity int
	mu       sync.RWMutex
}
