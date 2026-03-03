<p align="center">
  <img src="docs/minilogo884x484.png" width="300" alt="mlc-localembed logo">
</p>

# mlc-localembed v0.1.1

[![Go Version](https://img.shields.io/github/go-mod/go-version/hmsoft0815/mlc-localembed)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Air Gapped](https://img.shields.io/badge/Security-Air--Gapped-blue.svg)](#motivation)

LocalEmbed is a high-performance Go-based service for generating text embeddings using ONNX Runtime. It provides an Ollama-compatible API for easy integration into existing AI workflows.

## Motivation

The primary goal of LocalEmbed is to provide a **cost-effective and efficient infrastructure** for AI applications. While powerful services like Ollama or commercial APIs are excellent for running large language models (LLMs) like Gemma, Llama, or GPT-4, using them for high-volume embedding tasks can be expensive or introduce unnecessary latency.

We use this tool internally for **RAG (Retrieval-Augmented Generation)** workflows. In RAG, documents must be embedded and indexed frequently to provide context to the LLM. By handling embeddings locally:
- **Golang over Python**: Most embedding libraries are Python-based, which often leads to "dependency hell" (version conflicts, virtual environment management, large container images). Go allows us to distribute a single, high-performance binary with minimal external dependencies.
- **Cost Savings**: No per-token costs for embedding large datasets for indexing.
- **Efficiency**: Optimized ONNX execution is often faster for small embedding models than general-purpose LLM runners.
- **Separation of Concerns**: Keep your "heavy" LLM processing separate from your "high-frequency" embedding tasks.

## Features
...

- **Fast & Lightweight**: Built with Go and ONNX Runtime for minimal overhead.
- **Ollama Compatible**: Supports `/api/embed` and `/api/tags` endpoints.
- **Configurable**: Easily manage models and runtime settings via YAML.
- **Resource Management**: Built-in protection for multi-core systems (Xeon freeze protection).

## Configuration

Settings are managed via `config.yaml` or environment variables (which take precedence).

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MLC_PORT` | Server port | `9000` |
| `MLC_LOG_LEVEL` | Log level (`info`, `debug`) | `info` |
| `MLC_CACHE_DIR` | Path to model cache | `./mlcembed` |
| `MLC_INTRA_THREADS` | ONNX intra-op threads | From config |
| `MLC_INTER_THREADS` | ONNX inter-op threads | From config |
| `MLC_DEFAULT_MODEL` | Default model name | From config |

### config.yaml

```yaml
onnx:
  intra_op_num_threads: 4 # Limit threads to prevent system freezes
  inter_op_num_threads: 4

models:
  available:
    - name: "multilingual-e5-small"
      enabled: true
    - name: "BAAI/bge-small-en-v1.5"
      enabled: false # Disabled by default due to Intel VNNI instruction requirements

> **Note**: I am currently only testing with the models listed above. If you discover other models that work well with this infrastructure, please let me know! I would be happy to include them in the default configuration.
```

### Technical Notes

- **Xeon Freeze Protection**: On some high-core systems (like Xeon processors), ONNX Runtime may attempt to use all available cores, causing system instability. We limit `intra_op_num_threads` by default to ensure stable operation.
- **Intel VNNI Support**: Some optimized models require the Intel VNNI instruction set. If your CPU does not support this (e.g., older processors or non-Intel CPUs), certain models may fail or perform poorly. These models are disabled by default. When testing I even brought my xeon server down (which had no support for this)

This should not be a problem on modern hardware (e.g. M1 etc) - but be warned, older CPUs might cause problems for
some models.

## Usage

The project includes three main tools in the `bin/` directory:

### 1. Preloader (`bin/preloader`)
Downloads and prepares models from Hugging Face based on your `config.yaml`.
```bash
# Optional: Provide Hugging Face token for restricted models
export HF_TOKEN=your_token_here
./bin/preloader
```

### 2. Server (`bin/server`)
Starts the Ollama-compatible API server.
```bash
./bin/server
```
The server will be available at `http://localhost:9000` (default).

### 3. CLI Tool (`bin/cli`)
A simple tool to test embeddings directly from the command line.
```bash
./bin/cli -text "Your text here" -model "multilingual-e5-small"
```

## API Endpoints

- `POST /api/embed`: Generate embeddings for one or more strings.
- `GET /api/tags`: List available and enabled models.
- `GET /api/health`: Basic health check.

## Support & Consulting

If you need professional support, custom model integrations, or enterprise-grade deployment assistance, feel free to reach out. I offer consulting services for:

- **Performance Optimization**: Tuning for specific server hardware (e.g., high-core Xeon systems).
- **Custom Models**: Integrating and optimizing specialized ONNX embedding models.
- **Enterprise RAG**: Architecture design and integration into your existing RAG workflows.

For support inquiries, please **open a GitHub Issue** or contact me via my **GitHub Profile**.

## Related Projects & Credits

While LocalEmbed is now an independent implementation, it was inspired by and built upon the ideas of other great projects. If you are looking for alternatives or the original libraries that paved the way, check these out:

- [fastembed-go](https://github.com/anush008/fastembed-go) - The original Go implementation that inspired this project.
- [fastembed](https://github.com/qdrant/fastembed) - The highly efficient Python library by Qdrant.
- [onnxruntime-go](https://github.com/yalue/onnxruntime_go) - The essential Go bindings for ONNX.
- [tokenizer](https://github.com/sugarme/tokenizer) - Excellent Go implementation of Hugging Face tokenizers.

## Acknowledgments

This project is primarily an infrastructure layer built on top of excellent existing work. I am grateful to be able to leverage the following libraries and runtimes:

- [onnxruntime-go](https://github.com/yalue/onnxruntime_go) for high-performance model execution.
- [tokenizer](https://github.com/sugarme/tokenizer) for text processing.

LocalEmbed focuses on providing the necessary "glue" (Ollama-compatible API, thread management, and model preloading) to make these tools easily accessible in a server environment.

## License

MIT - Copyright (c) 2026 Michael Lechner
