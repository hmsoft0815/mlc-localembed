// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mlc-localembed/internal/api"
	"mlc-localembed/internal/embedding"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port           int    `yaml:"port"`
		LogLevel       string `yaml:"log_level"`
		MaxConcurrency int    `yaml:"max_concurrency"`
	} `yaml:"server"`
	Storage struct {
		CacheDir string `yaml:"cache_dir"`
	} `yaml:"storage"`
	Onnx struct {
		IntraOpNumThreads int `yaml:"intra_op_num_threads"`
		InterOpNumThreads int `yaml:"inter_op_num_threads"`
	} `yaml:"onnx"`
	Models struct {
		Default   string           `yaml:"default"`
		Available []api.ConfigModel `yaml:"available"`
	} `yaml:"models"`
}

func main() {
	fmt.Printf("Starting mlc-localembed %s\n", api.Version)
	// 1. Load config
	configPath := "config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try parent directory (if run from bin/)
		if _, err := os.Stat("../config.yaml"); err == nil {
			configPath = "../config.yaml"
		}
	}

	configFile, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("warning: %s not found, using defaults and environment variables", configPath)
	}

	var config Config
	if configFile != nil {
		if err := yaml.Unmarshal(configFile, &config); err != nil {
			log.Fatalf("failed to parse config: %v", err)
		}
	}

	// 2. Override with Environment Variables
	if val := os.Getenv("MLC_PORT"); val != "" {
		fmt.Sscanf(val, "%d", &config.Server.Port)
	}
	if val := os.Getenv("MLC_LOG_LEVEL"); val != "" {
		config.Server.LogLevel = val
	}
	if val := os.Getenv("MLC_MAX_CONCURRENCY"); val != "" {
		fmt.Sscanf(val, "%d", &config.Server.MaxConcurrency)
	}
	if val := os.Getenv("MLC_CACHE_DIR"); val != "" {
		config.Storage.CacheDir = val
	}
	if val := os.Getenv("MLC_INTRA_THREADS"); val != "" {
		fmt.Sscanf(val, "%d", &config.Onnx.IntraOpNumThreads)
	}
	if val := os.Getenv("MLC_INTER_THREADS"); val != "" {
		fmt.Sscanf(val, "%d", &config.Onnx.InterOpNumThreads)
	}
	if val := os.Getenv("MLC_DEFAULT_MODEL"); val != "" {
		config.Models.Default = val
	}

	// Set defaults if still zero
	if config.Server.Port == 0 {
		config.Server.Port = 9142
	}
	if config.Server.MaxConcurrency == 0 {
		config.Server.MaxConcurrency = 4
	}
	if config.Storage.CacheDir == "" {
		config.Storage.CacheDir = "./mlcembed"
	}

	// 3. Initialize embedding manager
	manager := embedding.NewManager(config.Storage.CacheDir)
	manager.SetOnnxOptions(config.Onnx.IntraOpNumThreads, config.Onnx.InterOpNumThreads)

	// Filter available models based on config and apply custom settings
	var availableModels []api.ConfigModel
	for _, m := range config.Models.Available {
		if m.Enabled == nil || *m.Enabled {
			availableModels = append(availableModels, m)
			if m.ModelFile != "" {
				manager.SetModelConfig(m.Name, m.ModelFile)
			}
			if m.Pooling != "" {
				manager.SetPoolingConfig(m.Name, m.Pooling)
			}
		}
	}

	// 3. Initialize API handler
	handler := api.NewHandler(manager, availableModels, config.Models.Default, config.Server.MaxConcurrency)

	// 4. Setup Gin
	if config.Server.LogLevel == "info" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	r.POST("/api/embed", handler.HandleEmbed)
	r.GET("/api/tags", handler.HandleTags)
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": api.Version})
	})
	r.GET("/api/stats", handler.HandleStats)
	r.POST("/api/embed/faker", handler.HandleEmbedFaker)
	r.POST("/api/test/similarity", handler.HandleSimilarity)

	// 5. Start server with Graceful Shutdown support
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Server.Port),
		Handler: r,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		fmt.Printf("Server listening on %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no parameter) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so no need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	// 6. Close Manager (closes ONNX sessions)
	log.Println("Closing embedding sessions...")
	manager.Close()

	log.Println("Server exiting")
}
