// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStatsCollector(t *testing.T) {
	s := NewStatsCollector()
	assert.NotNil(t, s)
	assert.Equal(t, int64(0), s.TotalRequests)

	model := "test-model"
	duration := 100 * time.Millisecond

	// Record 1st request
	s.RecordRequest(model, duration)
	assert.Equal(t, int64(1), s.TotalRequests)
	assert.Equal(t, int64(1), s.ModelStats[model].RequestCount)
	assert.Equal(t, 100.0, s.ModelStats[model].AvgDurationMs)

	// Record 2nd request (300ms)
	s.RecordRequest(model, 300*time.Millisecond)
	assert.Equal(t, int64(2), s.TotalRequests)
	assert.Equal(t, int64(2), s.ModelStats[model].RequestCount)
	assert.Equal(t, 200.0, s.ModelStats[model].AvgDurationMs)

	// Check GetStats
	stats := s.GetStats()
	assert.Equal(t, int64(2), stats.TotalRequests)
	assert.Equal(t, Version, stats.Version)
	assert.NotNil(t, stats.Models[model])
}
