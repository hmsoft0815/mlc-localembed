// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"mlc-localembed/internal/embedding"
)

func TestHandleTags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := embedding.NewManager("./test_cache")
	models := []ConfigModel{{Name: "test-model", Dim: 384, Description: "Test"}}
	handler := NewHandler(manager, models, "test-model", 4)
	r := gin.Default()
	r.GET("/api/tags", handler.HandleTags)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/tags", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test-model")
}

func TestHandleHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestHandleEmbedFaker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	models := []ConfigModel{{Name: "test", Dim: 8}}
	handler := NewHandler(nil, models, "test", 4)
	r := gin.Default()
	r.POST("/api/embed/faker", handler.HandleEmbedFaker)

	body := `{"model": "test", "input": "hello"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/embed/faker", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "embeddings")
}

func TestHandleShow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	models := []ConfigModel{{Name: "test-model", Dim: 384, Description: "Test Description"}}
	handler := NewHandler(nil, models, "test-model", 4)
	r := gin.Default()
	r.POST("/api/show", handler.HandleShow)

	body := `{"name": "test-model"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/show", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test-model")
	assert.Contains(t, w.Body.String(), "Test Description")
}

func TestHandleStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(nil, nil, "", 4)
	r := gin.Default()
	r.GET("/api/stats", handler.HandleStats)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "total_requests")
}
