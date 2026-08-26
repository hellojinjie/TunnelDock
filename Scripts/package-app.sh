#!/bin/sh
set -eu

REPOSITORY_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
SDK_PATH=/Library/Developer/CommandLineTools/SDKs/MacOSX15.4.sdk
if [ ! -d "$SDK_PATH" ]; then
    SDK_PATH=$(xcrun --sdk macosx --show-sdk-path)
fi

export SDKROOT="$SDK_PATH"
export CLANG_MODULE_CACHE_PATH="$REPOSITORY_ROOT/.build/clang-module-cache"

sh "$REPOSITORY_ROOT/Scripts/generate-icon.sh"

swift build \
    --package-path "$REPOSITORY_ROOT" \
    --scratch-path "$REPOSITORY_ROOT/.build" \
    -c release \
    --product TunnelDock

CLANG_MODULE_CACHE_PATH="$REPOSITORY_ROOT/.build/clang-module-cache-x86_64" \
swift build \
    --package-path "$REPOSITORY_ROOT" \
    --scratch-path "$REPOSITORY_ROOT/.build/x86_64" \
    --triple x86_64-apple-macosx13.0 \
    -c release \
    --product TunnelDock

APP="$REPOSITORY_ROOT/.build/release/TunnelDock.app"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
/usr/bin/lipo -create \
    "$REPOSITORY_ROOT/.build/release/TunnelDock" \
    "$REPOSITORY_ROOT/.build/x86_64/release/TunnelDock" \
    -output "$APP/Contents/MacOS/TunnelDock"
cp "$REPOSITORY_ROOT/Resources/Info.plist" "$APP/Contents/Info.plist"
cp "$REPOSITORY_ROOT/Resources/TunnelDock.icns" "$APP/Contents/Resources/TunnelDock.icns"
chmod 755 "$APP/Contents/MacOS/TunnelDock"
/usr/bin/codesign --force --sign - "$APP"

echo "$APP"
