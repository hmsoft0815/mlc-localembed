// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"context"
	"flag"
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

const ollamaVersion = "0.17.4"

func printBanner(version string) {
	green  := "\033[38;2;0;255;0m"
	orange := "\033[38;2;255;165;0m"
	red    := "\033[38;2;255;0;0m"
	reset  := "\033[0m"

	// stupid games here ;)
	line1 := `  _                 _ _____           _              _ `
	line2 := ` | | ___   ___ __ _| | ____|_ __ ___ | |__   ___  __| |`
	line3 := ` | |/ _ \ / __/ _` + "`" + ` | |  _| | '_ ` + "`" + ` _ \| '_ \ / _ \/ _` + "`" + ` |`
	line4 := ` | | (_) | (_| (_| | | |___| | | | | | |_)  | __/ (_| |`
	line5 := ` |_|\___/ \__\__,__|_|_____|_| |_| |_|_.__/ \___|\__,_|`

	fmt.Println(green + line1 + reset)
	fmt.Println(green + line2 + reset)
	fmt.Println(orange + line3 + reset)
	fmt.Println(red + line4 + reset)
	fmt.Println(red + line5 + reset)

	fmt.Printf(" [ Local Embedding Engine | %s ]\n", version)
	fmt.Println(" [ MIT License | Written by Michael Lechner ]")
	fmt.Println(" ------------------------------------------------")
}

func printHelp() {
	fmt.Println("\nUsage: mlcembedder [options]")
	fmt.Println("\nOptions:")
	fmt.Println("  -config <path>        Path to config file (default: config.yaml)")
	fmt.Println("  -port <int>           Port to listen on (default: 9142)")
	fmt.Println("  -cache-dir <path>     Directory to store models (default: ./mlcembed)")
	fmt.Println("  -concurrency <int>    Max concurrent requests (default: 4)")
	fmt.Println("  -lru-cache-size <int> Max items in embedding cache (default: 10, 0 to disable)")
	fmt.Println("  -log-level <string>   Log level: info, debug (default: info)")
	fmt.Println("  -gpu                  Enable GPU acceleration (auto-detect provider)")
	fmt.Println("  -gpu-ep <string>      Force specific GPU execution provider (cuda, coreml, directml, tensorrt)")
	fmt.Println("  -h, --help            Show this help message")
	fmt.Println("\nConfiguration Priority:")
	fmt.Println("  1. Command-line flags (highest)")
	fmt.Println("  2. Environment variables")
	fmt.Println("  3. config.yaml file")
	fmt.Println("  4. Compiled-in defaults (lowest)")
	fmt.Println("\nEnvironment Variable Overrides:")
	fmt.Println("  MLC_PORT              Matches -port")
	fmt.Println("  MLC_LOG_LEVEL         Matches -log-level")
	fmt.Println("  MLC_MAX_CONCURRENCY   Matches -concurrency")
	fmt.Println("  MLC_CACHE_DIR         Matches -cache-dir")
	fmt.Println("  MLC_INTRA_THREADS     ONNX Intra-op threads")
	fmt.Println("  MLC_INTER_THREADS     ONNX Inter-op threads")
	fmt.Println("  MLC_USE_GPU           Enable GPU acceleration (true/false)")
	fmt.Println("  MLC_GPU_EP            Specific GPU execution provider")
	fmt.Println("  MLC_DEFAULT_MODEL     Default model name to use")
	fmt.Println("  MLC_LRU_CACHE         Matches -lru-cache-size")
	fmt.Println("")
}

type Config struct {
	Server struct {
		Port           int    `yaml:"port"`
		LogLevel       string `yaml:"log_level"`
		MaxConcurrency int    `yaml:"max_concurrency"`
		LRUCacheSize   *int   `yaml:"lru_cache_size"`
	} `yaml:"server"`
	Storage struct {
		CacheDir string `yaml:"cache_dir"`
	} `yaml:"storage"`
	Onnx struct {
		IntraOpNumThreads int    `yaml:"intra_op_num_threads"`
		InterOpNumThreads int    `yaml:"inter_op_num_threads"`
		UseGPU            bool   `yaml:"use_gpu"`
		ExecutionProvider string `yaml:"execution_provider"`
	} `yaml:"onnx"`
	Models struct {
		Default   string            `yaml:"default"`
		Available []api.ConfigModel `yaml:"available"`
	} `yaml:"models"`
}

