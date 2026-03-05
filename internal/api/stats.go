// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package api

import (
	"sync"
	"time"
)

type StatsCollector struct {
	mu            sync.RWMutex
	StartTime     time.Time
	TotalRequests int64
	ModelStats    map[string]*ModelMetrics
}

type ModelMetrics struct {
	RequestCount  int64         `json:"request_count"`
	TotalDuration time.Duration `json:"total_duration_ns"`
	AvgDurationMs float64       `json:"avg_duration_ms"`
}

func NewStatsCollector() *StatsCollector {
	return &StatsCollector{
		StartTime:  time.Now(),
		ModelStats: make(map[string]*ModelMetrics),
	}
}

func (s *StatsCollector) RecordRequest(model string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalRequests++
	m, ok := s.ModelStats[model]
	if !ok {
		m = &ModelMetrics{}
		s.ModelStats[model] = m
	}

	m.RequestCount++
	m.TotalDuration += duration
	m.AvgDurationMs = float64(m.TotalDuration.Milliseconds()) / float64(m.RequestCount)
}

type GlobalStats struct {
	UptimeSeconds int64                    `json:"uptime_seconds"`
	Version       string                   `json:"version"`
	TotalRequests int64                    `json:"total_requests"`
	Models        map[string]*ModelMetrics `json:"models"`
}

func (s *StatsCollector) GetStats() GlobalStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return GlobalStats{
		UptimeSeconds: int64(time.Since(s.StartTime).Seconds()),
		Version:       Version,
		TotalRequests: s.TotalRequests,
		Models:        s.ModelStats,
	}
}
