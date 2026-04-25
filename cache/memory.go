package cache

import (
	"context"
	"strconv"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

type MemoryCache struct {
	client *gocache.Cache
}

func NewMemoryCache(defaultExpiration, cleanupInterval time.Duration) *MemoryCache {
	return &MemoryCache{
		client: gocache.New(defaultExpiration, cleanupInterval),
	}
}

func (m *MemoryCache) Get(ctx context.Context, key string) (interface{}, error) {
	val, found := m.client.Get(key)
	if !found {
		return nil, errNotFound
	}
	return val, nil
}

func (m *MemoryCache) GetMulti(ctx context.Context, keys []string) ([]interface{}, error) {
	result := make([]interface{}, len(keys))
	for i, key := range keys {
		val, found := m.client.Get(key)
		if found {
			result[i] = val
		}
	}
	return result, nil
}

func (m *MemoryCache) Put(ctx context.Context, key string, val interface{}, timeout time.Duration) error {
	m.client.Set(key, val, timeout)
	return nil
}

func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	m.client.Delete(key)
	return nil
}

func (m *MemoryCache) Incr(ctx context.Context, key string) error {
	err := m.client.Increment(key, 1)
	return err
}

func (m *MemoryCache) Decr(ctx context.Context, key string) error {
	err := m.client.Decrement(key, 1)
	return err
}

func (m *MemoryCache) IsExist(ctx context.Context, key string) (bool, error) {
	_, found := m.client.Get(key)
	return found, nil
}

func (m *MemoryCache) ClearAll(ctx context.Context) error {
	m.client.Flush()
	return nil
}

func (m *MemoryCache) StartAndGC(config string) error {
	return nil
}

func parseMemoryConfig(config string) time.Duration {
	if sec, err := strconv.Atoi(config); err == nil && sec > 0 {
		return time.Duration(sec) * time.Second
	}
	return 60 * time.Second
}

var errNotFound = &notFoundError{}

type notFoundError struct{}

func (e *notFoundError) Error() string { return "cache: key not found" }
