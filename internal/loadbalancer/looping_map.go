package loadbalancer

import "sync"

// LoopingMap 是一个并发安全的、只存储固定 N 个元素的 Map
// 当超出容量时，会采用 FIFO (先进先出) 的策略淘汰最早的元素
type LoopingMap[K comparable, V any] struct {
	data     map[K]V
	keys     []K
	head     int
	capacity int
	mu       sync.RWMutex
}

// NewLoopingMap 创建并初始化一个 LoopingMap
func NewLoopingMap[K comparable, V any](capacity int) *LoopingMap[K, V] {
	if capacity <= 0 {
		panic("capacity must be greater than 0")
	}
	return &LoopingMap[K, V]{
		data:     make(map[K]V, capacity),
		keys:     make([]K, 0, capacity),
		head:     0,
		capacity: capacity,
	}
}

// Put 插入或更新一个键值对
func (m *LoopingMap[K, V]) Put(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果 key 已经存在，只更新 value，不改变其在 FIFO 队列中的淘汰顺序
	if _, exists := m.data[key]; exists {
		m.data[key] = value
		return
	}

	// 如果达到了容量上限，需要淘汰最老的元素 (head 所指向的元素)
	if len(m.keys) == m.capacity {
		oldestKey := m.keys[m.head]
		delete(m.data, oldestKey) // 从 map 中移除最老的元素

		m.keys[m.head] = key // 在环形缓冲区中覆盖旧 key
		m.data[key] = value  // 存入新值

		// 移动 head 指针，使用取模实现环形循环
		m.head = (m.head + 1) % m.capacity
	} else {
		// 如果还没达到容量上限，直接追加
		m.keys = append(m.keys, key)
		m.data[key] = value
	}
}

// Get 获取指定 key 的值
func (m *LoopingMap[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, exists := m.data[key]
	return val, exists
}

// Delete 删除一个键值对
// 注意：在环形缓冲区中删除中间元素需要移动切片，时间复杂度为 O(N)
func (m *LoopingMap[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.data[key]; !exists {
		return
	}

	delete(m.data, key)

	// 从 keys 队列中找到并移除该 key
	for i, k := range m.keys {
		if k == key {
			// 从切片中移除该元素
			m.keys = append(m.keys[:i], m.keys[i+1:]...)

			// 修正 head 指针
			if m.head > i {
				m.head--
			} else if m.head == len(m.keys) {
				// 如果 head 刚好等于当前长度（越界），重置为 0
				m.head = 0
			}
			break
		}
	}
}

// Len 返回当前存储的元素数量
func (m *LoopingMap[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}
