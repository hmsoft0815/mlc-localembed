# Phase 1: Vorbereitung (Online-Schritt)

Da das Zielsystem kein Internet hat, müssen die Modelle einmalig geladen werden.

## Schritte
1.  **Download Script:** Ein kleines Go-Tool schreiben, das `fastembed.NewFlagEmbedding` mit einem spezifischen `CacheDir` aufruft.
2.  **Struktur sichern:** Den Inhalt des Cache-Verzeichnisses (z.B. `models--qdrant--multilingual-e5-small`) sichern.
3.  **Verschieben:** Daten auf das Zielsystem nach `~/mlcembed` kopieren.

## Geplante Tools
*   `cmd/preloader/main.go`: Ein Hilfstool zum Herunterladen der Modelle in das Cache-Verzeichnis.
