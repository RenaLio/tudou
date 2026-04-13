package cache

import (
	"context"
	"time"

	"github.com/allegro/bigcache/v3" // 注意：建议使用 v3 版本
	"github.com/goccy/go-json"
)

type JsonCache struct {
	client *bigcache.BigCache
}

func DefaultConfig() bigcache.Config {
	return bigcache.Config{
		Shards:             2048,
		LifeWindow:         10 * time.Minute,
		CleanWindow:        5 * time.Minute,
		MaxEntriesInWindow: 1000 * 10 * 60,
		MaxEntrySize:       500,
		Verbose:            true,
		HardMaxCacheSize:   256,
	}
}

func New(ctx context.Context, config bigcache.Config) (*JsonCache, error) {
	client, err := bigcache.New(ctx, config)
	if err != nil {
		return nil, err
	}
	return &JsonCache{client: client}, nil
}

func Get[T any](c *JsonCache, key string) (T, error) {
	var result T
	entry, err := c.client.Get(key)
	if err != nil {
		// 这里可能返回 bigcache.ErrEntryNotFound
		return result, err
	}

	err = json.Unmarshal(entry, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func GetBytes(c *JsonCache, key string) ([]byte, error) {
	return c.client.Get(key)
}

func Set[T any](c *JsonCache, key string, value T) error {
	entry, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(key, entry)
}

func SetBytes(c *JsonCache, key string, value []byte) error {
	return c.client.Set(key, value)
}

func (c *JsonCache) Delete(key string) error {
	return c.client.Delete(key)
}

func (c *JsonCache) Clear() error {
	return c.client.Reset()
}
