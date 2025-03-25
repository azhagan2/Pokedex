package pokecache

import (
	"sync"
	"time"
)

type Cache struct {
	items map[string]cacheEntry
	mu    sync.RWMutex
}

type cacheEntry struct {
	createdAt time.Time
	val       []byte
}

func NewCache(interval time.Duration) *Cache {
	cache := &Cache{
		items: make(map[string]cacheEntry),
	}
	cache.reapLoop(interval)
	return cache
}

func (c *Cache) Add(key string, val []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheEntry{
		val:       val,
		createdAt: time.Now(),
	}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	return item.val, true
}

func (c *Cache) reapLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			<-ticker.C
			c.mu.Lock()
			now := time.Now()

			// Create a list of keys to delete to avoid modifying the map during iteration
			var keysToDelete []string
			for k, v := range c.items {
				if now.Sub(v.createdAt) > interval {
					keysToDelete = append(keysToDelete, k)
				}
			}

			// Delete the keys after iteration
			for _, k := range keysToDelete {
				delete(c.items, k)
			}

			c.mu.Unlock()
		}
	}()
}
