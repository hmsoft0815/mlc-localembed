// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"mlc-localembed/internal/embedding"

	"github.com/gin-gonic/gin"
)

// EmbedRequest follows Ollama's /api/embed request structure
type EmbedRequest struct {
	Model string      `json:"model" binding:"required"`
	Input interface{} `json:"input" binding:"required"` // Can be string or []string
}

// EmbedResponse follows Ollama's /api/embed response structure
type EmbedResponse struct {
	Model      string      `json:"model"`
	Embeddings [][]float32 `json:"embeddings"`
}

// TagResponse follows Ollama's /api/tags response structure
type TagResponse struct {
	Models []ModelDetails `json:"models"`
}

// SimilarityRequest for the test API
type SimilarityRequest struct {
	Model     string   `json:"model" binding:"required"`
	Query     string   `json:"query" binding:"required"`
	Documents []string `json:"documents" binding:"required"`
}

// SimilarityResponse for the test API
type SimilarityResponse struct {
	Model  string             `json:"model"`
	Scores []SimilarityResult `json:"scores"`
}

type SimilarityResult struct {
	Document string  `json:"document"`
	Score    float32 `json:"score"`
}

type ModelDetails struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    Details   `json:"details"`
}

type Details struct {
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

type ConfigModel struct {
	Name        string `yaml:"name"`
	Dim         int    `yaml:"dim"`
	Description string `yaml:"description"`
	Enabled     *bool  `yaml:"enabled"`
}

type Handler struct {
	manager      *embedding.Manager
	configModels []ConfigModel
	defaultModel string
	stats        *StatsCollector
	concurrency  chan struct{}
}

// HandleEmbedFaker returns dummy embeddings for testing
func (h *Handler) HandleEmbedFaker(c *gin.Context) {
       var req EmbedRequest
       if err := c.ShouldBindJSON(&req); err != nil {
	       c.JSON(400, gin.H{"error": err.Error()})
	       return
       }

       // Accept string or array of strings
       var inputs []string
       switch v := req.Input.(type) {
       case string:
	       inputs = []string{v}
       case []interface{}:
	       for _, item := range v {
		       if s, ok := item.(string); ok {
			       inputs = append(inputs, s)
		       }
	       }
       case []string:
	       inputs = v
       default:
	       c.JSON(400, gin.H{"error": "input must be string or array of strings"})
	       return
       }

       // Generate fake embeddings (e.g. all zeros, dim=8)
       dim := 8
       if len(h.configModels) > 0 {
	       dim = h.configModels[0].Dim
       }
       fake := make([][]float32, len(inputs))
       for i := range fake {
	       fake[i] = make([]float32, dim)
       }

       resp := EmbedResponse{
	       Model: req.Model,
	       Embeddings: fake,
       }
       c.JSON(200, resp)
}

func NewHandler(manager *embedding.Manager, models []ConfigModel, defaultModel string, maxConcurrency int) *Handler {
	if maxConcurrency <= 0 {
		maxConcurrency = 4
	}
	return &Handler{
		manager:      manager,
		configModels: models,
		defaultModel: defaultModel,
		stats:        NewStatsCollector(),
		concurrency:  make(chan struct{}, maxConcurrency),
	}
}

func (h *Handler) HandleEmbed(c *gin.Context) {
	var req EmbedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if model is available and enabled
	found := false
	for _, m := range h.configModels {
		if m.Name == req.Model {
			found = true
			break
		}
	}
	if !found {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model not found or disabled"})
		return
	}

	var inputs []string
	switch v := req.Input.(type) {
	case string:
		inputs = []string{v}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				inputs = append(inputs, s)
			}
		}
	case []string:
		inputs = v
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "input must be string or array of strings"})
		return
	}

	// Apply concurrency l2imit
	h.concurrency <- struct{}{}
	defer func() { <-h.concurrency }()

	start := time.Now()
	embeddings, err := h.manager.Embed(req.Model, inputs)
	duration := time.Since(start)

	if err != nil {
		slog.Error("Embedding failed", "model", req.Model, "error", err)

		errMsg := err.Error()
		// Return 400 for user errors
		if strings.Contains(errMsg, "token limit") ||
		   strings.Contains(errMsg, "empty document") ||
		   strings.Contains(errMsg, "no documents") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg})
		return
	}

	h.stats.RecordRequest(req.Model, duration)

	slog.Info("Embedding request",
		"model", req.Model,
		"inputs_count", len(inputs),
		"duration_ms", duration.Milliseconds(),
	)

	c.JSON(http.StatusOK, EmbedResponse{
		Model:      req.Model,
		Embeddings: embeddings,
	})
}

func (h *Handler) HandleTags(c *gin.Context) {
	var models []ModelDetails
	for _, m := range h.configModels {
		models = append(models, ModelDetails{
			Name:       m.Name,
			ModifiedAt: time.Now(), // Placeholder
			Size:       0,          // Placeholder
			Digest:     "sha256:...",
			Details: Details{
				Format:            "onnx",
				Family:            "bert",
				Families:          []string{"bert"},
				ParameterSize:     "small",
				QuantizationLevel: "f32",
			},
		})
	}

	c.JSON(http.StatusOK, TagResponse{
		Models: models,
	})
}

func (h *Handler) HandleSimilarity(c *gin.Context) {
	var req SimilarityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply concurrency limit
	h.concurrency <- struct{}{}
	defer func() { <-h.concurrency }()

	start := time.Now()
	// 1. Get embeddings for all texts (Query + Documents)
	allTexts := append([]string{req.Query}, req.Documents...)
	embeddings, err := h.manager.Embed(req.Model, allTexts)
	duration := time.Since(start)

	if err != nil {
		slog.Error("Similarity failed", "model", req.Model, "error", err)

		errMsg := err.Error()
		// Return 400 for user errors
		if strings.Contains(errMsg, "token limit") ||
		   strings.Contains(errMsg, "empty document") ||
		   strings.Contains(errMsg, "no documents") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg})
		return
	}

	h.stats.RecordRequest(req.Model, duration)

	queryEmb := embeddings[0]
	docEmbeddings := embeddings[1:]

	// 2. Calculate Dot Product (Cosine Similarity because normalized)
	var results []SimilarityResult
	for i, docEmb := range docEmbeddings {
		var score float32
		for j := range queryEmb {
			score += queryEmb[j] * docEmb[j]
		}
		results = append(results, SimilarityResult{
			Document: req.Documents[i],
			Score:    score,
		})
	}

	c.JSON(http.StatusOK, SimilarityResponse{
		Model:  req.Model,
		Scores: results,
	})
}

func (h *Handler) HandleStats(c *gin.Context) {
	c.JSON(http.StatusOK, h.stats.GetStats())
}
