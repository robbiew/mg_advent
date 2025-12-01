#!/usr/bin/env bash
set -e

# Simple build script for mg_advent DOS (go32v2, 32-bit DOS extender)
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC_DIR="$ROOT_DIR/src"
OUT_DIR="$ROOT_DIR/bin"

mkdir -p "$OUT_DIR"

echo "Building mg_advent (DOS / go32v2) ..."
fpc -Tgo32v2 -O2 -Sm -Sd \
-Fu"$SRC_DIR" \
-Fi"$SRC_DIR" \
-FE"$OUT_DIR" \
"$SRC_DIR/advent.pas"

echo "Done. Output: $OUT_DIR/advent.exe"
