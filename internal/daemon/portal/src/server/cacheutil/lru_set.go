package cacheutil

import (
	"container/list"
	"sync"
)

// LruSet is a concurrency-safe set with a fixed capacity.
// It evicts the least recently used element when adding a new element at capacity.
type LruSet[T comparable] struct {
	mutex    sync.Mutex
	capacity int
	elements map[T]*list.Element
	order    *list.List
}

// NewLruSet creates an empty LruSet with the given positive capacity.
// It panics when capacity is not positive.
func NewLruSet[T comparable](capacity int) *LruSet[T] {
	if capacity <= 0 {
		panic("cacheutil: LruSet capacity must be positive")
	}
	return new(LruSet[T]{
		capacity: capacity,
		elements: make(map[T]*list.Element, capacity),
		order:    list.New(),
	})
}

// Contains reports whether value is present and marks it as recently used.
func (s *LruSet[T]) Contains(value T) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	element, ok := s.elements[value]
	if ok {
		s.order.MoveToBack(element)
	}
	return ok
}

// Add inserts value or marks an existing value as recently used.
func (s *LruSet[T]) Add(value T) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if element, ok := s.elements[value]; ok {
		s.order.MoveToBack(element)
		return
	}

	s.elements[value] = s.order.PushBack(value)
	if s.order.Len() <= s.capacity {
		return
	}

	oldest := s.order.Front()
	delete(s.elements, oldest.Value.(T))
	s.order.Remove(oldest)
}

// Len returns the number of elements in the set.
func (s *LruSet[T]) Len() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.order.Len()
}
