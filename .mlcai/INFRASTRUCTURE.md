# 🏗 Infrastructure & Deployment: LocalEmbed

## Hosting & Umgebungen

| Umgebung | URL / Adresse | Anmerkung |
|----------|--------------|-----------|
| Production (Linux) | `http://<server-ip>:9142` | Systemd-Service auf RHEL/Fedora/Ubuntu |
| Production (macOS) | `localhost:9142` | Native Swift StatusBar App + Go Backend |
| Production (Windows)| `localhost:9142` | System Tray App (systray) |
| Local Dev | `localhost:9142` | `task run` oder direktes Binary |

## Deployment

- **Methode:**
  - **Linux:** RPM-Package via `task rpm`, Installation via `dnf`/`yum`.
  - **macOS:** DMG-Installer via `task package-macos` oder `scripts/install.sh`.
  - **Windows:** NSIS-Installer (.exe) oder ZIP-Bundle via `task build`.
- **Start-Befehl:**
  - **Linux:** `sudo systemctl start localembed`
  - **macOS:** `launchctl load ~/Library/LaunchAgents/com.localembed.server.plist`
  - **Manual:** `./bin/mlcembedder -config config.yaml`
- **Config-Pfad:**
  - **Linux:** `/etc/localembed/config.yaml`
  - **macOS:** `/usr/local/etc/localembed/config.yaml`
  - **Windows:** `C:\ProgramData\LocalEmbed\config.yaml`
- **Logs:**
  - **Linux:** `journalctl -u localembed -f`
  - **macOS:** `tail -f /usr/local/var/log/localembed.log`
  - **Windows:** Sichtbar über das System Tray Menu oder in `C:\ProgramData\LocalEmbed\logs\`.

## Abhängigkeiten (Laufzeit)

- **ONNX Runtime:** Erfordert `libonnxruntime.so` (Linux), `.dylib` (macOS) oder `.dll` (Windows). Wird in Release-Packages mitgeliefert.
- **Tokenizers:** Statisch gelinkt via `libtokenizers.a`.
- **Modelle:** Erfordert heruntergeladene ONNX-Modelle im `cache_dir` (Standard: `./mlcembed`).
- **Hardware:** CPU-fokussiert (Intel VNNI empfohlen für optimierte Modelle), optional GPU (CUDA/CoreML) via ONNX-Provider.

## Secrets / Umgebungsvariablen

| Variable | Zweck | Standard |
|----------|-------|----------|
| `MLC_PORT` | API Port | `9142` |
| `MLC_LOG_LEVEL` | Logging Tiefe (`info`, `debug`) | `info` |
| `MLC_CACHE_DIR` | Pfad zu den Modellen | `./mlcembed` |
| `ONNX_PATH` | Pfad zur ONNX Library | Automatisch |
| `HF_TOKEN` | Hugging Face Token (nur für Preload) | — |

## Monitoring & Alerts

- **Health-Check:** `GET /api/health` (liefert Status "OK" und Version).
- **Statistiken:** `GET /api/stats` (liefert Uptime, Request-Counts und Modell-Latenzen).
- **Monitoring:** Integration in MLC Doc Hub Worklogs; lokal via `task ping`.

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-17
- **Aktualisiert von:** gemini-2.0-flash-exp
- **Status:** Aktuell
