// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"mlc-localembed/internal/embedding"

	"github.com/gin-gonic/gin"
)

// EmbedRequest follows Ollama's /api/embed request structure.
// It supports both the modern "input" field (string or array of strings)
// and the legacy "prompt" field.
type EmbedRequest struct {
	Model  string      `json:"model" binding:"required"`
	Input  interface{} `json:"input"`  // New style: string or []string
	Prompt string      `json:"prompt"` // Legacy style: string
}

// EmbedResponse follows Ollama's /api/embed response structure.
// It provides a list of embedding vectors and maintains compatibility
// with legacy clients expecting a single "embedding" field.
type EmbedResponse struct {
	Model      string      `json:"model"`
	Embeddings [][]float32 `json:"embeddings,omitempty"`
	Embedding  []float32   `json:"embedding,omitempty"` // Legacy style support
}

// TagResponse follows Ollama's /api/tags response structure.
type TagResponse struct {
	Models []ModelDetails `json:"models"`
}

// ProcessResponse follows Ollama's /api/ps response structure, showing currently loaded models.
type ProcessResponse struct {
	Models []ModelDetails `json:"models"`
}

// ShowRequest follows Ollama's /api/show request structure to retrieve model metadata.
type ShowRequest struct {
	Name string `json:"name" binding:"required"`
}

// ShowResponse follows Ollama's /api/show response structure.
type ShowResponse struct {
	Modelfile  string                 `json:"modelfile"`
	Parameters string                 `json:"parameters"`
	Template   string                 `json:"template"`
	Details    Details                `json:"details"`
	ModelInfo  map[string]interface{} `json:"model_info"`
}

// SimilarityRequest defines the input for the semantic similarity calculation endpoint.
type SimilarityRequest struct {
	Model     string   `json:"model" binding:"required"`
	Query     string   `json:"query" binding:"required"`
	Documents []string `json:"documents" binding:"required"`
}

// SimilarityResponse contains the calculated scores for a list of documents.
type SimilarityResponse struct {
	Model  string             `json:"model"`
	Scores []SimilarityResult `json:"scores"`
}

// SimilarityResult represents the similarity score for a specific document.
type SimilarityResult struct {
	Document string  `json:"document"`
	Score    float32 `json:"score"`
}

// ModelDetails contains metadata about an embedding model.
type ModelDetails struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    Details   `json:"details"`
}

