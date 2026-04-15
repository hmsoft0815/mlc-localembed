# 🔌 API Contract: LocalEmbed

## 1. Bereitgestellte Endpunkte (Exposed)

### REST API (v1)
- **Basis-URL:** `/api` (Standard-Port: 9142)
- **Auth:** Keine (vorausgesetzt wird ein sicheres internes Netz oder Reverse Proxy)

#### Ollama Kompatibel
- `POST /api/embed`: Generiert Einbettungen für einen oder mehrere Texte.
- `GET /api/tags`: Listet alle verfügbaren und aktivierten Modelle auf.

#### OpenAI Kompatibel
- `POST /api/embeddings`: OpenAI-kompatibler Endpoint (unterstützt `prompt` Feld und Standard-Response-Format).

#### Utility & Monitoring
- `GET /api/health`: Status des Servers und Version.
- `GET /api/stats`: Nutzungsstatistiken (Requests, Uptime, Modell-Performance).
- `POST /api/test/similarity`: Custom Endpoint zum direkten Vergleich von Query vs. Dokumenten (Semantic Similarity).
- `POST /api/generate`: Dummy-Endpoint (501 Not Implemented), um Client-Crashes zu vermeiden.

---

## 2. Konsumierte APIs (Consumed)

### Extern: Hugging Face
- **Zweck:** Herunterladen von Modell-Gewichten und Tokenizer-Konfigurationen.
- **Tool:** `bin/preloader` nutzt die Hugging Face API.
- **Auth:** Optionaler `HF_TOKEN` für zugriffsbeschränkte Modelle.

---

## 3. Globale Konventionen
- **Vektoren:** Werden als Float-Arrays zurückgegeben.
- **Threading:** Die API ist konkurrierend ausgelegt (`MLC_MAX_CONCURRENCY` gesteuert).
- **Fehlerformat:** Standard JSON-Responses mit Fehlermeldung und HTTP Status Codes.

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI
- **Status:** Aktuell
