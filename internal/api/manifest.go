// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package api

import (
	"encoding/json"
	"fmt"
	"os"
)

// Manifest tracks the metadata of generated embeddings in a storage directory
type Manifest struct {
	ModelName      string `json:"model_name"`
	Dimension      int    `json:"dimension"`
	QueryPrefix    string `json:"query_prefix"`
	DocumentPrefix string `json:"document_prefix"`
	GeneratedAt    string `json:"generated_at"`
}

// LoadManifest reads the manifest from a file
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// SaveManifest writes the manifest to a file
func (m *Manifest) Save(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// CheckMismatch compares the manifest with the current configuration
func (m *Manifest) CheckMismatch(currentModel string, currentDim int) error {
	if m.ModelName != currentModel {
		return fmt.Errorf("model mismatch: stored model is %s, but current is %s", m.ModelName, currentModel)
	}
	if m.Dimension != currentDim {
		return fmt.Errorf("dimension mismatch: stored dimension is %d, but current is %d", m.Dimension, currentDim)
	}
	return nil
}
