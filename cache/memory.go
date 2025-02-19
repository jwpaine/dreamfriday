package cache

import "sync"

// MemoryCache implements Cache using sync.Map
type MemoryCache struct {
	store sync.Map
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{}
}

func (c *MemoryCache) Set(key string, value interface{}) {
	c.store.Store(key, value)
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	return c.store.Load(key)
}

func (c *MemoryCache) Delete(key string) {
	c.store.Delete(key)
}
