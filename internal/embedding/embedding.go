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
		// RPM standard paths
		filepath.Join("/usr/lib64/localembed", libName),
		filepath.Join("/usr/lib/localembed", libName),
		"/usr/local/lib/" + libName,
	}

	// Add relative path from binary directory
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		// Check if lib is next to binary or in ../lib/localembed (typical for RPM)
		searchPaths = append(searchPaths, filepath.Join(exeDir, libName))
		searchPaths = append(searchPaths, filepath.Join(exeDir, "..", "lib64", "localembed", libName))
		searchPaths = append(searchPaths, filepath.Join(exeDir, "..", "lib", "localembed", libName))
	}

	for _, p := range searchPaths {
		if _, err := os.Stat(p); err == nil {
			if absPath, err := filepath.Abs(p); err == nil {
				return absPath
			}
			return p
		}
	}
	return ""
}

// Embedder defines an interface for model execution
type Embedder interface {
	Embed(documents []string) ([][]float32, error)
	Close() error
	SetPooling(pooling string)
}

// CustomEmbedder implements manual ONNX execution for models
type CustomEmbedder struct {
	tokenizer *tokenizer.Tokenizer
	session   *ort.AdvancedSession
	dim       int
	maxLen    int
	pooling   string // "mean" or "cls"

	// Buffers for input/output to avoid allocations
	inputIds      []int64
	attentionMask []int64
	tokenTypeIds  []int64
	outputData    []float32

	tensors []ort.ArbitraryTensor
	mu      sync.Mutex
}

func (e *CustomEmbedder) SetPooling(pooling string) {
	e.pooling = pooling
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

		en, err := e.tokenizer.EncodeSingle(doc, true)
		if err != nil {
			return nil, fmt.Errorf("failed to encode document at index %d: %w", idx, err)
		}

		tokenCount := len(en.GetIds())
		if tokenCount > e.maxLen {
			tokenCount = e.maxLen
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

		embedding := make([]float32, e.dim)

		if e.pooling == "cls" {
			// [CLS] Pooling: Take the first token vector
			copy(embedding, e.outputData[0:e.dim])
		} else {
			// Masked Mean Pooling
			var activeTokens float32 = 0
			// Strictly iterate over ALL tokens up to maxLen and filter by mask
			// This ensures we divide by the correct denominator and only sum seen tokens.
			for i := 0; i < e.maxLen; i++ {
				if e.attentionMask[i] == 1 {
					activeTokens++
					offset := i * e.dim
					for j := 0; j < e.dim; j++ {
						embedding[j] += e.outputData[offset+j]
					}
				}
			}

			if activeTokens > 0 {
				// 1. Average (Mean)
				for j := 0; j < e.dim; j++ {
					embedding[j] /= activeTokens
				}
			}
		}

		// 2. L2 Normalize
		var normSum float64 = 0
		for j := 0; j < e.dim; j++ {
			normSum += float64(embedding[j]) * float64(embedding[j])
		}
		norm := float32(math.Sqrt(normSum))

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
	cacheDir         string
	models           map[string]Embedder
	mu               sync.RWMutex
	onnxIntraThreads int
	onnxInterThreads int
	useGPU           bool
	gpuEP            string            // specific EP if desired: "cuda", "coreml", "tensorrt", "directml"
	configFiles      map[string]string // model name -> custom filename
	configDims       map[string]int    // model name -> dimension
	poolingConfigs   map[string]string // model name -> pooling strategy
	aliases          map[string]string // alias -> primary name
	lruCache         *LRUCache
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
		cacheDir:       cacheDir,
		models:         make(map[string]Embedder),
		configFiles:    make(map[string]string),
		configDims:     make(map[string]int),
		poolingConfigs: make(map[string]string),
		aliases:        make(map[string]string),
	}
}

func (m *Manager) SetGPU(useGPU bool, ep string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.useGPU = useGPU
	m.gpuEP = strings.ToLower(ep)
}

func (m *Manager) SetLRUCache(size int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if size > 0 {
		m.lruCache = NewLRUCache(size)
	} else {
		m.lruCache = nil
	}
}

func (m *Manager) SetModelConfig(name, filename string, dim int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configFiles[name] = filename
	m.configDims[name] = dim
}

func (m *Manager) AddAlias(primary, alias string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.aliases[alias] = primary
}

