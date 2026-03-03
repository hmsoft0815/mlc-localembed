// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package embedding

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
	ort "github.com/yalue/onnxruntime_go"
)

// findOnnxRuntime searches for the ONNX runtime library in common locations
func findOnnxRuntime() string {
	onnxPath := os.Getenv("ONNX_PATH")
	if onnxPath != "" {
		return onnxPath
	}

	var libName string
	switch runtime.GOOS {
	case "darwin":
		libName = "libonnxruntime.dylib"
	case "windows":
		libName = "onnxruntime.dll"
	default:
		libName = "libonnxruntime.so"
	}

	// Search paths
	searchPaths := []string{
		libName,
		filepath.Join("..", libName),
		filepath.Join("bin", libName),
	}

	// Add absolute path for development environment if it exists
	devPath := "/mnt/data2tb/mlcmcp/mcp-proxy/toolrag/localembed"
	searchPaths = append(searchPaths, filepath.Join(devPath, libName))

	for _, p := range searchPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

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
	mu      sync.Mutex
}

func (e *CustomEmbedder) Embed(docs []string) ([][]float32, error) {
	if len(docs) == 0 {
		return nil, fmt.Errorf("no documents provided")
	}

	// Protect shared buffers and session
	e.mu.Lock()
	defer e.mu.Unlock()

	results := make([][]float32, len(docs))
	for idx, doc := range docs {
		if doc == "" {
			return nil, fmt.Errorf("empty document at index %d", idx)
		}

		en, err := e.tokenizer.EncodeSingle(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to encode document at index %d: %w", idx, err)
		}

		tokenCount := len(en.GetIds())
		if tokenCount > e.maxLen {
			return nil, fmt.Errorf("document at index %d exceeds token limit: %d > %d", idx, tokenCount, e.maxLen)
		}

		// Reset buffers
		for i := 0; i < e.maxLen; i++ {
			if i < tokenCount {
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
			return nil, fmt.Errorf("ONNX execution failed for document %d: %w", idx, err)
		}

		// Mean Pooling
		embedding := make([]float32, e.dim)
		count := float32(tokenCount)
		if count == 0 {
			results[idx] = embedding // All zeros
			continue
		}

		for i := 0; i < tokenCount; i++ {
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
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.session == nil {
		return nil
	}
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
	configModels     map[string]string // model name -> custom filename
}

func NewManager(cacheDir string) *Manager {
	// If cacheDir is relative (like "./mlcembed"), we should ensure it's found
	// even if running from 'bin/'.
	if cacheDir != "" && !filepath.IsAbs(cacheDir) {
		if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
			parentCache := filepath.Join("..", cacheDir)
			if _, err := os.Stat(parentCache); err == nil {
				cacheDir = parentCache
			}
		}
	}

	return &Manager{
		cacheDir:     cacheDir,
		models:       make(map[string]Embedder),
		configModels: make(map[string]string),
	}
}

func (m *Manager) SetModelConfig(name, filename string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configModels[name] = filename
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
	customFile := m.configModels[name]
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
	// We check for both our custom naming and the default naming
	var path string
	if name == "multilingual-e5-small" {
		path = filepath.Join(m.cacheDir, "fast-multilingual-e5-small")
	} else if name == "BAAI/bge-small-en-v1.5" {
		path = filepath.Join(m.cacheDir, "fast-bge-small-en-v1.5")
	} else {
		folderName := "models--" + strings.ReplaceAll(name, "/", "--")
		if !strings.Contains(name, "/") {
			folderName = "models--qdrant--" + name
		}
		path = filepath.Join(m.cacheDir, folderName)
	}

	// Determine dimension
	dim := 384

	// Handle custom filename
	onnxFile := "model.onnx"
	if customFile != "" {
		onnxFile = customFile
	}

	// If the custom filename contains path segments (like "onnx/model.onnx"),
	// and we are just looking for the file in 'path', we need to join them.
	fullOnnxPath := filepath.Join(path, onnxFile)

	newEmb, err := NewCustomEmbedderWithFile(path, fullOnnxPath, dim, intra, inter)
	if err != nil {
		return nil, err
	}
	m.models[name] = newEmb
	return newEmb, nil
}

func NewCustomEmbedderWithFile(modelPath, onnxPath string, dim int, intraThreads, interThreads int) (*CustomEmbedder, error) {
	// 1. Load Tokenizer
	tk, err := pretrained.FromFile(filepath.Join(modelPath, "tokenizer.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to load tokenizer at %s: %w", modelPath, err)
	}

	maxLen := 512
	// We disable auto-truncation to detect when input is too long
	tk.WithTruncation(nil)

	// 2. Initialize ONNX Environment if needed
	if !ort.IsInitialized() {
		rtPath := findOnnxRuntime()
		if rtPath != "" {
			ort.SetSharedLibraryPath(rtPath)
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
		onnxPath,
		[]string{"input_ids", "attention_mask", "token_type_ids"},
		[]string{"last_hidden_state"},
		[]ort.ArbitraryTensor{t1, t2, t3},
		[]ort.ArbitraryTensor{tOut},
		options,
	)
	if err != nil {
		t1.Destroy(); t2.Destroy(); t3.Destroy(); tOut.Destroy()
		return nil, fmt.Errorf("failed to create session for %s: %w", onnxPath, err)
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

func NewCustomEmbedder(modelPath string, dim int, intraThreads, interThreads int) (*CustomEmbedder, error) {
	return NewCustomEmbedderWithFile(modelPath, filepath.Join(modelPath, "model.onnx"), dim, intraThreads, interThreads)
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
