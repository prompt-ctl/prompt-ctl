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
mkdir -p "$BUNDLE_NAME/Contents/Resources"
cp "$APP_DIR/.build/release/promptctl-app" "$BUNDLE_NAME/Contents/MacOS/"
cp "$APP_DIR/Info.plist" "$BUNDLE_NAME/Contents/"
if [[ -f "$APP_DIR/AppIcon.icns" ]]; then
  cp "$APP_DIR/AppIcon.icns" "$BUNDLE_NAME/Contents/Resources/"
elif [[ -f "$APP_DIR/Resources/AppIcon.icns" ]]; then
  cp "$APP_DIR/Resources/AppIcon.icns" "$BUNDLE_NAME/Contents/Resources/"
fi

# Ad-hoc sign so Gatekeeper doesn't report app as "damaged" when opened from DMG
codesign -s - --force --deep "$BUNDLE_NAME"

# Create DMG in promptctl-app/ so upload path matches workflow
rm -f "$APP_DIR/$DMG_NAME"
hdiutil create -volname "promptctl" -srcfolder "$BUNDLE_NAME" -ov -format UDZO "$APP_DIR/$DMG_NAME"
rm -rf "$BUNDLE_NAME"
echo "Created $APP_DIR/$DMG_NAME"