// Details provides technical specifications of the model format and architecture.
type Details struct {
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// ConfigModel represents a model configuration from the YAML config file.
type ConfigModel struct {
	Name           string   `yaml:"name"`
	Aliases        []string `yaml:"aliases"`         // Optional: Alternative names
	SourceRepo     string   `yaml:"source_repo"`     // Optional: HuggingFace repo
	ModelFile      string   `yaml:"model_file"`      // Optional: Specific ONNX file
	Pooling        string   `yaml:"pooling"`         // Optional: "mean" or "cls"
	QueryPrefix    string   `yaml:"query_prefix"`    // Optional: e.g. "query: " or "search_query: "
	DocumentPrefix string   `yaml:"document_prefix"` // Optional: e.g. "passage: "
	Dim            int      `yaml:"dim"`
	Description    string   `yaml:"description"`
	Enabled        *bool    `yaml:"enabled"`
}

// Handler manages the API endpoints and coordinates between the web server and the embedding manager.
type Handler struct {
	manager      *embedding.Manager
	configModels []ConfigModel
	defaultModel string
	stats        *StatsCollector
	concurrency  chan struct{}
}

// resolveModel looks up the primary model name if a known name or alias was provided.
func (h *Handler) resolveModel(requestedModel string) string {
	for _, m := range h.configModels {
		if m.Name == requestedModel {
			return m.Name
		}
		for _, alias := range m.Aliases {
			if alias == requestedModel {
				return m.Name
			}
		}
	}
	return ""
}

// HandleEmbedFaker returns synthetic, zeroed embeddings for testing and development
// without requiring the ONNX runtime or model weights.
func (h *Handler) HandleEmbedFaker(c *gin.Context) {
	var req EmbedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	modelName := h.resolveModel(req.Model)
	if modelName == "" {
		modelName = req.Model // Fallback for faker
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
		Model:      req.Model, // Return the requested name (following Ollama behavior)
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

// HandleEmbed generates embeddings for the provided input using the specified model.
// It supports both modern array-based input and legacy single-string prompts.
func (h *Handler) HandleEmbed(c *gin.Context) {
	var req EmbedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Resolve Model Name/Alias
	actualModelName := h.resolveModel(req.Model)
	if actualModelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model not found or disabled"})
		return
	}

	var inputs []string
	if req.Input != nil {
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
	} else if req.Prompt != "" {
		inputs = []string{req.Prompt}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "either input or prompt is required"})
		return
	}

	// Apply concurrency limit to prevent system overload
	h.concurrency <- struct{}{}
	defer func() { <-h.concurrency }()

	start := time.Now()
	embeddings, err := h.manager.Embed(actualModelName, inputs)
	duration := time.Since(start)

	if err != nil {
		slog.Error("Embedding failed", "model", actualModelName, "error", err)

		errMsg := err.Error()
		// Return 400 for user-correctable errors
		if strings.Contains(errMsg, "token limit") ||
			strings.Contains(errMsg, "empty document") ||
			strings.Contains(errMsg, "no documents") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg})
		return
	}

	h.stats.RecordRequest(actualModelName, duration)

	slog.Info("Embedding request",
		"model", actualModelName,
		"requested_as", req.Model,
		"inputs_count", len(inputs),
		"duration_ms", duration.Milliseconds(),
	)

	resp := EmbedResponse{
		Model:      req.Model, // Ollama returns the name that was used in the request
		Embeddings: embeddings,
	}

	// Support legacy single-embedding response format if only one input was provided
	if len(embeddings) == 1 {
		resp.Embedding = embeddings[0]
	}

	c.JSON(http.StatusOK, resp)
}

