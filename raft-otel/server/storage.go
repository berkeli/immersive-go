package server

import "sync"

// Storage is an interface implemented by stable storage providers.
type Storage interface {
	Set(key string, value []byte)

	Get(key string) ([]byte, bool)

	// HasData returns true iff any Sets were made on this Storage.
	HasData() bool
}

// MapStorage is a simple in-memory implementation of Storage for testing.
type MapStorage struct {
	sync.Mutex
	m map[string][]byte
}

func NewMapStorage() *MapStorage {
	m := make(map[string][]byte)
	return &MapStorage{
		m: m,
	}
}

func (ms *MapStorage) Get(key string) ([]byte, bool) {
	ms.Lock()
	defer ms.Unlock()
	v, found := ms.m[key]
	return v, found
}

func (ms *MapStorage) Set(key string, value []byte) {
	ms.Lock()
	defer ms.Unlock()
	ms.m[key] = value
}

func (ms *MapStorage) HasData() bool {
	ms.Lock()
	defer ms.Unlock()
	return len(ms.m) > 0
}
