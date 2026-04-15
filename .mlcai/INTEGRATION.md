# 🤖 AI & Integration Context: LocalEmbed — Local Embedding Server

## 1. Identität & Zweck
- **Kernaufgabe:** Hochperformanter Go-basierter Service zur Generierung von Texteinbettungen (Embeddings) mittels ONNX Runtime. Bietet eine Ollama- und OpenAI-kompatible API.
- **Technischer Stack:** Go, Gin Gonic, ONNX Runtime, Hugging Face Tokenizers (Daulet-Bindings).
- **Hoster/Infrastruktur:** 
  - **Linux:** Systemd Service (RPM Package).
  - **macOS:** Launchd (Guided Installer/DMG).
  - **Windows:** Native System Tray Application (Portable/Installer).
  - **Docker:** Verfügbar für Tests und CI/CD.

## 2. Die "Nachbarschaft" (System-Kontext)
- **Upstream (Wovon hänge ich ab?):**
  - **Hugging Face** -> [huggingface.co](https://huggingface.co) -> Download der Modelle via `preloader`.
  - **ONNX Runtime** -> [microsoft/onnxruntime](https://github.com/microsoft/onnxruntime) -> Native Bibliotheken für Modell-Ausführung.
  - **Tokenizers** -> [daulet/tokenizers](https://github.com/daulet/tokenizers) -> Native Rust-Library für Text-Processing.
- **Downstream (Wer nutzt mich?):**
  - **RAG-Workflows:** Lokale Indizierung von Dokumenten ohne Cloud-Kosten.
  - **Ollama/OpenAI Clients:** Kann als Drop-in Replacement für Einbettungs-Aufgaben genutzt werden.
- **Shared Resources:**
  - Nutzt das Verzeichnis `./mlcembed` (oder konfigurierten Pfad) zum Cachen der Modelle.

## 3. Schnittstellen-Vertrag
- **Primäre API:** REST (Standardport 9142).
- **Auth-Mechanismus:** Keine (für interne/air-gapped Netze optimiert). Kann via Reverse Proxy (Caddy/Nginx) abgesichert werden.
- **Wichtige Datenmodelle:**
  - `Embedding`: Liste von Floats (Vektor).
- **API-Doku:** Beschrieben in `README.md` und `API_CONTRACT.md`.

## 4. Leitplanken & Regeln
- **Naming:** Go Standard (camelCase für Variablen, PascalCase für exportierte Typen).
- **Testing:** Unit-Tests via `go test ./...`, Integrationstests via `tests/run_curl_tests.sh`.
- **Sicherheit:** 100% Air-Gapped Runtime möglich, sobald Modelle via Preloader heruntergeladen wurden.

## 5. Aktueller Fokus (Status)
- **Status:** Version 1.0.2 (Stable).
- **Bekannte Themen:** Xeon "Freeze Protection" aktiv (Threading-Limits zur Stabilität).
- **Nächste Schritte:** Erweiterung der getesteten Modelle, Performance-Feintuning für Apple Silicon (M4/M5).

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI
- **Status:** Aktuell
