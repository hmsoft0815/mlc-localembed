// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
)

type EmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

func main() {
	text := os.Getenv("TEST_TEXT")
	if text == "" {
		text = "query: wie ist das wetter heute in berlin?"
	}

	modelCustom := os.Getenv("MLC_MODEL")
	if modelCustom == "" {
		modelCustom = "multilingual-e5-small"
	}

	modelOllama := os.Getenv("OLLAMA_MODEL")
	if modelOllama == "" {
		modelOllama = "multilingual-e5-small" // Often the same, but can be overridden
	}

	// Defaults to our project's port 9142
	customURL := os.Getenv("MLC_BASE_URL")
	if customURL == "" {
		customURL = "http://localhost:9142"
	}
	customURL += "/api/embed"

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	ollamaURL += "/api/embed"

	fmt.Printf("Vergleiche Vektoren für: \"%s\"\n", text)
	fmt.Printf("Local Model:  %s\n", modelCustom)
	fmt.Printf("Ollama Model: %s\n\n", modelOllama)

	// 1. Vektor von unserem Service holen
	vecCustom, err := fetchVector(customURL, modelCustom, text, true)
	if err != nil {
		fmt.Printf("❌ Fehler Custom Service (%s): %v\n", customURL, err)
		os.Exit(1)
	}

	// 2. Vektor von Ollama holen
	vecOllama, err := fetchVector(ollamaURL, modelOllama, text, true)
	if err != nil {
		fmt.Printf("❌ Fehler Ollama (%s): %v\n", ollamaURL, err)
		fmt.Printf("Hinweis: Läuft Ollama? Ist das Modell geladen? (ollama run %s)\n", modelOllama)
		os.Exit(1)
	}

	// 3. Mathematischer Vergleich
	if len(vecCustom) != len(vecOllama) {
		fmt.Printf("❌ Dimension-Mismatch: Custom=%d, Ollama=%d\n", len(vecCustom), len(vecOllama))
		return
	}

	fmt.Printf("Custom Vector (first 5): %v\n", vecCustom[:5])
	fmt.Printf("Ollama Vector (first 5): %v\n\n", vecOllama[:5])

	similarity := cosineSimilarity(vecCustom, vecOllama)

	fmt.Printf("Dimensionen: %d\n", len(vecCustom))
	fmt.Printf("Cosine Similarity: %.6f\n", similarity)

	if similarity > 0.99 {
		fmt.Println("✅ EXZELLENT: Die Implementierung ist mathematisch identisch.")
	} else if similarity > 0.95 {
		fmt.Println("⚠️ GUT: Leichte Abweichungen (wahrscheinlich durch Quantisierung).")
	} else {
		fmt.Println("❌ KRITISCH: Große Abweichung! Prüfe Pooling, Normalisierung oder Präfixe.")
	}
}

func fetchVector(url, model, input string, isOllama bool) ([]float32, error) {
	var requestBody []byte
	// Both services now use the array input format
	req := struct {
		Model string   `json:"model"`
		Input []string `json:"input"`
	}{Model: model, Input: []string{input}}
	requestBody, _ = json.Marshal(req)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, errResp.Error)
	}

	var result struct {
		Embeddings interface{} `json:"embeddings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	switch v := result.Embeddings.(type) {
	case []interface{}:
		if len(v) > 0 {
			// Check for nested array [][]float32
			if first, ok := v[0].([]interface{}); ok {
				return convertToFloat32(first), nil
			}
			return convertToFloat32(v), nil
		}
	}
	return nil, fmt.Errorf("unbekanntes Antwortformat")
}

func convertToFloat32(slice []interface{}) []float32 {
	f32 := make([]float32, len(slice))
	for i, v := range slice {
		f32[i] = float32(v.(float64))
	}
	return f32
}

func cosineSimilarity(a, b []float32) float32 {
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}
