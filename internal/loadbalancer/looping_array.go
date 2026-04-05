package loadbalancer

import (
	"sync"
)

// LoopingArray 线程安全的循环数组 (Ring Buffer)
type LoopingArray[T any] struct {
	array    []T
	head     int // 下一个写入的位置
	count    int // 当前实际包含的元素数量
	capacity int // 最大容量
	mu       sync.RWMutex
}

// NewLoopingArray 初始化循环数组
func NewLoopingArray[T any](capacity int) *LoopingArray[T] {
	if capacity <= 0 {
		panic("capacity must be greater than 0")
	}
	return &LoopingArray[T]{
		array:    make([]T, capacity),
		head:     0,
		count:    0,
		capacity: capacity,
	}
}

// Push 添加一个元素，如果满了会覆盖最老的元素
func (la *LoopingArray[T]) Push(value T) {
	la.mu.Lock()
	defer la.mu.Unlock()

	la.array[la.head] = value
	la.head = (la.head + 1) % la.capacity

	// 维护实际元素数量
	if la.count < la.capacity {
		la.count++
	}
}

// Get 获取指定位置的元素 (0 表示最老的元素, count-1 表示最新的元素)
// 增加 bool 返回值，类似 map 的 ok idiom，防止越界 panic
func (la *LoopingArray[T]) Get(index int) (T, bool) {
	la.mu.RLock()
	defer la.mu.RUnlock()

	var zero T // 泛型零值
	if index < 0 || index >= la.count {
		return zero, false // 索引不合法或越界
	}

	// 计算真实的物理索引
	// 如果数组还没满，最老的元素在索引 0
	// 如果数组满了，最老的元素在 head 位置
	oldestIndex := 0
	if la.count == la.capacity {
		oldestIndex = la.head
	}

	actualIndex := (oldestIndex + index) % la.capacity
	return la.array[actualIndex], true
}

// GetCount 返回当前实际元素数量
func (la *LoopingArray[T]) GetCount() int {
	la.mu.RLock()
	defer la.mu.RUnlock()
	return la.count
}

// GetCapacity 返回最大容量
func (la *LoopingArray[T]) GetCapacity() int {
	la.mu.RLock()
	defer la.mu.RUnlock()
	return la.capacity
}

// List 按照时间顺序（最老 -> 最新）返回所有有效元素的副本
func (la *LoopingArray[T]) List() []T {
	la.mu.RLock()
	defer la.mu.RUnlock()

	// 预分配切片以提高性能 (长度为 count)
	result := make([]T, 0, la.count)

	if la.count < la.capacity {
		// 还没满，直接拷贝 0 到 count 的部分
		result = append(result, la.array[:la.count]...)
	} else {
		// 已经满了并且发生过覆盖，分为两段拷贝
		result = append(result, la.array[la.head:]...)
		result = append(result, la.array[:la.head]...)
	}

	return result
}

// ForEach 按最老 -> 最新遍历；fn 返回 false 可提前中断
func (la *LoopingArray[T]) ForEach(fn func(i int, v T) bool) {
	if fn == nil {
		return
	}

	la.mu.RLock()
	defer la.mu.RUnlock()

	oldest := 0
	if la.count == la.capacity {
		oldest = la.head
	}

	for i := 0; i < la.count; i++ {
		idx := (oldest + i) % la.capacity
		if !fn(i, la.array[idx]) {
			return
		}
	}
}
