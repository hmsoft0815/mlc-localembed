// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package api

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"mlc-localembed/internal/embedding"
)

// RunMigration checks for model mismatches and re-embeds data if necessary.
// It returns true if a migration was performed, false if skip was successful.
func RunMigration(manager *embedding.Manager, model ConfigModel, dataDir, outputDir string) (bool, error) {
	manifestPath := filepath.Join(outputDir, "manifest.json")

	// 1. Check for mismatch
	manifest, err := LoadManifest(manifestPath)
	if err == nil {
		err = manifest.CheckMismatch(model.Name, model.Dim)
		if err == nil {
			log.Printf("Migration: Model %s matches existing embeddings in %s. Skipping.", model.Name, outputDir)
			return false, nil
		}
		log.Printf("Migration: MISMATCH for %s: %v. Starting re-embedding...", model.Name, err)
	} else {
		log.Printf("Migration: No manifest found in %s. Initializing embeddings...", outputDir)
	}

	// 2. Scan Data Directory
	files, err := os.ReadDir(dataDir)
	if err != nil {
		return false, fmt.Errorf("failed to read data directory %s: %w", dataDir, err)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	results := make(map[string][]float32)
	count := 0
	for _, f := range files {
		if f.IsDir() || filepath.HasPrefix(f.Name(), ".") {
			continue
		}
		ext := filepath.Ext(f.Name())
		if ext != ".md" && ext != ".txt" {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dataDir, f.Name()))
		if err != nil {
			log.Printf("Warning: could not read %s: %v", f.Name(), err)
			continue
		}

		emb, err := manager.Embed(model.Name, []string{string(content)})
		if err != nil {
			return false, fmt.Errorf("embedding failed for %s: %w", f.Name(), err)
		}
		results[f.Name()] = emb[0]
		count++
	}

	// 3. Save Embeddings
	outputFile := filepath.Join(outputDir, "embeddings.json")
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return false, err
	}

	// 4. Update Manifest
	newManifest := Manifest{
		ModelName:      model.Name,
		Dimension:      model.Dim,
		QueryPrefix:    model.QueryPrefix,
		DocumentPrefix: model.DocumentPrefix,
		GeneratedAt:    time.Now().Format(time.RFC3339),
	}
	if err := newManifest.Save(manifestPath); err != nil {
		return false, err
	}

	log.Printf("Migration: Successfully embedded %d files for model %s into %s", count, model.Name, outputDir)
	return true, nil
}
