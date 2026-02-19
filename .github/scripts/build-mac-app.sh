#!/usr/bin/env bash
# Build promptctl Mac app and create promptctl-macos.dmg. Run from repo root.
set -e
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO_ROOT"
APP_DIR=promptctl-app
DMG_NAME=promptctl-macos.dmg
BUNDLE_NAME=promptctl.app

cd "$APP_DIR"
swift build -c release
cd ..

mkdir -p "$BUNDLE_NAME/Contents/MacOS"
cp "$APP_DIR/.build/release/promptctl-app" "$BUNDLE_NAME/Contents/MacOS/"
cp "$APP_DIR/Info.plist" "$BUNDLE_NAME/Contents/"

# Create DMG in promptctl-app/ so upload path matches workflow
rm -f "$APP_DIR/$DMG_NAME"
hdiutil create -volname "promptctl" -srcfolder "$BUNDLE_NAME" -ov -format UDZO "$APP_DIR/$DMG_NAME"
rm -rf "$BUNDLE_NAME"
echo "Created $APP_DIR/$DMG_NAME"
