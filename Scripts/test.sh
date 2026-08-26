#!/bin/sh
set -eu

TUNNELDOCK_PROJECT_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
TUNNELDOCK_CLT_SDK=/Library/Developer/CommandLineTools/SDKs/MacOSX15.4.sdk

if [ -d "$TUNNELDOCK_CLT_SDK" ]; then
    TUNNELDOCK_SDK_ROOT=$TUNNELDOCK_CLT_SDK
else
    TUNNELDOCK_SDK_ROOT=$(xcrun --sdk macosx --show-sdk-path)
fi

TUNNELDOCK_FILTER=${2-}
cd "$TUNNELDOCK_PROJECT_ROOT"

SDKROOT="$TUNNELDOCK_SDK_ROOT" \
CLANG_MODULE_CACHE_PATH="$TUNNELDOCK_PROJECT_ROOT/.build/clang-module-cache" \
swift run --scratch-path "$TUNNELDOCK_PROJECT_ROOT/.build" TunnelDockCoreTests "$TUNNELDOCK_FILTER"

SDKROOT="$TUNNELDOCK_SDK_ROOT" \
CLANG_MODULE_CACHE_PATH="$TUNNELDOCK_PROJECT_ROOT/.build/clang-module-cache" \
swift run --scratch-path "$TUNNELDOCK_PROJECT_ROOT/.build" TunnelDockAppTests "$TUNNELDOCK_FILTER"
