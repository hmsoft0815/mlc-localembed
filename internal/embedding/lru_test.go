// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package embedding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLRUCache(t *testing.T) {
	cache := NewLRUCache(2)

	v1 := []float32{1.0, 2.0}
	v2 := []float32{3.0, 4.0}
	v3 := []float32{5.0, 6.0}

	cache.Add("k1", v1)
	cache.Add("k2", v2)

	res, ok := cache.Get("k1")
	assert.True(t, ok)
	assert.Equal(t, v1, res)

	cache.Add("k3", v3) // Should evict k2 because k1 was recently used

	res, ok = cache.Get("k2")
	assert.False(t, ok)

	res, ok = cache.Get("k1")
	assert.True(t, ok)
	assert.Equal(t, v1, res)

	res, ok = cache.Get("k3")
	assert.True(t, ok)
	assert.Equal(t, v3, res)
}
