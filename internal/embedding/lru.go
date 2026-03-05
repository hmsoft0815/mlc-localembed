// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package embedding

import (
	"container/list"
	"sync"
)

type lruEntry struct {
	key   string
	value []float32
}

type LRUCache struct {
	capacity  int
	items     map[string]*list.Element
	evictList *list.List
	mu        sync.Mutex
}

func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity:  capacity,
		items:     make(map[string]*list.Element),
		evictList: list.New(),
	}
}

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
