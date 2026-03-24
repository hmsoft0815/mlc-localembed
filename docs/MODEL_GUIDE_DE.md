# Model Selection & Optimization Guide

Dieser Leitfaden hilft bei der Auswahl des richtigen Embedding-Modells für Ihre Anwendung (insbesondere für MCP-Server und RAG-Workflows) und erklärt, wie Sie die beste Genauigkeit erzielen.

## Übersicht der unterstützten Architekturen

### 1. E5 (Spezialist für semantische Suche)
Das E5-Design ("Enhanced English Encoders") wurde explizit für die Ähnlichkeitssuche entwickelt. Es ist besonders stark darin, die semantische Verbindung zwischen einer Frage und einem Dokument zu finden.
- **Stärke:** Exzellente Ergebnisse bei asymmetrischer Suche (kurze Frage -> langer Text).
- **Besonderheit:** Arbeitet mit Präfixen (`query:` und `passage:`), um den Kontext der Anfrage zu verstehen.

### 2. BGE (Sehr Effizient)
BGE-Modelle (Beijing Academy of Artificial Intelligence) gehören zu den leistungsfähigsten Open-Source-Modellen auf der MTEB-Bestenliste.
- **Stärke:** Extrem hohe Informationsdichte. Findet Übereinstimmungen oft schon bei sehr kurzen Tool-Namen oder technischen Begriffen.
- **Besonderheit:** Sehr stabil und effizient in der ONNX-Ausführung.

### 3. BERT / all-MiniLM (Der bewährte Allrounder)
Basierend auf der klassischen Transformer-Architektur, optimiert für Satz-Vektoren.
- **Stärke:** Extrem schnell, geringster Ressourcenverbrauch.
- **Einsatz:** Ideal für einfache Klassifizierungen oder wenn die Hardware sehr limitiert ist.

---

## Modell-Vergleich für MCP & RAG

| Feature | `all-MiniLM` | `multilingual-e5-small` | `bge-small-v1.5` |
| :--- | :--- | :--- | :--- |
| **Kurze Tool-Namen** | 🟡 Durchschnittlich | 🟢 Gut | 🏆 Bestwert |
| **Lange Beschreibungen** | 🟢 Gut | 🏆 Bestwert | 🟢 Gut |
| **Sprachmix (DE/EN)** | ❌ Eingeschränkt | 🏆 Bestwert | 🟡 Befriedigend |
| **Geschwindigkeit** | 🚀 Extrem schnell | 🟡 Moderat | 🟢 Schnell |
| **Hardware-Anspruch** | 📉 Minimal | 📈 Mittel | 📉 Gering |

---

## Best Practices für maximale Genauigkeit

### Automatische Optimierungen (seit v1.0.0)
Seit Version 1.0.0 übernimmt `mlc-localembed` wichtige Aufgaben automatisch, um Sie zu entlasten:
- **Special Token Handling:** `[CLS]` und `[SEP]` Token werden automatisch korrekt gesetzt.
- **Automatic Truncation:** Texte, die länger als 512 Token sind, werden automatisch gekürzt, anstatt einen Fehler zu verursachen. Dies entspricht dem Verhalten von Ollama.

### Manuelle Optimierungstipps

#### 1. Nutzung von Präfixen (E5 & BGE)
Um die volle Leistung der Modelle abzurufen, sollten Sie (je nach Modell-Konfiguration in der `config.yaml`) Präfixe verwenden:
- **E5:** Kennzeichnen Sie Suchanfragen mit `query: ` und die zu indexierenden Dokumente (oder Tool-Beschreibungen) mit `passage: `.
- **BGE:** Für Suchanfragen empfiehlt sich oft die Instruktion: `Represent this sentence for searching relevant passages: `.

#### 2. Sprachwahl
Wenn Ihre MCP-Tools englische Namen/Beschreibungen haben, Ihre Nutzer aber auf Deutsch fragen, ist das **Multilingual-E5-Small** die erste Wahl. Es kann semantische Brücken zwischen verschiedenen Sprachen schlagen.

#### 3. Normalisierung
Alle Vektoren werden von `mlc-localembed` automatisch L2-normalisiert. Das bedeutet, dass Sie für den Vergleich der Vektoren einfach das **Punktprodukt (Dot Product)** verwenden können, was rechnerisch effizienter ist als Cosine Similarity, aber zum selben Ergebnis führt.

---

## Fazit: Welches Modell für mein Projekt?

*   **MCP-Server (Standard):** Wählen Sie `multilingual-e5-small` für die beste Balance zwischen Sprachunterstützung und Genauigkeit.
*   **Technische Tools:** Wählen Sie `bge-small-v1.5`, wenn Sie hauptsächlich technische Dokumentationen oder Funktionsnamen matchen müssen.
*   **Edge-Computing / Alte Hardware:** Wählen Sie `all-minilm`, wenn Millisekunden wichtiger sind als die letzte Nuance an Genauigkeit.
