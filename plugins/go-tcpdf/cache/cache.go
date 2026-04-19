// Package cache provides a simple key-value document cache.
package cache

import "sync"

// Cache is a thread-safe key→value store.
type Cache struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// New creates a new Cache.
func New() *Cache {
	return &Cache{data: make(map[string]interface{})}
}

// Set stores a value.
func (c *Cache) Set(key string, val interface{}) {
	c.mu.Lock()
	c.data[key] = val
	c.mu.Unlock()
}

// Get retrieves a value.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	v, ok := c.data[key]
	c.mu.RUnlock()
	return v, ok
}

// Delete removes a key.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
}

// Clear empties the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	c.data = make(map[string]interface{})
	c.mu.Unlock()
}
