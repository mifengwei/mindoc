package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	prefix string
}

func NewRedisCache(addr, password string, db int, prefix string) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis连接失败: %w", err)
	}
	return &RedisCache{client: client, prefix: prefix}, nil
}

func (r *RedisCache) key(k string) string {
	if r.prefix != "" {
		return r.prefix + ":" + k
	}
	return k
}

func (r *RedisCache) Get(ctx context.Context, key string) (interface{}, error) {
	val, err := r.client.Get(ctx, r.key(key)).Result()
	if err == redis.Nil {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (r *RedisCache) GetMulti(ctx context.Context, keys []string) ([]interface{}, error) {
	pipeKeys := make([]string, len(keys))
	for i, k := range keys {
		pipeKeys[i] = r.key(k)
	}
	vals, err := r.client.MGet(ctx, pipeKeys...).Result()
	if err != nil {
		return nil, err
	}
	return vals, nil
}

func (r *RedisCache) Put(ctx context.Context, key string, val interface{}, timeout time.Duration) error {
	data, ok := val.(string)
	if !ok {
		b, err := json.Marshal(val)
		if err != nil {
			return err
		}
		data = string(b)
	}
	return r.client.Set(ctx, r.key(key), data, timeout).Err()
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.key(key)).Err()
}

func (r *RedisCache) Incr(ctx context.Context, key string) error {
	return r.client.Incr(ctx, r.key(key)).Err()
}

func (r *RedisCache) Decr(ctx context.Context, key string) error {
	return r.client.Decr(ctx, r.key(key)).Err()
}

func (r *RedisCache) IsExist(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, r.key(key)).Result()
	return n > 0, err
}

func (r *RedisCache) ClearAll(ctx context.Context) error {
	if r.prefix != "" {
		iter := r.client.Scan(ctx, 0, r.prefix+":*", 0).Iterator()
		for iter.Next(ctx) {
			_ = r.client.Del(ctx, iter.Val()).Err()
		}
		return iter.Err()
	}
	return r.client.FlushDB(ctx).Err()
}

func (r *RedisCache) StartAndGC(config string) error {
	return nil
}