func main() {
	// Define flags
	configPathFlag := flag.String("config", "config.yaml", "Path to config file")
	portFlag := flag.Int("port", 0, "Port to listen on")
	cacheDirFlag := flag.String("cache-dir", "", "Directory to store models")
	concurrencyFlag := flag.Int("concurrency", 0, "Max concurrent requests")
	lruCacheSizeFlag := flag.Int("lru-cache-size", -1, "LRU cache size")
	logLevelFlag := flag.String("log-level", "", "Log level (info, debug)")
	gpuFlag := flag.Bool("gpu", false, "Enable GPU acceleration")
	gpuEPFlag := flag.String("gpu-ep", "", "Execution provider (cuda, coreml, directml, tensorrt)")
	helpFlag := flag.Bool("help", false, "Show help")
	hFlag := flag.Bool("h", false, "Show help")

	flag.Parse()

	if *helpFlag || *hFlag {
		printHelp()
		return
	}

	printBanner(api.Version)

	// 1. Resolve Config Path and Load file
	configPath := *configPathFlag
	if configPath == "config.yaml" {
		// If using default, check current and parent dir
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			if _, err := os.Stat("../config.yaml"); err == nil {
				configPath = "../config.yaml"
			}
		}
	}

	var config Config
	if configFile, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(configFile, &config); err != nil {
			log.Fatalf("failed to parse config: %v", err)
		}
	} else if *configPathFlag != "config.yaml" {
		log.Fatalf("failed to read config file at %s: %v", configPath, err)
	} else {
		log.Printf("warning: %s not found, using defaults, env, and flags", configPath)
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
	if val := os.Getenv("MLC_USE_GPU"); val != "" {
		config.Onnx.UseGPU = (val == "true" || val == "1")
	}
	if val := os.Getenv("MLC_GPU_EP"); val != "" {
		config.Onnx.ExecutionProvider = val
	}
	if val := os.Getenv("MLC_DEFAULT_MODEL"); val != "" {
		config.Models.Default = val
	}
	if val := os.Getenv("MLC_LRU_CACHE"); val != "" {
		var size int
		if _, err := fmt.Sscanf(val, "%d", &size); err == nil {
			config.Server.LRUCacheSize = &size
		}
	}

	// 3. Override with Flags (if provided)
	if *portFlag != 0 {
		config.Server.Port = *portFlag
	}
	if *cacheDirFlag != "" {
		config.Storage.CacheDir = *cacheDirFlag
	}
	if *concurrencyFlag != 0 {
		config.Server.MaxConcurrency = *concurrencyFlag
	}
	if *lruCacheSizeFlag != -1 {
		config.Server.LRUCacheSize = lruCacheSizeFlag
	}
	if *logLevelFlag != "" {
		config.Server.LogLevel = *logLevelFlag
	}
	if *gpuFlag {
		config.Onnx.UseGPU = true
	}
	if *gpuEPFlag != "" {
		config.Onnx.ExecutionProvider = *gpuEPFlag
	}

	// Set defaults if still zero
	if config.Server.Port == 0 {
		config.Server.Port = 9142
	}
	if config.Server.MaxConcurrency == 0 {
		config.Server.MaxConcurrency = 4
	}
	if config.Server.LRUCacheSize == nil {
		ten := 10
		config.Server.LRUCacheSize = &ten
	}
	if config.Storage.CacheDir == "" {
		config.Storage.CacheDir = "./mlcembed"
	}

	// 4. Initialize embedding manager
	manager := embedding.NewManager(config.Storage.CacheDir)
	manager.SetOnnxOptions(config.Onnx.IntraOpNumThreads, config.Onnx.InterOpNumThreads)
	manager.SetGPU(config.Onnx.UseGPU, config.Onnx.ExecutionProvider)
	manager.SetLRUCache(*config.Server.LRUCacheSize)

	// Filter available models based on config and apply custom settings
	var availableModels []api.ConfigModel
	for _, m := range config.Models.Available {
		if m.Enabled == nil || *m.Enabled {
			availableModels = append(availableModels, m)
			if m.ModelFile != "" || m.Dim != 0 {
				manager.SetModelConfig(m.Name, m.ModelFile, m.Dim)
			}
			if m.Pooling != "" {
				manager.SetPoolingConfig(m.Name, m.Pooling)
			}
			for _, alias := range m.Aliases {
				manager.AddAlias(m.Name, alias)
			}
		}
	}

	// 5. Initialize API handler
	handler := api.NewHandler(manager, availableModels, config.Models.Default, config.Server.MaxConcurrency)

	// 6. Setup Gin
	if config.Server.LogLevel == "info" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	r.POST("/api/embed", handler.HandleEmbed)
	r.POST("/api/embeddings", handler.HandleEmbed) // Plural for older clients
	r.POST("/api/generate", handler.HandleGenerate)
	r.GET("/api/tags", handler.HandleTags)
	r.GET("/api/ps", handler.HandlePs)
	r.POST("/api/show", handler.HandleShow)
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"version": ollamaVersion,
			"via":     fmt.Sprintf("mlc-localembed %s", api.Version),
		})
	})
	r.GET("/api/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version": ollamaVersion,
			"via":     fmt.Sprintf("mlc-localembed %s", api.Version),
		})
	})
	r.GET("/api/stats", handler.HandleStats)
	r.POST("/api/embed/faker", handler.HandleEmbedFaker)
	r.POST("/api/test/similarity", handler.HandleSimilarity)

	// 7. Start server with Graceful Shutdown support
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Server.Port),
		Handler: r,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		fmt.Printf(" Listening on %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	// 8. Close Manager (closes ONNX sessions)
	log.Println("Closing embedding sessions...")
	manager.Close()

	log.Println("Server exiting")
}
