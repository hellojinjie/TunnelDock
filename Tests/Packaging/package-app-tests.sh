#!/bin/sh
set -eu

REPOSITORY_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
APP="$REPOSITORY_ROOT/.build/release/TunnelDock.app"

test -x "$APP/Contents/MacOS/TunnelDock"
test "$(printf '%s' "$(/usr/bin/lipo -archs "$APP/Contents/MacOS/TunnelDock")" | tr -s ' \n' '\n\n' | sort | tr '\n' ' ')" = "arm64 x86_64 "
/usr/bin/codesign --verify --deep --strict "$APP"
test "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleExecutable' "$APP/Contents/Info.plist")" = "TunnelDock"
test "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$APP/Contents/Info.plist")" = "com.tunneldock.TunnelDock"
test "$(/usr/libexec/PlistBuddy -c 'Print :CFBundlePackageType' "$APP/Contents/Info.plist")" = "APPL"
test "$(/usr/libexec/PlistBuddy -c 'Print :LSMinimumSystemVersion' "$APP/Contents/Info.plist")" = "13.0"
test "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIconFile' "$APP/Contents/Info.plist")" = "TunnelDock"
test -s "$APP/Contents/Resources/TunnelDock.icns"

ICONSET_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/tunneldock-package-icon.XXXXXX")
trap 'rm -rf "$ICONSET_ROOT"' EXIT INT TERM
/usr/bin/iconutil --convert iconset \
    --output "$ICONSET_ROOT/TunnelDock.iconset" \
    "$APP/Contents/Resources/TunnelDock.icns"
for image in \
    icon_16x16.png icon_16x16@2x.png \
    icon_32x32.png icon_32x32@2x.png \
    icon_128x128.png icon_128x128@2x.png \
    icon_256x256.png icon_256x256@2x.png \
    icon_512x512.png icon_512x512@2x.png
do
    test -s "$ICONSET_ROOT/TunnelDock.iconset/$image"
done

if /usr/libexec/PlistBuddy -c 'Print :LSUIElement' "$APP/Contents/Info.plist" >/dev/null 2>&1; then
    echo "LSUIElement must be omitted for a foreground Dock application" >&2
    exit 1
fi
