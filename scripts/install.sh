#!/bin/bash

# LocalEmbed Installer for Linux and macOS
# v1.2.0-dev

set -e

echo "------------------------------------------------"
echo " LocalEmbed Installer"
echo "------------------------------------------------"

OS="$(uname -s)"
ARCH="$(uname -m)"

install_linux() {
    echo "Detected Linux ($ARCH)..."
    if [ -f "/etc/redhat-release" ] || [ -f "/etc/almalinux-release" ]; then
        echo "RPM-based system detected. Building and installing RPM..."
        if command -v task >/dev/null 2>&1; then
            task rpm
            sudo dnf install -y build/rpmbuild/RPMS/x86_64/localembed-*.rpm
            echo "Installation complete via RPM."
            echo "Use 'sudo systemctl start localembed' to start the service."
        else
            echo "Error: 'task' command not found. Please install Taskfile first."
            exit 1
        fi
    else
        echo "Non-RPM Linux detected. Manual installation required."
        # Placeholder for other linux distros if needed
    fi
}

install_macos() {
    echo "Detected macOS ($ARCH)..."
    
    BIN_DIR="/usr/local/bin"
    LIB_DIR="/usr/local/lib"
    VAR_DIR="/usr/local/var/localembed"
    LOG_DIR="/usr/local/var/log"
    PLIST_DIR="$HOME/Library/LaunchAgents"
    
    mkdir -p "$BIN_DIR" "$LIB_DIR" "$VAR_DIR" "$LOG_DIR" "$PLIST_DIR"
    
    echo "Building binaries..."
    GOWORK=off go build -o "$BIN_DIR/mlcembedder" ./cmd/server/main.go
    GOWORK=off go build -o "$BIN_DIR/localembed-cli" ./cmd/cli/main.go
    GOWORK=off go build -o "$BIN_DIR/localembed-preloader" ./cmd/preloader/main.go
    
    echo "Downloading ONNX Runtime..."
    if [ ! -f "$LIB_DIR/libonnxruntime.dylib" ]; then
        # Use existing task logic or simple curl
        task download-onnx
        cp libonnxruntime.dylib "$LIB_DIR/"
    fi
    
    echo "Configuring service..."
    cp packaging/macos/com.localembed.server.plist "$PLIST_DIR/"
    
    # Update paths in plist if necessary (placeholder)
    
    echo "Loading service..."
    launchctl unload "$PLIST_DIR/com.localembed.server.plist" 2>/dev/null || true
    launchctl load "$PLIST_DIR/com.localembed.server.plist"
    
    echo "Installation complete."
    echo "Models will be stored in $VAR_DIR/mlcembed"
    echo "Logs are available at $LOG_DIR/localembed.log"
}

case "$OS" in
    Linux)
        install_linux
        ;;
    Darwin)
        install_macos
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

echo "------------------------------------------------"
echo " Setup finished!"
echo " Next step: Run 'localembed-preloader' to download models."
echo "------------------------------------------------"
