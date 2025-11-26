#!/bin/bash

# Build script for Mistigris Advent Calendar (Linux/Unix)
# This script builds the application for multiple platforms
# Automatically embeds Windows manifest to prevent rename delays on Windows 7
#
# Usage: ./build.sh
# Output: Creates binaries in dist/ directory

set -e

echo "Building Mistigris Advent Calendar..."

# Default values
OUTPUT_DIR="dist"
PLATFORMS=("linux/amd64" "linux/arm64" "windows/386")

# Check if go1.20.14 is available for Windows 7 builds
GO120="$HOME/go/bin/go1.20.14"
if [ ! -f "$GO120" ]; then
    echo "Warning: go1.20.14 not found at $GO120"
    echo "Windows builds require Go 1.20 for Windows 7 compatibility"
    echo "Install with: go install golang.org/dl/go1.20.14@latest && go1.20.14 download"
    GO120="go"  # Fallback to default go
fi

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
    
    # Use Go 1.20.14 for Windows builds (Windows 7 compatibility)
    if [ "$GOOS" = "windows" ]; then
        echo "  Using Go 1.20.14 for Windows 7 compatibility..."
        
        # Compile Windows manifest to prevent rename delays on Windows 7
        if command -v x86_64-w64-mingw32-windres &> /dev/null; then
            echo "  Embedding Windows manifest (prevents executable rename delays)..."
            cd cmd/advent
            if x86_64-w64-mingw32-windres -F pe-i386 resource.rc -O coff -o resource.syso 2>&1; then
                echo "  ✓ Manifest embedded successfully"
            else
                echo "  Warning: Resource compilation failed, building without manifest"
                echo "  (This is OK - build will continue but renamed .exe may have startup delay on Windows 7)"
            fi
            cd ../..
        elif command -v windres &> /dev/null; then
            echo "  Embedding Windows manifest (prevents executable rename delays)..."
            cd cmd/advent
            if windres -F pe-i386 resource.rc -O coff -o resource.syso 2>&1; then
                echo "  ✓ Manifest embedded successfully"
            else
                echo "  Warning: Resource compilation failed, building without manifest"
                echo "  (This is OK - build will continue but renamed .exe may have startup delay on Windows 7)"
            fi
            cd ../..
        else
            echo "  Note: windres not found - building without manifest (may have 15-20s delay on Windows 7 when renamed)"
            echo "        Install mingw-w64 to enable manifest embedding: apt-get install mingw-w64 or brew install mingw-w64"
        fi
        
        GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 "$GO120" build -ldflags="-s -w" -o "$output_path" ./cmd/advent
        
        # Clean up resource file
        rm -f cmd/advent/resource.syso
    else
        GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags="-s -w" -o "$output_path" ./cmd/advent
    fi

    # Create checksum
    sha256sum "$output_path" > "$output_path.sha256"
done

echo ""
echo "Build complete! Binaries available in $OUTPUT_DIR/"
ls -lh "$OUTPUT_DIR/"