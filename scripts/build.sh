#!/bin/bash

# Build script for Mistigris Advent Calendar (Linux/Unix)
# This script builds the application for multiple platforms
# Optionally embeds Windows version information and application manifest
#
# Usage: ./build.sh                    (version from `git describe`)
#        VERSION=v2.0.0 ./build.sh     (explicit version)
# Output: Creates binaries in dist/ directory
#
# Every binary is stamped with the version and build date (`advent -version`).
# Placeholder art blocks the build; ALLOW_PLACEHOLDERS=1 allows a test build.

set -e
cd "$(dirname "$0")/.."

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
BUILD_DATE="$(date -u +%Y-%m-%d)"
LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE}"

echo "Building Mistigris Advent Calendar ${VERSION} (${BUILD_DATE})..."

# Refuse to build release binaries that embed placeholder art
./scripts/check-placeholders.sh

# Default values
OUTPUT_DIR="dist"
PLATFORMS=("linux/amd64" "linux/arm64" "windows/386")

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Build for each platform
for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r -a parts <<< "$platform"
    GOOS="${parts[0]}"
    GOARCH="${parts[1]}"

    output_path="$OUTPUT_DIR/advent-$GOOS-$GOARCH"
    if [ "$GOOS" = "windows" ]; then
        output_path="$OUTPUT_DIR/mg-advent.exe"
    fi

    echo "Building for $GOOS/$GOARCH..."
    
    # Optionally embed Windows resources with the current toolchain
    if [ "$GOOS" = "windows" ]; then
        
        # Compile Windows version information and application manifest
        if command -v x86_64-w64-mingw32-windres &> /dev/null; then
            echo "  Embedding Windows manifest (version information and application metadata)..."
            cd cmd/advent
            if x86_64-w64-mingw32-windres -F pe-i386 resource.rc -O coff -o resource.syso 2>&1; then
                echo "  ✓ Manifest embedded successfully"
            else
                echo "  Warning: Resource compilation failed, building without manifest"
            fi
            cd ../..
        elif command -v windres &> /dev/null; then
            echo "  Embedding Windows manifest (version information and application metadata)..."
            cd cmd/advent
            if windres -F pe-i386 resource.rc -O coff -o resource.syso 2>&1; then
                echo "  ✓ Manifest embedded successfully"
            else
                echo "  Warning: Resource compilation failed, building without manifest"
            fi
            cd ../..
        else
            echo "  Note: windres not found - building without optional Windows metadata"
            echo "        Install mingw-w64 to enable manifest embedding: apt-get install mingw-w64 or brew install mingw-w64"
        fi
        
        GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 go build -trimpath -ldflags="$LDFLAGS" -o "$output_path" ./cmd/advent
        
        # Clean up resource file
        rm -f cmd/advent/resource.syso
    else
        GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 go build -trimpath -ldflags="$LDFLAGS" -o "$output_path" ./cmd/advent
    fi

    # Create checksum
    sha256sum "$output_path" > "$output_path.sha256"
done

echo ""
echo "Build complete! Binaries available in $OUTPUT_DIR/"
ls -lh "$OUTPUT_DIR/"