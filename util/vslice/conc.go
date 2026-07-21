package vslice

import "sync"

// MutexSlice is a generic slice protected by sync.RWMutex.
type MutexSlice[E any] struct {
	mu   sync.RWMutex
	list []E
}

// NewMutexSlice creates a synchronized slice initialized with a copy of items.
func NewMutexSlice[E any](items ...E) *MutexSlice[E] {
	return &MutexSlice[E]{
		list: append([]E{}, items...),
	}
}

// Append adds items to the end of the slice.
func (s *MutexSlice[E]) Append(items ...E) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.list = append(s.list, items...)
}

// Load returns the item at index and reports whether the index is valid.
func (s *MutexSlice[E]) Load(index int) (E, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if index < 0 || index >= len(s.list) {
		var zero E
		return zero, false
	}
	return s.list[index], true
}

// Store replaces the item at index and reports whether the index is valid.
func (s *MutexSlice[E]) Store(index int, item E) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index < 0 || index >= len(s.list) {
		return false
	}
	s.list[index] = item
	return true
}

// Delete removes and returns the item at index when the index is valid.
func (s *MutexSlice[E]) Delete(index int) (E, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index < 0 || index >= len(s.list) {
		var zero E
		return zero, false
	}
	item := s.list[index]
	s.list = append(s.list[:index], s.list[index+1:]...)
	return item, true
}

// Len returns the current number of items.
func (s *MutexSlice[E]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.list)
}

// Snapshot returns a copy of the current items.
func (s *MutexSlice[E]) Snapshot() []E {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return append([]E{}, s.list...)
}

// Range calls iteratee for a snapshot of the items until iteratee returns false.
func (s *MutexSlice[E]) Range(iteratee func(int, E) bool) {
	list := s.Snapshot()
	for index, item := range list {
		if !iteratee(index, item) {
			return
		}
	}
}
