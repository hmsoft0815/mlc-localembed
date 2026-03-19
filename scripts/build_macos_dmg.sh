#!/bin/bash
set -e

APP_NAME="LocalEmbed"
VERSION="1.2.0"
STAGING_DIR="build/macos_staging"
APP_BUNDLE="$STAGING_DIR/$APP_NAME.app"
CONTENTS="$APP_BUNDLE/Contents"
MACOS_DIR="$CONTENTS/MacOS"
RESOURCES_DIR="$CONTENTS/Resources"

echo "🚀 Building macOS App Bundle..."

# 1. Cleanup & Preparation
rm -rf "$STAGING_DIR"
mkdir -p "$MACOS_DIR"
mkdir -p "$RESOURCES_DIR"

# 2. Build Go Binaries
echo "📦 Building Go binaries..."
GOWORK=off go build -o "$MACOS_DIR/mlcembedder" ./cmd/server/main.go
GOWORK=off go build -o "$MACOS_DIR/localembed-cli" ./cmd/cli/main.go
GOWORK=off go build -o "$MACOS_DIR/localembed-preloader" ./cmd/preloader/main.go

# 3. Build Swift GUI
echo "🎨 Building Swift GUI..."
swiftc packaging/macos/pro-bundle/gui/StatusBarApp.swift -o "$MACOS_DIR/LocalEmbedManager" -target arm64-apple-macos11.0

# 4. Copy Assets & Config
echo "📄 Copying assets..."
cp packaging/macos/pro-bundle/Info.plist "$CONTENTS/Info.plist"
cp config.yaml "$RESOURCES_DIR/config.yaml"
# Copy ONNX library if present (check multiple names/locations)
for lib in "libonnxruntime.dylib" "libonnxruntime.1.17.0.dylib"; do
    if [ -f "$lib" ]; then
        echo "Found ONNX library: $lib"
        cp "$lib" "$MACOS_DIR/libonnxruntime.dylib"
        break
    fi
done

# 5. Create DMG
echo "💿 Creating DMG..."
rm -f "$APP_NAME-$VERSION.dmg"
hdiutil create -volname "$APP_NAME" -srcfolder "$STAGING_DIR" -ov -format UDZO "$APP_NAME-$VERSION.dmg"

echo "✅ DMG build complete: $APP_NAME-$VERSION.dmg"
