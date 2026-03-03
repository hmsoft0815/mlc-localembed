markdown
# Projektplan: Go-Embedding-Microservice (Ollama Simulation)

## 1. Zielsetzung
Erstellung eines leichtgewichtigen Microservices in Go, der:
*   Die Ollama-Endpunkte `/api/embed` und `/api/tags` bereitstellt.
*   Das Modell `intfloat/multilingual-e5-small` (oder large) via `fastembed-go` nutzt.
*   Vollständig **offline** (Air-Gapped) funktioniert.
*   Modelle aus einem lokalen Verzeichnis (`~/mlcembed`) lädt.

## 2. Architektur & Komponenten
*   **Sprache:** Go 1.21+
*   **Framework:** [Gin Gonic](https://github.com) (für schnelles HTTP-Routing).
*   **Embedding Engine:** [fastembed-go](https://github.com) (nutzt ONNX Runtime).
*   **Modell:** `multilingual-e5-small` (384 Dimensionen, ~450MB).

## 3. Phasenmodell

### Phase 1: Vorbereitung (Online-Schritt)
Da das Zielsystem kein Internet hat, müssen die Modelle einmalig geladen werden.
1.  **Download Script:** Ein kleines Go-Tool schreiben, das `fastembed.NewFlagEmbedding` mit einem spezifischen `CacheDir` aufruft.
2.  **Struktur sichern:** Den Inhalt des Cache-Verzeichnisses (z.B. `models--qdrant--multilingual-e5-small`) sichern.
3.  **Verschieben:** Daten auf das Zielsystem nach `~/mlcembed` kopieren.

### Phase 2: Core-Entwicklung (Go)
1.  **Modell-Manager:** Singleton-Pattern implementieren, das das Modell beim Starten aus `~/mlcembed` lädt.
2.  **API Endpunkte:**
    *   `POST /api/embed`: Mapping des JSON-Bodys auf die `model.Embed()` Funktion.
    *   `GET /api/tags`: Statische JSON-Antwort mit Metadaten der erlaubten Modelle.
3.  **Error Handling:** Rückgabe von Ollama-konformen Fehlermeldungen, falls das Modell nicht gefunden wird oder die Eingabe zu lang ist.

### Phase 3: Optimierung & Deployment
1.  **Logging:** Implementierung von Middleware zur Überwachung der Latenz pro Embedding-Request.
2.  **Dockerisierung (Optional):** Erstellung eines Dockerfiles, das das lokale Modell-Verzeichnis per `VOLUME` einbindet.
3.  **Validierung:** Test mit Tools wie `curl` oder Python (LangChain), um die Kompatibilität zur echten Ollama-API zu prüfen.

## 1. Verzeichnisstruktur (Standard Go Layout)
```text
.
├── bin/                    # Hier landen die fertigen Binaries (gitignored)
├── cmd/
│   ├── server/
│   │   └── main.go         # API-Server (Ollama Simulation)
│   └── cli/
│       └── main.go         # CLI-Tool zum Testen lokaler Embeddings
├── internal/
│   ├── api/                # HTTP-Handler & Routen
│   ├── embedding/          # Logik für fastembed-go Wrapper
│   └── logger/             # Zentrales Logging-Setup (Zap oder Logrus)
├── scripts/
│   └── build.sh            # Build-Script für beide Binaries
├── go.mod
└── go.sum

2. Komponenten-Details
A. Logging (Internal)
Verwendung von Uber-Zap oder dem Standard slog (ab Go 1.21).
Request-Logging: Jede Anfrage an /api/embed loggt:
Timestamp, Methode, Pfad.
Dauer der Vektorberechnung (Latenz).
Anzahl der Input-Tokens (Länge des Strings).
Client-IP.
B. CMD: Server (/api/embed & /api/tags)
Implementiert das Gin-Framework.
Lädt das Modell aus ~/mlcembed beim Start.
Simuliert die Ollama-JSON-Struktur 1:1.
C. CMD: CLI (Test-Tool)
Ein einfaches Tool, um ohne HTTP-Server schnell Vektoren in der Shell zu prüfen.
Aufruf: ./bin/cli --text "Hausbau in Deutschland".
3. Implementierungsschritte (Der "Offline-Weg")
Phase "Download" (Online):
Einmaliger Lauf eines Hilfsscripts, um die Dateien von Hugging Face nach ~/mlcembed zu ziehen.
Phase "Core":
Erstellung des internal/embedding Pakets, das prüft, ob das Modell lokal vorhanden ist (os.Stat).
Phase "API":
Bau des Gin-Servers mit einer Middleware für das Request-Logging.
Phase "Build":
go build -o bin/server ./cmd/server/main.go
4. Beispiel für das Request-Logging (slog)
In deinem Server-Code wird dies etwa so aussehen:
go
start := time.Now()
// ... Embedding Logik ...
slog.Info("Embedding Request",
    "method", "POST",
    "path", "/api/embed",
    "duration", time.Since(start),
    "input_length", len(req.Input),
)

Da du eine Air-Gapped-Umgebung planst, ist eine zentrale config.yaml oder config.json Gold wert, um Pfade und erlaubte Modelle zu steuern.
Hier ist die kuratierte Liste der Modelle, die du für den Start (Multilingual & Performance) in deine Konfiguration aufnehmen solltest:
1. Empfohlene Modell-Liste (Initialer Load)
Modell-ID (FastEmbed)	Typ	Dimension	Fokus	RAM-Bedarf
multilingual-e5-small	E5	384	Standard (DE/EN), extrem schnell	~600 MB
multilingual-e5-large	E5	1024	Höchste Präzision für komplexe Texte	~2.5 GB
bge-small-en-v1.5	BGE	384	Nur Englisch, aber Weltklasse Speed	~400 MB
paraphrase-multilingual-miniLM-L12-v2	ST	384	Sehr robust bei kurzen Sätzen/Phrasen	~500 MB
nomic-embed-text-v1.5	Nomic	768	Matryoshka-Modell (Vektorgröße variabel)	~1.2 GB
2. Beispiel: config.yaml
So könnte deine Konfiguration aussehen, um den Server und die CLI zu steuern:
yaml
server:
  port: 8080
  log_level: "info" # debug, info, warn, error

storage:
  cache_dir: "/home/user/mlcembed" # Dein ~/mlcembed Pfad
  allow_offline_download: false   # Verhindert Internet-Zugriff zur Laufzeit

models:
  default: "multilingual-e5-small"
  available:
    - name: "multilingual-e5-small"
      dim: 384
      description: "Fast multilingual embedding"
    - name: "multilingual-e5-large"
      dim: 1024
      description: "High precision multilingual embedding"
Use code with caution.

3. Warum die Config wichtig ist
In einer Offline-Umgebung dient die Config als "Source of Truth":
Validation: Der /api/tags Endpunkt liest einfach das models.available Array aus der Config und zeigt es an.
Sicherheit: Wenn ein Request ein Modell anfragt, das nicht in der Liste steht, blockst du es sofort ab, bevor die Library versucht, im leeren Cache-Ordner zu suchen.
Flexibilität: Du kannst neue Modelle einfach per Dateitransfer in den Ordner schieben und die config.yaml erweitern, ohne den Code neu kompilieren zu müssen.

Da du eine Air-Gapped-Umgebung planst, ist eine zentrale config.yaml oder config.json Gold wert, um Pfade und erlaubte Modelle zu steuern.
Hier ist die kuratierte Liste der Modelle, die du für den Start (Multilingual & Performance) in deine Konfiguration aufnehmen solltest:
1. Empfohlene Modell-Liste (Initialer Load)
Modell-ID (FastEmbed)	Typ	Dimension	Fokus	RAM-Bedarf
multilingual-e5-small	E5	384	Standard (DE/EN), extrem schnell	~600 MB
multilingual-e5-large	E5	1024	Höchste Präzision für komplexe Texte	~2.5 GB
bge-small-en-v1.5	BGE	384	Nur Englisch, aber Weltklasse Speed	~400 MB
paraphrase-multilingual-miniLM-L12-v2	ST	384	Sehr robust bei kurzen Sätzen/Phrasen	~500 MB
nomic-embed-text-v1.5	Nomic	768	Matryoshka-Modell (Vektorgröße variabel)	~1.2 GB
2. Beispiel: config.yaml
So könnte deine Konfiguration aussehen, um den Server und die CLI zu steuern:
yaml
server:
  port: 8080
  log_level: "info" # debug, info, warn, error

storage:
  cache_dir: "/home/user/mlcembed" # Dein ~/mlcembed Pfad
  allow_offline_download: false   # Verhindert Internet-Zugriff zur Laufzeit

models:
  default: "multilingual-e5-small"
  available:
    - name: "multilingual-e5-small"
      dim: 384
      description: "Fast multilingual embedding"
    - name: "multilingual-e5-large"
      dim: 1024
      description: "High precision multilingual embedding"
Use code with caution.

3. Warum die Config wichtig ist
In einer Offline-Umgebung dient die Config als "Source of Truth":
Validation: Der /api/tags Endpunkt liest einfach das models.available Array aus der Config und zeigt es an.
Sicherheit: Wenn ein Request ein Modell anfragt, das nicht in der Liste steht, blockst du es sofort ab, bevor die Library versucht, im leeren Cache-Ordner zu suchen.
Flexibilität: Du kannst neue Modelle einfach per Dateitransfer in den Ordner schieben und die config.yaml erweitern, ohne den Code neu kompilieren zu müssen.

.. initialie yaml:

´´´yaml
storage:
  cache_dir: "./mlcembed" # Hier landen die Modelle

models:
  - "multilingual-e5-small"
  - "multilingual-e5-large"
  - "nomic-embed-text-v1.5"
´´´

Das Pre-loader Script (cmd/preloader/main.go)
Dieses Tool nutzt die Library selbst, um den Download-Mechanismus zu triggern.
go
package main

import (
	"fmt"
	"log"
	"os"

	"://github.com"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Storage struct {
		CacheDir string `yaml:"cache_dir"`
	} `yaml:"storage"`
	Models []string `yaml:"models"`
}

func main() {
	// 1. Config einlesen
	configFile, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Fehler beim Lesen der config.yaml: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		log.Fatalf("Fehler beim Parsen der YAML: %v", err)
	}

	fmt.Printf("Starte Pre-loading in Verzeichnis: %s\n", config.Storage.CacheDir)

	// 2. Jedes Modell initialisieren (löst Download aus)
	for _, modelName := range config.Models {
		fmt.Printf("--- Lade Modell: %s ---\n", modelName)

		options := fastembed.InitOptions{
			Model:                fastembed.Model(modelName),
			CacheDir:             config.Storage.CacheDir,
			ShowDownloadProgress: true,
		}

		// NewFlagEmbedding triggert intern den Download, falls nicht vorhanden
		_, err := fastembed.NewFlagEmbedding(&options)
		if err != nil {
			fmt.Printf("Fehler beim Laden von %s: %v\n", modelName, err)
			continue
		}

		fmt.Printf("Erfolgreich geladen: %s\n", modelName)
	}

	fmt.Println("\nAlle Modelle sind im Cache-Verzeichnis bereit für den Offline-Transfer.")
}

