# Phase 2: Core-Entwicklung (Go)

Implementierung der Hauptlogik des Embedding-Microservices.

## Schritte
1.  **Modell-Manager:** Singleton-Pattern implementieren, das das Modell beim Starten aus `~/mlcembed` lädt.
2.  **API Endpunkte:**
    *   `POST /api/embed`: Mapping des JSON-Bodys auf die `model.Embed()` Funktion.
    *   `GET /api/tags`: Statische JSON-Antwort mit Metadaten der erlaubten Modelle.
3.  **Error Handling:** Rückgabe von Ollama-konformen Fehlermeldungen, falls das Modell nicht gefunden wird oder die Eingabe zu lang ist.

## Komponenten
*   `internal/embedding`: Wrapper für `fastembed-go`.
*   `internal/api`: Gin-Handler für die Endpunkte.
