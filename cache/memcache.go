package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

type MemcacheCache struct {
	client *memcache.Client
}

func NewMemcacheCache(addr string) *MemcacheCache {
	client := memcache.New(addr)
	return &MemcacheCache{client: client}
}

func (m *MemcacheCache) Get(ctx context.Context, key string) (interface{}, error) {
	item, err := m.client.Get(key)
	if err == memcache.ErrCacheMiss {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	return string(item.Value), nil
}

func (m *MemcacheCache) GetMulti(ctx context.Context, keys []string) ([]interface{}, error) {
	items, err := m.client.GetMulti(keys)
	if err != nil && err != memcache.ErrCacheMiss {
		return nil, err
	}
	result := make([]interface{}, len(keys))
	for i, key := range keys {
		if item, ok := items[key]; ok {
			result[i] = string(item.Value)
		}
	}
	return result, nil
}

func (m *MemcacheCache) Put(ctx context.Context, key string, val interface{}, timeout time.Duration) error {
	var data []byte
	switch v := val.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		data = []byte(fmt.Sprint(v))
	}
	return m.client.Set(&memcache.Item{Key: key, Value: data, Expiration: int32(timeout.Seconds())})
}

func (m *MemcacheCache) Delete(ctx context.Context, key string) error {
	return m.client.Delete(key)
}

func (m *MemcacheCache) Incr(ctx context.Context, key string) error {
	_, err := m.client.Increment(key, 1)
	return err
}

func (m *MemcacheCache) Decr(ctx context.Context, key string) error {
	_, err := m.client.Decrement(key, 1)
	return err
}

func (m *MemcacheCache) IsExist(ctx context.Context, key string) (bool, error) {
	_, err := m.client.Get(key)
	if err == memcache.ErrCacheMiss {
		return false, nil
	}
	return err == nil, err
}

func (m *MemcacheCache) ClearAll(ctx context.Context) error {
	return m.client.FlushAll()
}

func (m *MemcacheCache) StartAndGC(config string) error {
	return nil
}