func (m *Manager) SetPoolingConfig(name, pooling string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.poolingConfigs[name] = pooling
}

func (m *Manager) SetOnnxOptions(intra, inter int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onnxIntraThreads = intra
	m.onnxInterThreads = inter
}

func (m *Manager) resolveName(name string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if primary, ok := m.aliases[name]; ok {
		return primary
	}
	return name
}

func (m *Manager) GetEmbedder(requestedName string) (Embedder, error) {
	name := m.resolveName(requestedName)

	m.mu.RLock()
	emb, ok := m.models[name]
	intra := m.onnxIntraThreads
	inter := m.onnxInterThreads
	useGPU := m.useGPU
	gpuEP := m.gpuEP
	customFile := m.configFiles[name]
	dim := m.configDims[name]
	pooling := m.poolingConfigs[name]
	m.mu.RUnlock()
	if ok {
		return emb, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if emb, ok = m.models[name]; ok {
		return emb, nil
	}

	// Map name to folder
	var path string
	var possibleFolders []string

	if name == "multilingual-e5-small" {
		possibleFolders = append(possibleFolders, "fast-multilingual-e5-small", "models--qdrant--multilingual-e5-small")
	} else if name == "BAAI/bge-small-en-v1.5" {
		possibleFolders = append(possibleFolders, "fast-bge-small-en-v1.5", "models--BAAI--bge-small-en-v1.5")
	}

	folderName := "models--" + strings.ReplaceAll(name, "/", "--")
	if !strings.Contains(name, "/") {
		folderName = "models--qdrant--" + name
	}
	possibleFolders = append(possibleFolders, folderName)

	for _, f := range possibleFolders {
		testPath := filepath.Join(m.cacheDir, f)
		if _, err := os.Stat(testPath); err == nil {
			path = testPath
			break
		}
	}

	if path == "" {
		return nil, fmt.Errorf("model folder not found for %s in %s (tried %v)", name, m.cacheDir, possibleFolders)
	}

	if dim == 0 {
		dim = 384 // Fallback
	}

	onnxFile := "model.onnx"
	if customFile != "" {
		onnxFile = customFile
	}

	fullOnnxPath := filepath.Join(path, onnxFile)

	newEmb, err := NewCustomEmbedderWithFile(path, fullOnnxPath, dim, intra, inter, useGPU, gpuEP)
	if err != nil {
		return nil, err
	}

	if pooling != "" {
		newEmb.SetPooling(pooling)
	}

	m.models[name] = newEmb
	return newEmb, nil
}

func NewCustomEmbedderWithFile(modelPath, onnxPath string, dim int, intraThreads, interThreads int, useGPU bool, gpuEP string) (*CustomEmbedder, error) {
	// 1. Load Tokenizer
	tk, err := pretrained.FromFile(filepath.Join(modelPath, "tokenizer.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to load tokenizer at %s: %w", modelPath, err)
	}

	maxLen := 512
	tk.WithTruncation(nil)

	// 2. Initialize ONNX Environment
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

	// GPU Acceleration
	if useGPU {
		ep := gpuEP
		if ep == "" {
			// Auto-detect based on platform
			switch runtime.GOOS {
			case "darwin":
				ep = "coreml"
			case "windows":
				ep = "directml"
			default:
				ep = "cuda"
			}
		}

		switch ep {
		case "coreml":
			// 0 = default flags
			err = options.AppendExecutionProviderCoreML(0)
		case "cuda":
			// We can pass default CUDA options
			cudaOptions, _ := ort.NewCUDAProviderOptions()
			defer cudaOptions.Destroy()
			err = options.AppendExecutionProviderCUDA(cudaOptions)
		case "tensorrt":
			trtOptions, _ := ort.NewTensorRTProviderOptions()
			defer trtOptions.Destroy()
			err = options.AppendExecutionProviderTensorRT(trtOptions)
		case "directml":
			err = options.AppendExecutionProviderDirectML(0)
		default:
			err = fmt.Errorf("unsupported execution provider: %s", ep)
		}

		if err != nil {
			fmt.Printf("Warning: Failed to enable GPU acceleration (%s): %v. Falling back to CPU.\n", ep, err)
		} else {
			fmt.Printf("Enabled GPU acceleration using %s provider\n", ep)
		}
	}

	// 3. Prepare pre-allocated buffers
	inputShape := ort.NewShape(1, int64(maxLen))
	inputIds := make([]int64, maxLen)
	attentionMask := make([]int64, maxLen)
	tokenTypeIds := make([]int64, maxLen)

	t1, err := ort.NewTensor(inputShape, inputIds)
	if err != nil {
		return nil, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	t2, err := ort.NewTensor(inputShape, attentionMask)
	if err != nil {
		t1.Destroy()
		return nil, fmt.Errorf("failed to create attention_mask tensor: %w", err)
	}
	t3, err := ort.NewTensor(inputShape, tokenTypeIds)
	if err != nil {
		t1.Destroy()
		t2.Destroy()
		return nil, fmt.Errorf("failed to create token_type_ids tensor: %w", err)
	}

	outputShape := ort.NewShape(1, int64(maxLen), int64(dim))
	outputData := make([]float32, 1*maxLen*dim)
	tOut, err := ort.NewTensor(outputShape, outputData)
	if err != nil {
		t1.Destroy()
		t2.Destroy()
		t3.Destroy()
		return nil, fmt.Errorf("failed to create output tensor: %w", err)
	}

	// 4. Create Session (dynamically detect required inputs)
	// Some models only have input_ids and attention_mask, others have token_type_ids
	// We check the model's actual requirements to avoid ONNX runtime panics.
	tempSession, err := ort.NewAdvancedSession(
		onnxPath,
		[]string{"input_ids", "attention_mask", "token_type_ids"},
		[]string{"last_hidden_state"},
		[]ort.ArbitraryTensor{t1, t2, t3},
		[]ort.ArbitraryTensor{tOut},
		options,
	)

	// If the above fails, try without token_type_ids (common for RoBERTa/XLM-R)
	if err != nil {
		tempSession, err = ort.NewAdvancedSession(
			onnxPath,
			[]string{"input_ids", "attention_mask"},
			[]string{"last_hidden_state"},
			[]ort.ArbitraryTensor{t1, t2},
			[]ort.ArbitraryTensor{tOut},
			options,
		)
	}

	if err != nil {
		t1.Destroy()
		t2.Destroy()
		t3.Destroy()
		tOut.Destroy()
		return nil, fmt.Errorf("failed to create session for %s: %w", onnxPath, err)
	}
	session := tempSession

	return &CustomEmbedder{
		tokenizer:     tk,
		session:       session,
		dim:           dim,
		maxLen:        maxLen,
		pooling:       "mean", // Default
		inputIds:      inputIds,
		attentionMask: attentionMask,
		tokenTypeIds:  tokenTypeIds,
		outputData:    outputData,
		tensors:       []ort.ArbitraryTensor{t1, t2, t3, tOut},
	}, nil
}

func NewCustomEmbedder(modelPath string, dim int, intraThreads, interThreads int, useGPU bool, gpuEP string) (*CustomEmbedder, error) {
	return NewCustomEmbedderWithFile(modelPath, filepath.Join(modelPath, "model.onnx"), dim, intraThreads, interThreads, useGPU, gpuEP)
}

func (m *Manager) Embed(requestedName string, docs []string) ([][]float32, error) {
	name := m.resolveName(requestedName)
	emb, err := m.GetEmbedder(name)
	if err != nil {
		return nil, err
	}

	results := make([][]float32, len(docs))
	var toEmbed []string
	var toEmbedIndices []int

	m.mu.RLock()
	cache := m.lruCache
	m.mu.RUnlock()

	for i, doc := range docs {
		if cache != nil {
			key := name + ":" + doc
			if val, ok := cache.Get(key); ok {
				results[i] = val
				continue
			}
		}
		toEmbed = append(toEmbed, doc)
		toEmbedIndices = append(toEmbedIndices, i)
	}

	if len(toEmbed) > 0 {
		embedded, err := emb.Embed(toEmbed)
		if err != nil {
			return nil, err
		}

		for i, val := range embedded {
			origIdx := toEmbedIndices[i]
			results[origIdx] = val
			if cache != nil {
				key := name + ":" + toEmbed[i]
				cache.Add(key, val)
			}
		}
	}

	return results, nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, emb := range m.models {
		emb.Close()
	}
	m.models = make(map[string]Embedder)
}

func (m *Manager) GetActiveModels() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	active := make([]string, 0, len(m.models))
	for name := range m.models {
		active = append(active, name)
	}
	return active
}
