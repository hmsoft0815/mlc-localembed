<p align="center">
  <img src="docs/minilogo884x484.png" width="300" alt="mlc-localembed logo">
</p>

# mlc-localembed v0.2.0

[![Go Version](https://img.shields.io/github/go-mod/go-version/hmsoft0815/mlc-localembed)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Air Gapped](https://img.shields.io/badge/Sicherheit-Air--Gapped-blue.svg)](#motivation)

LocalEmbed ist ein leistungsstarker Go-basierter Dienst zur Erzeugung von Texteinbettungen (Embeddings) unter Verwendung der ONNX-Runtime. Es bietet eine Ollama-kompatible API zur einfachen Integration in bestehende AI-Workflows.

## Motivation

Das Hauptziel von LocalEmbed ist es, eine **kosteneffiziente und effiziente Infrastruktur** für AI-Anwendungen bereitzustellen. Während leistungsstarke Dienste wie Ollama oder kommerzielle APIs hervorragend für den Betrieb großer Sprachmodelle (LLMs) wie Gemma, Llama oder GPT-4 geeignet sind, kann deren Nutzung für umfangreiche Embedding-Aufgaben teuer sein oder unnötige Latenzen verursachen.

Wir nutzen dieses Tool intern für **RAG (Retrieval-Augmented Generation)** Workflows. Bei RAG müssen Dokumente häufig eingebettet und indiziert werden, um dem LLM Kontext bereitzustellen. Durch die lokale Verarbeitung dieser Embeddings erreichen Sie:
- **Golang statt Python**: Die meisten Embedding-Bibliotheken basieren auf Python, was oft zur "Dependency Hell" führt (Versionskonflikte, Management virtueller Umgebungen, riesige Container-Images). Go ermöglicht es uns, ein einzelnes, hochperformantes Binary mit minimalen externen Abhängigkeiten zu verteilen.
- **Kosteneinsparungen**: Keine Kosten pro Token für das Einbetten großer Datensätze zur Indizierung.
- **Effizienz**: Die optimierte ONNX-Ausführung ist für kleine Embedding-Modelle oft schneller als allgemeine LLM-Runner.
- **Trennung der Zuständigkeiten**: Halten Sie Ihre "schwere" LLM-Verarbeitung getrennt von Ihren "hochfrequenten" Embedding-Aufgaben.
- **CPU-First Strategie**: Wir setzen bewusst auf die CPU-Ausführung. Während GPU-Unterstützung jederzeit möglich wäre, erlaubt die CPU-Optimierung den Betrieb auf nahezu jeder Hardware – von modernen M1/M2-Chips bis hin zu ausgedienten Servern oder günstigen Mini-PCs, ohne dass teure GPUs oder komplexe Treiber-Setups nötig sind.

Dies sollte auf moderner Hardware (z.B. M1 etc.) kein Problem sein – aber seien Sie gewarnt, ältere CPUs könnten bei einigen Modellen Probleme verursachen.

## Features

- **Schnell & Leichtgewichtig**: Entwickelt in Go mit ONNX Runtime für minimalen Overhead.
- **Ollama-kompatibel**: Unterstützt `/api/embed` und `/api/tags` Endpunkte.
- **Konfigurierbar**: Modelle und Runtime-Einstellungen lassen sich einfach über YAML verwalten.
- **Ressourcen-Management**: Integrierter Schutz für Multi-Core-Systeme (Xeon Freeze-Schutz).
- **Netzwerk-Isolation**: 100% Air-Gapped Laufzeit, sobald die Modelle vorgeladen sind.

## Konfiguration

Die Einstellungen werden über die `config.yaml` oder Umgebungsvariablen (diese haben Vorrang) verwaltet.

### Umgebungsvariablen

| Variable | Beschreibung | Standardwert |
|----------|-------------|---------|
| `MLC_PORT` | Server-Port | `9142` |
| `MLC_LOG_LEVEL` | Log-Level (`info`, `debug`) | `info` |
| `MLC_CACHE_DIR` | Pfad zum Modell-Cache | `./mlcembed` |
| `MLC_INTRA_THREADS` | ONNX intra-op Threads | Aus Config |
| `MLC_INTER_THREADS` | ONNX inter-op Threads | Aus Config |
| `MLC_DEFAULT_MODEL` | Standard-Modellname | Aus Config |

### config.yaml

```yaml
onnx:
  intra_op_num_threads: 4 # Threads limitieren, um System-Einfrieren zu verhindern
  inter_op_num_threads: 4

models:
  available:
    - name: "multilingual-e5-small"
      enabled: true
    - name: "BAAI/bge-small-en-v1.5"
      enabled: false # Standardmäßig deaktiviert wegen fehlender Intel VNNI Unterstützung
```

> **Hinweis**: Ich teste derzeit nur mit den oben aufgeführten Modellen. Falls Sie weitere Modelle finden, die mit dieser Infrastruktur gut funktionieren, freue ich mich über einen Hinweis! Gerne nehme ich diese dann in die Standard-Konfiguration mit auf.

### Technische Hinweise

- **Xeon Freeze-Schutz**: Auf einigen Systemen mit vielen Kernen (z. B. Xeon-Prozessoren) versucht die ONNX-Runtime unter Umständen, alle verfügbaren Kerne zu nutzen, was zu Systeminstabilität führen kann. Wir limitieren `intra_op_num_threads` standardmäßig, um einen stabilen Betrieb zu gewährleisten.
- **Intel VNNI Unterstützung**: Einige optimierte Modelle benötigen den Intel VNNI Befehlssatz. Wenn Ihre CPU dies nicht unterstützt (z. B. ältere Prozessoren oder Nicht-Intel CPUs), können bestimmte Modelle fehlschlagen oder langsam sein. Diese Modelle sind standardmäßig deaktiviert. Wenn ich es getestet habe, habe ich sogar meinen Xeon-Server zum Absturz gebracht (der keine Unterstützung dafür hatte).

## Benutzung

Das Projekt umfasst drei Hauptwerkzeuge im Verzeichnis `bin/`:

### 1. Preloader (`bin/preloader`)
Lädt Modelle von Hugging Face basierend auf Ihrer `config.yaml` herunter und bereitet sie vor.
```bash
# Optional: Hugging Face Token für geschützte Modelle angeben
export HF_TOKEN=ihr_token_hier
./bin/preloader
```

### 2. Server (`bin/server`)
Startet den Ollama-kompatiblen API-Server.
```bash
./bin/server
```
Der Server ist standardmäßig unter `http://localhost:9142` erreichbar.

> **Ollama Drop-in Replacement**: Um dies als Ersatz für den Ollama-Embedding-Dienst in bestehenden Tools zu nutzen, können Sie entweder die Konfiguration Ihres Tools auf Port `9142` ändern oder `MLC_PORT=11434` (Ollamas Standard-Port) setzen, bevor Sie den Server starten.

### 3. CLI-Tool (`bin/cli`)
Ein einfaches Werkzeug, um Embeddings direkt über die Kommandozeile zu testen.
```bash
./bin/cli -text "Ihr Text hier" -model "multilingual-e5-small"
```

## API Endpunkte

- `POST /api/embed`: Erzeugt Embeddings für einen oder mehrere Strings (Ollama-kompatibel).
- `GET /api/tags`: Listet verfügbare und aktivierte Modelle auf (Ollama-kompatibel).
- `GET /api/health`: Einfacher Gesundheitscheck.
- `POST /api/test/similarity`: **(Neu in v0.2.0)** Direkter Vergleich einer Suchanfrage mit mehreren Dokumenten, um semantische Ähnlichkeitswerte zu erhalten.

### Similarity API Beispiel

```bash
curl -X POST http://localhost:9142/api/test/similarity \
  -H "Content-Type: application/json" \
  -d '{
    "model": "multilingual-e5-small",
    "query": "Was ist die Hauptstadt von Frankreich?",
    "documents": ["Paris ist die Hauptstadt.", "Berlin liegt in Deutschland.", "Die Sonne ist ein Stern."]
  }'
```
**Antwort:**
```json
{
  "model": "multilingual-e5-small",
  "scores": [
    { "document": "Paris ist die Hauptstadt.", "score": 0.9115 },
    { "document": "Berlin liegt in Deutschland.", "score": 0.8593 },
    { "document": "Die Sonne ist ein Stern.", "score": 0.8685 }
  ]
}
```

## Tests

Das Projekt umfasst sowohl Unit-Tests als auch Integrationstests.

### Unit-Tests (Go)
Führen Sie alle internen Tests aus:
```bash
go test -v ./...
```

### Integrationstests (Bash/Curl)
Erfordert einen laufenden Server.
```bash
./tests/run_curl_tests.sh
```

## Support & Consulting

Falls Sie professionelle Unterstützung, maßgeschneiderte Modell-Integrationen oder Hilfe bei Enterprise-Deployments benötigen, können Sie mich gerne kontaktieren. Ich biete Beratung für:

- **Performance-Optimierung**: Tuning für spezifische Server-Hardware (z. B. High-Core Xeon-Systeme).
- **Custom Models**: Integration und Optimierung spezieller ONNX-Embedding-Modelle.
- **Enterprise RAG**: Architektur-Design und Integration in bestehende RAG-Workflows.

Für Support-Anfragen öffnen Sie bitte ein **GitHub Issue** oder kontaktieren Sie mich über mein **GitHub-Profil**.

## Ähnliche Projekte & Credits

Obwohl LocalEmbed nun eine unabhängige Implementierung ist, wurde es von den Ideen anderer großartiger Projekte inspiriert. Wenn Sie nach Alternativen oder den ursprünglichen Bibliotheken suchen, schauen Sie sich diese an:

- [fastembed-go](https://github.com/anush008/fastembed-go) - Die ursprüngliche Go-Implementierung, die dieses Projekt inspiriert hat.
- [fastembed](https://github.com/qdrant/fastembed) - Die hocheffiziente Python-Bibliothek von Qdrant.
- [onnxruntime-go](https://github.com/yalue/onnxruntime_go) - Die essenziellen Go-Bindings für ONNX.
- [tokenizer](https://github.com/sugarme/tokenizer) - Hervorragende Go-Implementierung der Hugging Face Tokenizer.

## Danksagung

Dieses Projekt ist in erster Linie eine Infrastrukturschicht, die auf exzellenter Vorarbeit aufbaut. Ich bin dankbar, die folgenden Bibliotheken und Runtimes nutzen zu können:

- [onnxruntime-go](https://github.com/yalue/onnxruntime_go) für die performante Modellausführung.
- [tokenizer](https://github.com/sugarme/tokenizer) für die Textverarbeitung.

LocalEmbed konzentriert sich darauf, den notwendigen "Klebstoff" bereitzustellen (Ollama-kompatible API, Thread-Management und Modell-Preloading), um diese Werkzeuge in einer Serverumgebung leicht zugänglich zu machen.

## Lizenz

MIT - Copyright (c) 2026 Michael Lechner
