#!/usr/bin/env bash
#
# Native systemd install for LocalEmbed on non-RPM Linux (Debian/Ubuntu/...).
# Mirrors the RPM layout but uses /usr/local paths appropriate for a manual
# install. Run as root:  sudo bash scripts/install-linux-systemd.sh
#
# Idempotent: safe to re-run to upgrade binaries / refresh the unit.

set -euo pipefail

# --- resolve repo root (this script lives in <repo>/scripts) ---------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "$SCRIPT_DIR/.." && pwd)"

BIN_DIR=/usr/local/bin
LIB_DIR=/usr/local/lib/localembed
CFG_DIR=/etc/localembed
STATE_DIR=/var/lib/localembed
MODELS_DIR=$STATE_DIR/models
LOG_DIR=/var/log/localembed
UNIT=/etc/systemd/system/localembed.service
USER=localembed
GROUP=localembed

if [ "$(id -u)" -ne 0 ]; then
    echo "Please run as root:  sudo bash $0" >&2
    exit 1
fi

echo "==> Repo: $REPO"

# --- 1. service user/group --------------------------------------------------
getent group "$GROUP" >/dev/null || groupadd -r "$GROUP"
getent passwd "$USER" >/dev/null || \
    useradd -r -g "$GROUP" -d "$STATE_DIR" -s /usr/sbin/nologin \
            -c "MLC LocalEmbed Service User" "$USER"

# --- 2. binaries ------------------------------------------------------------
# Prefer prebuilt binaries in bin/; their on-disk names are mlcembedder/cli/preloader.
install -d "$BIN_DIR"
install -m 0755 "$REPO/bin/mlcembedder"  "$BIN_DIR/mlcembedder"
install -m 0755 "$REPO/bin/cli"          "$BIN_DIR/localembed-cli"
install -m 0755 "$REPO/bin/preloader"    "$BIN_DIR/localembed-preloader"

# --- 3. ONNX runtime shared library ----------------------------------------
install -d "$LIB_DIR"
install -m 0755 "$REPO/libonnxruntime.so" "$LIB_DIR/libonnxruntime.so"

# --- 4. config -------------------------------------------------------------
# Base the service config on the repo's working config.yaml (its default model
# resolves against the bundled model cache). Point cache_dir at the service
# state dir and disable the dev-only auto-migration. The RPM template is NOT
# used here: its default model (fast-multilingual-e5-small) has no source_repo
# and only resolves if that exact HF cache is preloaded.
install -d "$CFG_DIR"
if [ -f "$CFG_DIR/config.yaml" ]; then
    echo "==> Keeping existing $CFG_DIR/config.yaml"
else
    sed -e "s#cache_dir: \"\./mlcembed\"#cache_dir: \"$MODELS_DIR\"#" \
        -e 's#auto_migrate: true#auto_migrate: false#' \
        "$REPO/config.yaml" > "$CFG_DIR/config.yaml"
fi

# --- 5. state + log dirs ----------------------------------------------------
install -d -o "$USER" -g "$GROUP" -m 0750 "$STATE_DIR" "$MODELS_DIR" "$LOG_DIR"

# --- 6. seed models from the repo cache (works offline immediately) --------
if [ -d "$REPO/mlcembed" ] && [ -n "$(ls -A "$REPO/mlcembed" 2>/dev/null)" ]; then
    echo "==> Seeding models into $MODELS_DIR"
    cp -rn "$REPO/mlcembed/." "$MODELS_DIR/"
    chown -R "$USER:$GROUP" "$MODELS_DIR"
fi

# --- 7. systemd unit --------------------------------------------------------
cat > "$UNIT" <<EOF
[Unit]
Description=MLC LocalEmbed - Fast Local Text Embedding Service
After=network.target

[Service]
Type=simple
User=$USER
Group=$GROUP
WorkingDirectory=$STATE_DIR
Environment=ONNX_PATH=$LIB_DIR/libonnxruntime.so
ExecStart=$BIN_DIR/mlcembedder -config $CFG_DIR/config.yaml
Restart=on-failure
StandardOutput=append:$LOG_DIR/server.log
StandardError=append:$LOG_DIR/server.log

[Install]
WantedBy=multi-user.target
EOF

# --- 8. enable + (re)start --------------------------------------------------
systemctl daemon-reload
systemctl enable localembed.service >/dev/null 2>&1 || true
systemctl restart localembed.service

# --- 9. health check --------------------------------------------------------
echo "==> Waiting for the service to come up..."
ok=0
for _ in $(seq 1 20); do
    if curl -sf http://localhost:9142/api/health >/dev/null 2>&1; then ok=1; break; fi
    sleep 1
done

echo "------------------------------------------------"
if [ "$ok" -eq 1 ]; then
    echo " LocalEmbed is running on http://localhost:9142"
    curl -s http://localhost:9142/api/health || true
    echo
else
    echo " Service did not answer /api/health in time. Recent logs:"
    journalctl -u localembed -n 30 --no-pager || tail -n 30 "$LOG_DIR/server.log"
fi
echo "------------------------------------------------"
echo " Manage:  systemctl status|restart|stop localembed"
echo " CLI:     localembed-cli -text 'Hallo Welt'"
echo "------------------------------------------------"
