package cache

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileCache struct {
	CachePath      string
	DirectoryLevel int
	EmbedExpiry    int
	FileSuffix     string
}

func NewFileCache() *FileCache {
	return &FileCache{
		CachePath:      "./runtime/cache",
		DirectoryLevel: 2,
		EmbedExpiry:    120,
		FileSuffix:     ".bin",
	}
}

func (f *FileCache) Get(ctx context.Context, key string) (interface{}, error) {
	fileData, err := f.readFromDisk(key)
	if err != nil {
		return nil, err
	}
	var val interface{}
	decoder := gob.NewDecoder(bytes.NewReader(fileData))
	if err := decoder.Decode(&val); err != nil {
		return nil, err
	}
	return val, nil
}

func (f *FileCache) GetMulti(ctx context.Context, keys []string) ([]interface{}, error) {
	result := make([]interface{}, len(keys))
	for i, key := range keys {
		val, err := f.Get(ctx, key)
		if err == nil {
			result[i] = val
		}
	}
	return result, nil
}

func (f *FileCache) Put(ctx context.Context, key string, val interface{}, timeout time.Duration) error {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	// wrap with expiry
	item := fileCacheItem{Val: val, ExpireAt: time.Now().Add(timeout).Unix()}
	if err := encoder.Encode(item); err != nil {
		return err
	}
	return f.writeToDisk(key, buf.Bytes())
}

func (f *FileCache) Delete(ctx context.Context, key string) error {
	path := f.filePath(key)
	return os.Remove(path)
}

func (f *FileCache) Incr(ctx context.Context, key string) error {
	val, err := f.Get(ctx, key)
	if err != nil {
		return err
	}
	if num, ok := val.(int); ok {
		return f.Put(ctx, key, num+1, 0)
	}
	if num, ok := val.(int64); ok {
		return f.Put(ctx, key, num+1, 0)
	}
	return fmt.Errorf("cache value is not numeric")
}

func (f *FileCache) Decr(ctx context.Context, key string) error {
	val, err := f.Get(ctx, key)
	if err != nil {
		return err
	}
	if num, ok := val.(int); ok {
		return f.Put(ctx, key, num-1, 0)
	}
	if num, ok := val.(int64); ok {
		return f.Put(ctx, key, num-1, 0)
	}
	return fmt.Errorf("cache value is not numeric")
}

func (f *FileCache) IsExist(ctx context.Context, key string) (bool, error) {
	path := f.filePath(key)
	if _, err := os.Stat(path); err != nil {
		return false, nil
	}
	data, err := f.readFromDisk(key)
	if err != nil {
		return false, nil
	}
	var item fileCacheItem
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&item); err != nil {
		return true, nil
	}
	if item.ExpireAt > 0 && time.Now().Unix() > item.ExpireAt {
		_ = os.Remove(path)
		return false, nil
	}
	return true, nil
}

func (f *FileCache) ClearAll(ctx context.Context) error {
	return filepath.Walk(f.CachePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, f.FileSuffix) {
			_ = os.Remove(path)
		}
		return nil
	})
}

func (f *FileCache) StartAndGC(config string) error {
	if f.CachePath != "" {
		_ = os.MkdirAll(f.CachePath, 0755)
	}
	return nil
}

func (f *FileCache) filePath(key string) string {
	// simple hash-based directory structure
	h := hashKey(key)
	dir := f.CachePath
	for i := 0; i < f.DirectoryLevel && i < len(h); i++ {
		dir = filepath.Join(dir, h[i:i+1])
	}
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, h+f.FileSuffix)
}

func (f *FileCache) writeToDisk(key string, data []byte) error {
	path := f.filePath(key)
	return os.WriteFile(path, data, 0644)
}

func (f *FileCache) readFromDisk(key string) ([]byte, error) {
	path := f.filePath(key)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var item fileCacheItem
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&item); err != nil {
		return nil, err
	}
	if item.ExpireAt > 0 && time.Now().Unix() > item.ExpireAt {
		_ = os.Remove(path)
		return nil, errNotFound
	}
	// re-decode the actual value
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	_ = encoder.Encode(item.Val)
	return buf.Bytes(), nil
}

func hashKey(key string) string {
	// simple hash
	h := 0
	for _, c := range key {
		h = h*31 + int(c)
	}
	return fmt.Sprintf("%x", h)
}

type fileCacheItem struct {
	Val      interface{}
	ExpireAt int64
}
