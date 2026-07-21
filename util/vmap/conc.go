package vmap

import "sync"

// MutexMap is a generic map protected by sync.RWMutex.
type MutexMap[K comparable, V any] struct {
	mu   sync.RWMutex
	dict map[K]V
}

// NewMutexMap creates an empty map protected by an RWMutex.
func NewMutexMap[K comparable, V any]() *MutexMap[K, V] {
	return &MutexMap[K, V]{
		dict: map[K]V{},
	}
}

// Store associates value with key.
func (m *MutexMap[K, V]) Store(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.dict[key] = value
}

// Load returns the value for key and reports whether it exists.
func (m *MutexMap[K, V]) Load(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, ok := m.dict[key]
	return value, ok
}

// Delete removes key.
func (m *MutexMap[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.dict, key)
}

// LoadOrStore returns the existing value for key if present.
// Otherwise it stores and returns value; loaded reports which case occurred.
func (m *MutexMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	actual, ok := m.dict[key]
	if ok {
		return actual, true
	}

	m.dict[key] = value
	return value, false
}

// LoadAndDelete returns and removes the value for key when present.
func (m *MutexMap[K, V]) LoadAndDelete(key K) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	value, ok := m.dict[key]
	if ok {
		delete(m.dict, key)
	}
	return value, ok
}

// Range calls iteratee for a snapshot of each entry until iteratee returns false.
// The map lock is not held while iteratee runs.
func (m *MutexMap[K, V]) Range(iteratee func(K, V) bool) {
	m.mu.RLock()
	entries := make([]struct {
		key   K
		value V
	}, 0, len(m.dict))
	for key, value := range m.dict {
		entries = append(entries, struct {
			key   K
			value V
		}{key: key, value: value})
	}
	m.mu.RUnlock()

	for _, entry := range entries {
		if !iteratee(entry.key, entry.value) {
			return
		}
	}
}

// SyncMap is a generic wrapper around the standard library sync.Map.
type SyncMap[K comparable, V any] struct {
	dict sync.Map
}

// NewSyncMap creates an empty typed sync.Map wrapper.
func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{}
}

// Store associates value with key.
func (m *SyncMap[K, V]) Store(key K, value V) {
	m.dict.Store(key, value)
}

// Load returns the value for key and reports whether it exists.
func (m *SyncMap[K, V]) Load(key K) (V, bool) {
	value, ok := m.dict.Load(key)
	if !ok {
		var zero V
		return zero, false
	}
	return value.(V), true
}

// Delete removes key.
func (m *SyncMap[K, V]) Delete(key K) {
	m.dict.Delete(key)
}

// LoadOrStore returns the existing value for key if present.
// Otherwise it stores and returns value; loaded reports which case occurred.
func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	actual, loaded := m.dict.LoadOrStore(key, value)
	return actual.(V), loaded
}

// LoadAndDelete returns and removes the value for key when present.
func (m *SyncMap[K, V]) LoadAndDelete(key K) (V, bool) {
	value, loaded := m.dict.LoadAndDelete(key)
	if !loaded {
		var zero V
		return zero, false
	}
	return value.(V), true
}

// Range calls iteratee for each entry until iteratee returns false.
func (m *SyncMap[K, V]) Range(iteratee func(K, V) bool) {
	m.dict.Range(func(key, value any) bool {
		return iteratee(key.(K), value.(V))
	})
}
