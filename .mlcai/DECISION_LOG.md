# 📜 Decision Log (ADR): LocalEmbed

## 2026-04-15: CPU-First Strategie
- **Kontext:** Einbettungsmodelle sind im Vergleich zu LLMs klein und effizient. Die Installation von GPU-Treibern (CUDA) ist oft komplex und fehleranfällig.
- **Entscheidung:** Wir priorisieren die CPU-Ausführung mittels ONNX Runtime.
- **Grund:** Moderne CPUs (insb. Apple Silicon M-Serie) bieten sub-millisekunden Latenz für Embeddings. Dies ermöglicht den Betrieb auf fast jeder Hardware ohne "Driver-Hell".
- **Konsequenz:** GPU-Unterstützung bleibt optional und muss explizit aktiviert werden.

## 2026-04-15: Xeon Freeze Protection
- **Kontext:** Auf Systemen mit sehr vielen Kernen (z.B. Dual-Xeon) versucht ONNX Runtime standardmäßig alle Kerne zu nutzen, was das gesamte System einfrieren kann ("Thread Overload").
- **Entscheidung:** Einführung von Standard-Limits für `intra_op_num_threads` in der Standard-Konfiguration.
- **Grund:** Gewährleistung der Systemstabilität auf Server-Hardware.
- **Konsequenz:** Nutzer mit speziellen Performance-Anforderungen müssen diese Werte manuell in der `config.yaml` erhöhen.

## 2026-04-15: Go statt Python
- **Kontext:** Die meisten KI-Tools basieren auf Python/PyTorch, was zu riesigen Docker-Images und Abhängigkeitskonflikten führt.
- **Entscheidung:** Vollständige Implementierung in Go mit nativen Bindings für ONNX und Tokenizers.
- **Grund:** Erstellung einer kompakten, performanten Single-Binary (plus Shared Libs), die leicht zu verteilen ist (RPM, DMG, EXE).
- **Konsequenz:** Höherer Initialaufwand bei den Bindings (CGO), aber deutlich einfacheres Deployment für Endnutzer.

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI
- **Status:** Aktuell
