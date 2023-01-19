package server

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("raft-otel-mapstorage")

type StorageConfig struct {
	delayRate     float64
	delayDuration time.Duration
	count         int
}

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
	m      map[string][]byte
	config *StorageConfig
}

func NewMapStorage() *MapStorage {
	m := make(map[string][]byte)
	ms := &MapStorage{
		m: m,
	}
	ms.loadConfig()
	return ms
}

func (ms *MapStorage) Get(ctx context.Context, key string) ([]byte, bool) {
	_, span := tracer.Start(ctx, "MapStorage.Get")
	defer span.End()
	ms.Lock()
	ms.config.count++
	if ms.config.delayRate > 0 {
		if ms.config.count%int(1/ms.config.delayRate) == 0 {
			time.Sleep(ms.config.delayDuration)
		}
	}
	v, found := ms.m[key]
	ms.Unlock()
	return v, found
}

func (ms *MapStorage) Set(ctx context.Context, key string, value []byte) {
	_, span := tracer.Start(ctx, "MapStorage.Set")
	ms.Lock()
	ms.config.count++
	if ms.config.delayRate > 0 {
		if ms.config.count%int(1/ms.config.delayRate) == 0 {
			time.Sleep(ms.config.delayDuration)
		}
	}
	ms.m[key] = value
	ms.Unlock()
	span.End()
}

func (ms *MapStorage) HasData(ctx context.Context) bool {
	_, span := tracer.Start(ctx, "MapStorage.HasData")
	defer span.End()
	ms.Lock()
	defer ms.Unlock()
	return len(ms.m) > 0
}

func (ms *MapStorage) loadConfig() {
	ms.config = &StorageConfig{
		delayRate:     0.0,
		delayDuration: 0,
	}

	delayRate := os.Getenv("STORAGE_SERVER_DELAY_RATE")
	delayDuration := os.Getenv("STORAGE_SERVER_DELAY_MS")

	if v, err := strconv.ParseFloat(delayRate, 64); err == nil {
		ms.config.delayRate = v
	}

	if v, err := strconv.ParseInt(delayDuration, 10, 64); err == nil && delayDuration != "" {
		ms.config.delayDuration = time.Duration(v) * time.Millisecond
	}
}
