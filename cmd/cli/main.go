// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"mlc-localembed/internal/api"
	"mlc-localembed/internal/embedding"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Storage struct {
		CacheDir string `yaml:"cache_dir"`
	} `yaml:"storage"`
	Models struct {
		Default   string           `yaml:"default"`
		Available []api.ConfigModel `yaml:"available"`
	} `yaml:"models"`
}

func main() {
	text := flag.String("text", "Hello, World!", "Text to embed")
	model := flag.String("model", "", "Model to use (optional, defaults to config default)")
	flag.Parse()

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
		log.Fatalf("failed to read %s: %v", configPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	if *model == "" {
		*model = config.Models.Default
	}

	// 2. Initialize embedding manager
	manager := embedding.NewManager(config.Storage.CacheDir)
	defer manager.Close()

	// Configure custom model files
	for _, m := range config.Models.Available {
		if m.ModelFile != "" {
			manager.SetModelConfig(m.Name, m.ModelFile)
		}
	}

	// 3. Generate embedding
	fmt.Printf("Using model: %s\n", *model)
	fmt.Printf("Input text: %s\n", *text)

	embeddings, err := manager.Embed(*model, []string{*text})
	if err != nil {
		log.Fatalf("failed to generate embedding: %v", err)
	}

	if len(embeddings) > 0 {
		fmt.Printf("Generated embedding (first 5 elements): %v...\n", embeddings[0][:5])
		fmt.Printf("Embedding dimension: %d\n", len(embeddings[0]))
	}
}
