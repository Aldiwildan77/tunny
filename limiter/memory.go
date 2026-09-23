package limiter

import (
	"fmt"
	"sync"
)

type MemoryLimiter struct {
	storage sync.Map
}

func NewMemoryLimiter() Limiter {
	return &MemoryLimiter{}
}

func (m *MemoryLimiter) Acquire(key string) bool {
	v, ok := m.storage.Load(key)
	if !ok {
		return false
	}

	sem, ok := v.(*Semaphore)
	if !ok {
		return false
	}

	return sem.Acquire()
}

func (m *MemoryLimiter) Release(key string) {
	v, ok := m.storage.Load(key)
	if !ok {
		return
	}

	sem, ok := v.(*Semaphore)
	if !ok {
		return
	}

	sem.Release()
}

func (m *MemoryLimiter) Remove(key string) error {
	m.storage.Delete(key)
	return nil
}

func (m *MemoryLimiter) Set(key string, limit int) error {
	if key == "" {
		return fmt.Errorf("limiter key cannot be empty")
	}

	if limit <= 0 {
		return fmt.Errorf("limiter limit must be greater than zero")
	}

	m.storage.Store(key, NewSemaphore(limit))
	return nil
}
