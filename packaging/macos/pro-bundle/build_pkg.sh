#!/bin/bash

# LocalEmbed macOS .pkg Build Script
# To be run on a Mac!

set -e

APP_VERSION="1.2.0"
IDENTIFIER="com.hmsoft.localembed"
BUILD_DIR="build/macos-pkg"
PAYLOAD_DIR="$BUILD_DIR/payload"

echo "Creating build structure..."
rm -rf "$BUILD_DIR"
mkdir -p "$PAYLOAD_DIR/usr/local/bin"
mkdir -p "$PAYLOAD_DIR/usr/local/lib"
mkdir -p "$PAYLOAD_DIR/Applications"
mkdir -p "$PAYLOAD_DIR/Library/LaunchAgents"

echo "Building Go Binaries..."
GOWORK=off GOOS=darwin GOARCH=arm64 go build -o "$PAYLOAD_DIR/usr/local/bin/localembed-server" ./cmd/server/main.go
GOWORK=off GOOS=darwin GOARCH=arm64 go build -o "$PAYLOAD_DIR/usr/local/bin/localembed-cli" ./cmd/cli/main.go
GOWORK=off GOOS=darwin GOARCH=arm64 go build -o "$PAYLOAD_DIR/usr/local/bin/localembed-preloader" ./cmd/preloader/main.go

echo "Compiling GUI App..."
# Simple compilation of the Swift file into an executable
swiftc packaging/macos/pro-bundle/gui/StatusBarApp.swift -o "$PAYLOAD_DIR/Applications/LocalEmbed.app/Contents/MacOS/LocalEmbed"
# Note: Real apps need a proper .app folder structure with Info.plist

echo "Adding LaunchAgent..."
cp packaging/macos/com.localembed.server.plist "$PAYLOAD_DIR/Library/LaunchAgents/"

echo "Building Package..."
pkgbuild --identifier "$IDENTIFIER" 
         --version "$APP_VERSION" 
         --root "$PAYLOAD_DIR" 
         --install-location / 
         localembed-macos-unsigned.pkg

echo "------------------------------------------------"
echo " Done! Created localembed-macos-unsigned.pkg"
echo " Next steps on Mac:"
echo " 1. codesign --deep -s "Your Cert" LocalEmbed.app"
echo " 2. productsign --sign "Your Installer Cert" localembed-macos-unsigned.pkg localembed-signed.pkg"
echo "------------------------------------------------"
