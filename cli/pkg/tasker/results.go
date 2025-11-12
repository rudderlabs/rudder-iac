package tasker

import "sync"

type Results[T any] struct {
	sync.RWMutex
	data map[string]T
}

func NewResults[T any]() *Results[T] {
	return &Results[T]{
		data: make(map[string]T),
	}
}

func (r *Results[T]) Store(k string, value T) {
	r.Lock()
	defer r.Unlock()

	r.data[k] = value
}

func (r *Results[T]) Get(k string) (T, bool) {
	r.RLock()
	defer r.RUnlock()

	val, ok := r.data[k]
	return val, ok
}

func (r *Results[T]) GetKeys() []string {
	r.RLock()
	defer r.RUnlock()

	keys := make([]string, 0, len(r.data))
	for k := range r.data {
		keys = append(keys, k)
	}
	return keys
}
