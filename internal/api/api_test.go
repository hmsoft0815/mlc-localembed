// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mlc-localembed/internal/embedding"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandleTags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create a dummy manager (won't be used for /api/tags)
	manager := embedding.NewManager("./test_cache")
	
	models := []ConfigModel{
		{Name: "test-model", Dim: 384, Description: "Test"},
	}
	
	handler := NewHandler(manager, models, "test-model")
	
	r := gin.Default()
	r.GET("/api/tags", handler.HandleTags)
	
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/tags", nil)
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test-model")
}
