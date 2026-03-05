// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package embedding

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func dotProduct(a, b []float32) float32 {
	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

func TestSemanticSimilarity(t *testing.T) {
	// Skip if models are not downloaded
	if _, err := os.Stat("../../mlcembed/fast-multilingual-e5-small/model.onnx"); os.IsNotExist(err) {
		t.Skip("Skipping semantic test: model not found in mlcembed/")
	}

	manager := NewManager("../../mlcembed")
	modelName := "multilingual-e5-small"

	// E5 models prefer "query: " and "passage: " prefixes
	documents := []string{
		"passage: The capital of France is Paris.",
		"passage: The weather is sunny and bright today.",
		"passage: Paris is the primary French city and its seat of government.",
	}
	query := "query: Which city is the capital of the French Republic?"

	// 1. Generate Embeddings
	allTexts := append(documents, query)
	embeddings, err := manager.Embed(modelName, allTexts)
	if err != nil {
		t.Fatalf("Failed to generate embeddings: %v", err)
	}

	embDocA := embeddings[0]
	embDocB := embeddings[1]
	embDocC := embeddings[2]
	embQuery := embeddings[3]

	// 2. Calculate Similarities (Dot Product works because vectors are normalized)
	simA := dotProduct(embQuery, embDocA)
	simB := dotProduct(embQuery, embDocB)
	simC := dotProduct(embQuery, embDocC)

	fmt.Printf("\n--- Semantic Similarity Results ---\n")
	fmt.Printf("Query: %s\n", query)
	fmt.Printf("Sim(A) [Paris/Capital]: %.4f\n", simA)
	fmt.Printf("Sim(B) [Weather]:       %.4f\n", simB)
	fmt.Printf("Sim(C) [Paris/Gov]:     %.4f\n", simC)
	fmt.Printf("-----------------------------------\n")

	// 3. Assertions
	assert.Greater(t, simA, simB, "A (Paris) should be more similar to query than B (Weather)")
	assert.Greater(t, simC, simB, "C (Paris Gov) should be more similar to query than B (Weather)")

	// C is often even more similar than A due to the word 'government'
	assert.Greater(t, simC, float32(0.8), "High similarity expected for semantically identical content")
}
