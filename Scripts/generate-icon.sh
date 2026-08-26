#!/bin/sh
set -eu

REPOSITORY_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
SOURCE="$REPOSITORY_ROOT/Resources/TunnelDockIconLight-v2.png"
OUTPUT="$REPOSITORY_ROOT/Resources/TunnelDock.icns"
TEMPORARY_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/tunneldock-icon.XXXXXX")
ICONSET="$TEMPORARY_ROOT/TunnelDock.iconset"

cleanup() {
    rm -rf "$TEMPORARY_ROOT"
}
trap cleanup EXIT INT TERM

test -f "$SOURCE"
mkdir -p "$ICONSET"

resize() {
    size=$1
    name=$2
    /usr/bin/sips -z "$size" "$size" "$SOURCE" --out "$ICONSET/$name" >/dev/null
}

resize 16 icon_16x16.png
resize 32 icon_16x16@2x.png
resize 32 icon_32x32.png
resize 64 icon_32x32@2x.png
resize 128 icon_128x128.png
resize 256 icon_128x128@2x.png
resize 256 icon_256x256.png
resize 512 icon_256x256@2x.png
resize 512 icon_512x512.png
resize 1024 icon_512x512@2x.png

# The Command Line Tools-only version of iconutil can decode .icns files but
# rejects otherwise valid iconsets when encoding them. Build the standard ICNS
# container directly so packaging remains independent of Xcode.
entry_size() {
    image_size=$(stat -f %z "$ICONSET/$1")
    echo $((image_size + 8))
}

write_uint32() {
    printf '%08x' "$1" | /usr/bin/xxd -r -p
}

append_entry() {
    type=$1
    image=$2
    image_size=$(stat -f %z "$ICONSET/$image")
    printf '%s' "$type" >> "$OUTPUT"
    write_uint32 $((image_size + 8)) >> "$OUTPUT"
    command cat "$ICONSET/$image" >> "$OUTPUT"
}

total_size=8
total_size=$((total_size + $(entry_size icon_16x16.png)))
total_size=$((total_size + $(entry_size icon_16x16@2x.png)))
total_size=$((total_size + $(entry_size icon_32x32.png)))
total_size=$((total_size + $(entry_size icon_32x32@2x.png)))
total_size=$((total_size + $(entry_size icon_128x128.png)))
total_size=$((total_size + $(entry_size icon_128x128@2x.png)))
total_size=$((total_size + $(entry_size icon_256x256.png)))
total_size=$((total_size + $(entry_size icon_256x256@2x.png)))
total_size=$((total_size + $(entry_size icon_512x512.png)))
total_size=$((total_size + $(entry_size icon_512x512@2x.png)))

printf 'icns' > "$OUTPUT"
write_uint32 "$total_size" >> "$OUTPUT"
append_entry icp4 icon_16x16.png
append_entry ic11 icon_16x16@2x.png
append_entry icp5 icon_32x32.png
append_entry ic12 icon_32x32@2x.png
append_entry ic07 icon_128x128.png
append_entry ic13 icon_128x128@2x.png
append_entry ic08 icon_256x256.png
append_entry ic14 icon_256x256@2x.png
append_entry ic09 icon_512x512.png
append_entry ic10 icon_512x512@2x.png

# Verify that macOS can decode every standard representation we emitted.
VERIFIED_ICONSET="$TEMPORARY_ROOT/Verified.iconset"
/usr/bin/iconutil --convert iconset --output "$VERIFIED_ICONSET" "$OUTPUT"
for image in \
    icon_16x16.png icon_16x16@2x.png \
    icon_32x32.png icon_32x32@2x.png \
    icon_128x128.png icon_128x128@2x.png \
    icon_256x256.png icon_256x256@2x.png \
    icon_512x512.png icon_512x512@2x.png
do
    test -s "$VERIFIED_ICONSET/$image"
done

echo "$OUTPUT"
