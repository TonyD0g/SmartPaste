#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

TARGET="arm64-apple-macos14.0"
OUT="build/SmartPaste"
mkdir -p build

RESOURCE_DIR="$(swiftc -print-target-info | python3 -c 'import json,sys; print(json.load(sys.stdin)["paths"]["runtimeResourcePath"])')"

# Some Command Line Tools installs ship both include/swift/module.modulemap and
# include/swift/bridging.modulemap, which both declare the SwiftBridging module.
# Swift then fails with a module redefinition error. Use a writable overlay that
# keeps the real stdlib and hides only the stale duplicate module map.
INCLUDE_DIR="$(cd "$RESOURCE_DIR/../../include" && pwd)"
if [ -f "$INCLUDE_DIR/swift/module.modulemap" ] && [ -f "$INCLUDE_DIR/swift/bridging.modulemap" ]; then
    OVERLAY="$ROOT/build/toolchain-overlay"
    rm -rf "$OVERLAY"
    mkdir -p "$OVERLAY/usr/lib/swift"
    cp -R "$INCLUDE_DIR" "$OVERLAY/usr/include"
    rm -f "$OVERLAY/usr/include/swift/module.modulemap"
    for entry in "$RESOURCE_DIR"/*; do
        ln -s "$entry" "$OVERLAY/usr/lib/swift/$(basename "$entry")"
    done
    RESOURCE_DIR="$OVERLAY/usr/lib/swift"
fi

swiftc -O \
    -target "$TARGET" \
    -module-name SmartPaste \
    -resource-dir "$RESOURCE_DIR" \
    -parse-as-library \
    Sources/SmartPaste/*.swift \
    -framework AppKit \
    -framework ApplicationServices \
    -framework CoreGraphics \
    -framework QuartzCore \
    -framework Security \
    -o "$OUT"

echo "Built $OUT"
