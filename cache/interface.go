package cache

import (
	"context"
	"time"
)

// Cache 缓存接口
type Cache interface {
	Get(ctx context.Context, key string) (interface{}, error)
	GetMulti(ctx context.Context, keys []string) ([]interface{}, error)
	Put(ctx context.Context, key string, val interface{}, timeout time.Duration) error
	Delete(ctx context.Context, key string) error
	Incr(ctx context.Context, key string) error
	Decr(ctx context.Context, key string) error
	IsExist(ctx context.Context, key string) (bool, error)
	ClearAll(ctx context.Context) error
	StartAndGC(config string) error
}
