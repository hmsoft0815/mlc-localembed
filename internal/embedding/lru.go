// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package embedding

import (
	"container/list"
	"sync"
)

// lruEntry holds a single key-value pair in the LRU cache.
type lruEntry struct {
	key   string
	value []float32
}

// LRUCache implements a thread-safe Least Recently Used cache for embedding vectors.
// It uses a map for O(1) lookups and a doubly-linked list for O(1) eviction of the oldest items.
type LRUCache struct {
	capacity  int
	items     map[string]*list.Element
	evictList *list.List
	mu        sync.Mutex
}

// NewLRUCache creates a new LRUCache with the specified maximum capacity.
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity:  capacity,
		items:     make(map[string]*list.Element),
		evictList: list.New(),
	}
}

// Get retrieves an embedding vector from the cache.
// It returns a copy of the cached vector to prevent external modification.
// If the key exists, it is moved to the front of the eviction list (marking it as most recently used).
func (c *LRUCache) Get(key string) ([]float32, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		// Return a copy to prevent external modification of cached data
		val := ent.Value.(*lruEntry).value
		res := make([]float32, len(val))
		copy(res, val)
		return res, true
	}
	return nil, false
}

// Add inserts or updates an embedding vector in the cache.
// If the cache exceeds its capacity, the least recently used item is evicted.
func (c *LRUCache) Add(key string, value []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		ent.Value.(*lruEntry).value = value
		return
	}

	ent := &lruEntry{key, value}
	entry := c.evictList.PushFront(ent)
	c.items[key] = entry

	if c.evictList.Len() > c.capacity {
		c.removeOldest()
	}
}

func (c *LRUCache) removeOldest() {
	ent := c.evictList.Back()
	if ent != nil {
		c.evictList.Remove(ent)
		kv := ent.Value.(*lruEntry)
		delete(c.items, kv.key)
	}
}
