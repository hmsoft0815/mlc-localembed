// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

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
	cmd := flag.String("cmd", "embed", "Command to run (embed, migrate)")
	text := flag.String("text", "Hello, World!", "Text to embed")
	model := flag.String("model", "", "Model to use")
	dataDir := flag.String("data", "./data", "Directory containing source markdown files for migration")
	outputDir := flag.String("out", "./embeddings", "Output directory for embeddings")
	flag.Parse()

	// 1. Load config
	configPath := "config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
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

	// Configure custom model settings
	var selectedModel api.ConfigModel
	found := false
	for _, m := range config.Models.Available {
		if m.Name == *model {
			selectedModel = m
			found = true
		}
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

	if !found {
		log.Fatalf("Model %s not found in config", *model)
	}

	switch *cmd {
	case "embed":
		runEmbed(manager, *model, *text)
	case "migrate":
		runMigrate(manager, selectedModel, *dataDir, *outputDir)
	default:
		fmt.Printf("Unknown command: %s\n", *cmd)
	}
}

func runEmbed(manager *embedding.Manager, model, text string) {
	fmt.Printf("Using model: %s\n", model)
	fmt.Printf("Input text: %s\n", text)

	embeddings, err := manager.Embed(model, []string{text})
	if err != nil {
		log.Fatalf("failed to generate embedding: %v", err)
	}

	if len(embeddings) > 0 {
		fmt.Printf("Generated embedding (first 5 elements): %v...\n", embeddings[0][:5])
		fmt.Printf("Embedding dimension: %d\n", len(embeddings[0]))
	}
}

func runMigrate(manager *embedding.Manager, model api.ConfigModel, dataDir, outputDir string) {
	manifestPath := filepath.Join(outputDir, "manifest.json")
	
	// 1. Automatic Detection
	fmt.Printf("Checking for model mismatch in %s...\n", outputDir)
	manifest, err := api.LoadManifest(manifestPath)
	if err == nil {
		err = manifest.CheckMismatch(model.Name, model.Dim)
		if err == nil {
			fmt.Printf("SUCCESS: Model %s with dimension %d matches existing embeddings. No migration needed.\n", model.Name, model.Dim)
			return
		}
		fmt.Printf("MISMATCH DETECTED: %v\n", err)
		fmt.Println("Starting migration to new model...")
	} else {
		fmt.Println("No existing manifest found. Creating new embedding index...")
	}

	// 2. Scan Data Directory
	files, err := os.ReadDir(dataDir)
	if err != nil {
		log.Fatalf("failed to read data directory: %v", err)
	}

	os.MkdirAll(outputDir, 0755)
	
	results := make(map[string][]float32)
	for _, f := range files {
		if f.IsDir() || !filepath.HasPrefix(f.Name(), ".") && (filepath.Ext(f.Name()) == ".md" || filepath.Ext(f.Name()) == ".txt") {
			content, err := os.ReadFile(filepath.Join(dataDir, f.Name()))
			if err != nil {
				fmt.Printf("Warning: could not read %s: %v\n", f.Name(), err)
				continue
			}

			fmt.Printf("Embedding %s...\n", f.Name())
			emb, err := manager.Embed(model.Name, []string{string(content)})
			if err != nil {
				log.Fatalf("Migration failed at file %s: %v", f.Name(), err)
			}
			results[f.Name()] = emb[0]
		}
	}

	// 3. Save Embeddings
	outputFile := filepath.Join(outputDir, "embeddings.json")
	data, _ := json.MarshalIndent(results, "", "  ")
	os.WriteFile(outputFile, data, 0644)

	// 4. Update Manifest
	newManifest := api.Manifest{
		ModelName:      model.Name,
		Dimension:      model.Dim,
		QueryPrefix:    model.QueryPrefix,
		DocumentPrefix: model.DocumentPrefix,
		GeneratedAt:    time.Now().Format(time.RFC3339),
	}
	newManifest.Save(manifestPath)
	fmt.Printf("Migration complete! Saved %d embeddings to %s\n", len(results), outputDir)
}
