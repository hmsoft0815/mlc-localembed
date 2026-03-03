// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package api

import (
	"log/slog"
	"net/http"
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
	manager     *embedding.Manager
	configModels []ConfigModel
	defaultModel string
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

func NewHandler(manager *embedding.Manager, models []ConfigModel, defaultModel string) *Handler {
	return &Handler{
		manager:     manager,
		configModels: models,
		defaultModel: defaultModel,
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

	start := time.Now()
	embeddings, err := h.manager.Embed(req.Model, inputs)
	duration := time.Since(start)

	if err != nil {
		slog.Error("Embedding failed", "model", req.Model, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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
