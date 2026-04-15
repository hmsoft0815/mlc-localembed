# 🛠 Tech Stack & Constraints: LocalEmbed

## Kern-Versionen
- **Sprache:** Go 1.25.0
- **Framework:** Gin Gonic v1.12.0
- **Runtime:** ONNX Runtime v1.17.0 (via `onnxruntime_go` v1.7.0)
- **Tokenization:** `github.com/daulet/tokenizers` v1.26.0 (Rust-based C-API)

## Bibliotheken (Erlaubt/Fixiert)
- **API:** Gin Gonic (REST)
- **GUI:** `systray` v1.2.2 (Cross-platform System Tray)
- **Config:** `gopkg.in/yaml.v3`
- **Testing:** `testify` v1.11.1

## Einschränkungen (Constraints)
- **CPU-First Strategie:** Fokus auf Optimierung für CPUs (insb. Apple Silicon), GPU (CUDA/CoreML) ist optional.
- **Single Binary (Mostly):** Ziel ist eine einfache Verteilung als Binary, wobei native Bibliotheken (`.so`, `.dylib`, `.dll`) mitgeliefert werden.
- **Xeon Freeze Protection:** Standardmäßige Limitierung von `intra_op_num_threads` in der `config.yaml`, um Systemhänger auf High-Core Servern zu vermeiden.
- **Kein Python:** Vermeidung von Python-Dependency-Hell durch native Go/C++/Rust Integration.

## Styling-Regeln
- **Go Standards:** Konsequente Nutzung von `go fmt`.
- **Error Handling:** Explizites Go Error Handling, keine Suppress-Hacks.
- **Threading:** Bewusster Umgang mit ONNX-Threading-Parametern zur Systemstabilität.

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI
- **Status:** Aktuell
