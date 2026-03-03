// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package embedding

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
	ort "github.com/yalue/onnxruntime_go"
)

// Embedder defines an interface for model execution
type Embedder interface {
	Embed(documents []string) ([][]float32, error)
	Close() error
}

// CustomEmbedder implements manual ONNX execution for models
type CustomEmbedder struct {
	tokenizer *tokenizer.Tokenizer
	session   *ort.AdvancedSession
	dim       int
	maxLen    int
	
	// Buffers for input/output to avoid allocations
	inputIds      []int64
	attentionMask []int64
	tokenTypeIds  []int64
	outputData    []float32
	
	tensors []ort.ArbitraryTensor
}

func NewCustomEmbedder(modelPath string, dim int, intraThreads, interThreads int) (*CustomEmbedder, error) {
	// 1. Load Tokenizer
	tk, err := pretrained.FromFile(filepath.Join(modelPath, "tokenizer.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to load tokenizer: %w", err)
	}

	maxLen := 512
	tk.WithTruncation(&tokenizer.TruncationParams{
		MaxLength: maxLen,
		Strategy:  tokenizer.LongestFirst,
	})

	// 2. Initialize ONNX Environment if needed
	if !ort.IsInitialized() {
		if onnxPath := os.Getenv("ONNX_PATH"); onnxPath != "" {
			ort.SetSharedLibraryPath(onnxPath)
		}
		if err := ort.InitializeEnvironment(); err != nil {
			return nil, fmt.Errorf("failed to init ONNX: %w", err)
		}
	}

	// Session options
	options, err := ort.NewSessionOptions()
	if err != nil {
		return nil, fmt.Errorf("failed to create session options: %w", err)
	}
	defer options.Destroy()

	if intraThreads > 0 {
		options.SetIntraOpNumThreads(intraThreads)
	}
	if interThreads > 0 {
		options.SetInterOpNumThreads(interThreads)
	}

	// 3. Prepare pre-allocated buffers
	inputShape := ort.NewShape(1, int64(maxLen))
	inputIds := make([]int64, maxLen)
	attentionMask := make([]int64, maxLen)
	tokenTypeIds := make([]int64, maxLen)

	t1, _ := ort.NewTensor(inputShape, inputIds)
	t2, _ := ort.NewTensor(inputShape, attentionMask)
	t3, _ := ort.NewTensor(inputShape, tokenTypeIds)

	outputShape := ort.NewShape(1, int64(maxLen), int64(dim))
	outputData := make([]float32, 1*maxLen*dim)
	tOut, _ := ort.NewTensor(outputShape, outputData)

	// 4. Create Session
	session, err := ort.NewAdvancedSession(
		filepath.Join(modelPath, "model.onnx"),
		[]string{"input_ids", "attention_mask", "token_type_ids"},
		[]string{"last_hidden_state"},
		[]ort.ArbitraryTensor{t1, t2, t3},
		[]ort.ArbitraryTensor{tOut},
		options,
	)
	if err != nil {
		t1.Destroy(); t2.Destroy(); t3.Destroy(); tOut.Destroy()
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &CustomEmbedder{
		tokenizer:     tk,
		session:       session,
		dim:           dim,
		maxLen:        maxLen,
		inputIds:      inputIds,
		attentionMask: attentionMask,
		tokenTypeIds:  tokenTypeIds,
		outputData:    outputData,
		tensors:       []ort.ArbitraryTensor{t1, t2, t3, tOut},
	}, nil
}

func (e *CustomEmbedder) Embed(docs []string) ([][]float32, error) {
	results := make([][]float32, len(docs))
	for idx, doc := range docs {
		en, err := e.tokenizer.EncodeSingle(doc)
		if err != nil {
			return nil, err
		}

		// Reset buffers
		for i := 0; i < e.maxLen; i++ {
			if i < len(en.GetIds()) {
				e.inputIds[i] = int64(en.GetIds()[i])
				e.attentionMask[i] = int64(en.GetAttentionMask()[i])
				e.tokenTypeIds[i] = int64(en.GetTypeIds()[i])
			} else {
				e.inputIds[i] = 0
				e.attentionMask[i] = 0
				e.tokenTypeIds[i] = 0
			}
		}

		if err := e.session.Run(); err != nil {
			return nil, err
		}

		// Mean Pooling
		embedding := make([]float32, e.dim)
		count := float32(len(en.GetIds()))
		for i := 0; i < len(en.GetIds()); i++ {
			for j := 0; j < e.dim; j++ {
				embedding[j] += e.outputData[i*e.dim+j]
			}
		}
		
		// Normalize
		norm := float32(0.0)
		for j := 0; j < e.dim; j++ {
			embedding[j] /= count
			norm += embedding[j] * embedding[j]
		}
		norm = float32(math.Sqrt(float64(norm)))
		if norm > 0 {
			for j := 0; j < e.dim; j++ {
				embedding[j] /= norm
			}
		}
		
		results[idx] = embedding
	}
	return results, nil
}

func (e *CustomEmbedder) Close() error {
	for _, t := range e.tensors {
		t.Destroy()
	}
	return e.session.Destroy()
}

// Manager handles multiple models
type Manager struct {
	cacheDir          string
	models            map[string]Embedder
	mu                sync.RWMutex
	onnxIntraThreads int
	onnxInterThreads int
}

func NewManager(cacheDir string) *Manager {
	return &Manager{
		cacheDir: cacheDir,
		models:   make(map[string]Embedder),
	}
}

func (m *Manager) SetOnnxOptions(intra, inter int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onnxIntraThreads = intra
	m.onnxInterThreads = inter
}

func (m *Manager) GetEmbedder(name string) (Embedder, error) {
	m.mu.RLock()
	emb, ok := m.models[name]
	intra := m.onnxIntraThreads
	inter := m.onnxInterThreads
	m.mu.RUnlock()
	if ok {
		return emb, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if emb, ok = m.models[name]; ok {
		return emb, nil
	}

	// Map name to folder (matching preloader logic)
	folderName := "models--" + strings.ReplaceAll(name, "/", "--")
	if !strings.Contains(name, "/") {
		folderName = "models--qdrant--" + name
	}
	path := filepath.Join(m.cacheDir, folderName)

	// Determine dimension (this should ideally come from a config or model info)
	dim := 384 // Default for many small models
	if strings.Contains(name, "e5-small") || strings.Contains(name, "bge-small") {
		dim = 384
	}

	newEmb, err := NewCustomEmbedder(path, dim, intra, inter)
	if err != nil {
		return nil, err
	}
	m.models[name] = newEmb
	return newEmb, nil
}

func (m *Manager) Embed(name string, docs []string) ([][]float32, error) {
	emb, err := m.GetEmbedder(name)
	if err != nil {
		return nil, err
	}
	return emb.Embed(docs)
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, emb := range m.models {
		emb.Close()
	}
	m.models = make(map[string]Embedder)
}
