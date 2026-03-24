// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package api

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestStatsCollector verifies that basic request recording and average duration
// calculations are accurate for a single model.
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

// TestStatsCollector_MultiModel ensures that metrics are correctly separated
// and aggregated when multiple different models are used.
func TestStatsCollector_MultiModel(t *testing.T) {
	s := NewStatsCollector()

	s.RecordRequest("model-A", 100*time.Millisecond)
	s.RecordRequest("model-B", 200*time.Millisecond)
	s.RecordRequest("model-A", 300*time.Millisecond)

	stats := s.GetStats()
	assert.Equal(t, int64(3), stats.TotalRequests)
	assert.Equal(t, int64(2), stats.Models["model-A"].RequestCount)
	assert.Equal(t, int64(1), stats.Models["model-B"].RequestCount)
	assert.Equal(t, 200.0, stats.Models["model-A"].AvgDurationMs)
	assert.Equal(t, 200.0, stats.Models["model-B"].AvgDurationMs)
}

// TestStatsCollector_Concurrency validates that the collector is thread-safe
// and maintains data integrity under high concurrent load.
func TestStatsCollector_Concurrency(t *testing.T) {
	s := NewStatsCollector()
	var wg sync.WaitGroup

	numRoutines := 10
	requestsPerRoutine := 100

	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			model := "model-concurrent"
			if id%2 == 0 {
				model = "model-even"
			}
			for j := 0; j < requestsPerRoutine; j++ {
				s.RecordRequest(model, 10*time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	stats := s.GetStats()
	expectedTotal := int64(numRoutines * requestsPerRoutine)
	assert.Equal(t, expectedTotal, stats.TotalRequests)

	// Total for specific models should also be correct
	countConcurrent := stats.Models["model-concurrent"].RequestCount
	countEven := stats.Models["model-even"].RequestCount
	assert.Equal(t, int64(numRoutines*requestsPerRoutine), countConcurrent+countEven)
}
