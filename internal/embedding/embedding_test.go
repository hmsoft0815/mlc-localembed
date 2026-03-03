// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package embedding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	cacheDir := "./test_cache"
	m := NewManager(cacheDir)
	assert.NotNil(t, m)
	assert.Equal(t, cacheDir, m.cacheDir)
	assert.NotNil(t, m.models)
}

func TestManagerSetOnnxOptions(t *testing.T) {
	m := NewManager("./cache")
	m.SetOnnxOptions(2, 4)
	assert.Equal(t, 2, m.onnxIntraThreads)
	assert.Equal(t, 4, m.onnxInterThreads)
}

func TestManagerClose(t *testing.T) {
	m := NewManager("./cache")
	// Manager should handle empty models map on close
	assert.NotPanics(t, func() {
		m.Close()
	})
	assert.Equal(t, 0, len(m.models))
}
