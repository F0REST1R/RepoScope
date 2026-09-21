package github

import (
	"sync"
	"time"
)

type cacheEntry struct {
	body    []byte
	expires time.Time
}
type responseCache struct {
	mu   sync.RWMutex
	ttl  time.Duration
	data map[string]cacheEntry
}

func newResponseCache(ttl time.Duration) *responseCache {
	return &responseCache{ttl: ttl, data: make(map[string]cacheEntry)}
}
func (c *responseCache) get(key string) ([]byte, bool) {
	if c.ttl <= 0 {
		return nil, false
	}
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expires) {
		if ok {
			c.mu.Lock()
			delete(c.data, key)
			c.mu.Unlock()
		}
		return nil, false
	}
	return append([]byte(nil), e.body...), true
}
func (c *responseCache) set(key string, body []byte) {
	if c.ttl <= 0 {
		return
	}
	c.mu.Lock()
	c.data[key] = cacheEntry{append([]byte(nil), body...), time.Now().Add(c.ttl)}
	c.mu.Unlock()
}
