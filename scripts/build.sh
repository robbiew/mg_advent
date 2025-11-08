#!/bin/bash

# Build script for Mistigris Advent Calendar
# This script builds the application for multiple platforms

set -e

echo "Building Mistigris Advent Calendar..."

# Default values
OUTPUT_DIR="dist"
PLATFORMS=("linux/amd64" "linux/arm64" "windows/amd64" "darwin/amd64" "darwin/arm64")

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Build for each platform
for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r -a parts <<< "$platform"
    GOOS="${parts[0]}"
    GOARCH="${parts[1]}"

    binary_name="advent"
    if [ "$GOOS" = "windows" ]; then
        binary_name="advent.exe"
    fi

    output_path="$OUTPUT_DIR/advent-$GOOS-$GOARCH"
    if [ "$GOOS" = "windows" ]; then
        output_path="$OUTPUT_DIR/advent-$GOOS-$GOARCH.exe"
    fi

    echo "Building for $GOOS/$GOARCH..."
    GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags="-s -w" -o "$output_path" ./cmd/advent

    # Create checksum
    sha256sum "$output_path" > "$output_path.sha256"
done

echo "Build complete! Binaries available in $OUTPUT_DIR/"
ls -la "$OUTPUT_DIR/"