// HandleTags returns a list of all available models configured in the system.
func (h *Handler) HandleTags(c *gin.Context) {
	var models []ModelDetails
	for _, m := range h.configModels {
		models = append(models, ModelDetails{
			Name:       m.Name,
			ModifiedAt: time.Now(), // Placeholder: actual modification time not tracked
			Size:       0,          // Placeholder: ONNX file size could be calculated here
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

// HandlePs returns information about models currently loaded into memory.
func (h *Handler) HandlePs(c *gin.Context) {
	active := h.manager.GetActiveModels()
	var models []ModelDetails
	for _, name := range active {
		// Try to find original config for better details
		var mCfg *ConfigModel
		for i := range h.configModels {
			if h.configModels[i].Name == name {
				mCfg = &h.configModels[i]
				break
			}
		}

		details := ModelDetails{
			Name:       name,
			ModifiedAt: time.Now(), // Loaded recently
			Size:       0,
			Digest:     "sha256:...",
			Details: Details{
				Format:            "onnx",
				Family:            "bert",
				Families:          []string{"bert"},
				ParameterSize:     "small",
				QuantizationLevel: "f32",
			},
		}
		if mCfg != nil {
			// Additional metadata from config could be added here
		}
		models = append(models, details)
	}

	c.JSON(http.StatusOK, ProcessResponse{
		Models: models,
	})
}

// HandleShow provides detailed metadata and a simulated "Modelfile" for a specific model.
func (h *Handler) HandleShow(c *gin.Context) {
	var req ShowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	actualName := h.resolveModel(req.Name)
	if actualName == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
		return
	}

	var mCfg *ConfigModel
	for i := range h.configModels {
		if h.configModels[i].Name == actualName {
			mCfg = &h.configModels[i]
			break
		}
	}

	// Construct a descriptive Modelfile for user information
	modelfile := fmt.Sprintf("# Modelfile for %s\nFROM %s\n", actualName, actualName)
	if mCfg != nil {
		if mCfg.Description != "" {
			modelfile += fmt.Sprintf("# %s\n", mCfg.Description)
		}
		modelfile += fmt.Sprintf("PARAMETER dimension %d\n", mCfg.Dim)
		if mCfg.Pooling != "" {
			modelfile += fmt.Sprintf("PARAMETER pooling %s\n", mCfg.Pooling)
		}
	}

	resp := ShowResponse{
		Modelfile: modelfile,
		Details: Details{
			Format:            "onnx",
			Family:            "bert",
			Families:          []string{"bert"},
			ParameterSize:     "small",
			QuantizationLevel: "f32",
		},
		ModelInfo: map[string]interface{}{
			"general.architecture": "bert",
			"general.description":  "Embedding-only model provided by LocalEmbed",
		},
	}

	if mCfg != nil {
		resp.ModelInfo["onnx.dimension"] = mCfg.Dim
		if mCfg.Description != "" {
			resp.ModelInfo["general.description"] = mCfg.Description
		}
	}

	c.JSON(http.StatusOK, resp)
}

// HandleSimilarity calculates semantic similarity scores between a query and multiple documents.
// It uses normalized dot product (equivalent to cosine similarity) on the generated embeddings.
func (h *Handler) HandleSimilarity(c *gin.Context) {
	var req SimilarityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Resolve Model Name/Alias and get config
	actualModelName := h.resolveModel(req.Model)
	if actualModelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model not found or disabled"})
		return
	}

	var modelCfg *ConfigModel
	for i := range h.configModels {
		if h.configModels[i].Name == actualModelName {
			modelCfg = &h.configModels[i]
			break
		}
	}

	// Apply prefixes if configured for the model (e.g., E5 models)
	query := req.Query
	if modelCfg != nil && modelCfg.QueryPrefix != "" {
		query = modelCfg.QueryPrefix + query
	}

	docs := make([]string, len(req.Documents))
	copy(docs, req.Documents)
	if modelCfg != nil && modelCfg.DocumentPrefix != "" {
		for i := range docs {
			docs[i] = modelCfg.DocumentPrefix + docs[i]
		}
	}

	// Apply concurrency limit
	h.concurrency <- struct{}{}
	defer func() { <-h.concurrency }()

	start := time.Now()
	// 1. Get embeddings for all texts (Query + Documents) in one batch if possible
	allTexts := append([]string{query}, docs...)
	embeddings, err := h.manager.Embed(actualModelName, allTexts)
	duration := time.Since(start)

	if err != nil {
		slog.Error("Similarity failed", "model", actualModelName, "error", err)

		errMsg := err.Error()
		if strings.Contains(errMsg, "token limit") ||
			strings.Contains(errMsg, "empty document") ||
			strings.Contains(errMsg, "no documents") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg})
		return
	}

	h.stats.RecordRequest(actualModelName, duration)

	queryEmb := embeddings[0]
	docEmbeddings := embeddings[1:]

	// 2. Calculate Dot Product (Cosine Similarity because vectors are normalized)
	var results []SimilarityResult
	for i, docEmb := range docEmbeddings {
		var score float32
		for j := range queryEmb {
			score += queryEmb[j] * docEmb[j]
		}
		results = append(results, SimilarityResult{
			Document: req.Documents[i], // Return original document text
			Score:    score,
		})
	}

	c.JSON(http.StatusOK, SimilarityResponse{
		Model:  req.Model,
		Scores: results,
	})
}

// HandleStats returns usage statistics for the server.
func (h *Handler) HandleStats(c *gin.Context) {
	c.JSON(http.StatusOK, h.stats.GetStats())
}

// HandleGenerate is a stub for the /api/generate endpoint, informing users that this is an embedding-only server.
func (h *Handler) HandleGenerate(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "This is an embedding-only server. Chat and text generation are not supported.",
	})
}
