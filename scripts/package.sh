#!/bin/bash

# Package script for Mistigris Advent Calendar GitHub releases
# Creates release archives for Windows 32-bit, Linux x86_64, and Linux ARM64
# Note: Only includes the binary executable and essential ANS files
#
# Usage: ./package.sh
# Output: Creates .zip and .tar.gz archives in dist/releases/

set -e

echo "Packaging Mistigris Advent Calendar for release..."

# Configuration
OUTPUT_DIR="dist"
RELEASE_DIR="dist/releases"

# Platform configurations: "platform-name:binary-name:archive-format"
PLATFORMS=(
    "windows-386:mg-advent.exe:zip"
    "linux-amd64:advent-linux-amd64:tar.gz"
    "linux-arm64:advent-linux-arm64:tar.gz"
)

# Files to include in all packages
COMMON_FILES=(
    "internal/embedded/FILE_ID.ANS"
    "internal/embedded/FILE_ID.DIZ"
    "internal/embedded/INFOFILE.ANS"
    "internal/embedded/LICENSE.TXT"
    "internal/embedded/MEMBERS.ANS"
    "README.TXT"
)

# Create release directory
mkdir -p "$RELEASE_DIR"

# Function to create package
create_package() {
    local platform_name=$1
    local binary_name=$2
    local archive_format=$3
    local package_name="advent-${platform_name}"
    local temp_dir="${RELEASE_DIR}/${package_name}"
    
    echo "Creating package for ${platform_name}..."
    
    # Create temporary directory structure
    mkdir -p "$temp_dir"
    
    # Copy binary
    if [ ! -f "${OUTPUT_DIR}/${binary_name}" ]; then
        echo "ERROR: Binary ${OUTPUT_DIR}/${binary_name} not found!"
        echo "Please run build script first."
        exit 1
    fi
    cp "${OUTPUT_DIR}/${binary_name}" "$temp_dir/"
    
    # Platform-specific launcher scripts are no longer included
    
    # Copy essential ANS files
    for file in "${COMMON_FILES[@]}"; do
        if [ ! -f "$file" ]; then
            echo "WARNING: $file not found, skipping..."
            continue
        fi
        cp "$file" "$temp_dir/"
    done
    
    # Note: Art assets are now embedded in the binary, no external art directory needed
    
    # Create archive based on format
    cd "$RELEASE_DIR"
    if [ "$archive_format" = "zip" ]; then
        # Check if zip is available
        if command -v zip &> /dev/null; then
            zip -r "${package_name}.zip" "${package_name}" > /dev/null
            echo "  ✓ Created ${package_name}.zip"
        else
            echo "  WARNING: zip command not found, skipping ZIP creation"
        fi
    else
        tar -czf "${package_name}.tar.gz" "${package_name}"
        echo "  ✓ Created ${package_name}.tar.gz"
    fi
    cd - > /dev/null
    
    # Clean up temporary directory
    rm -rf "$temp_dir"
}

# Check if binaries exist
echo "Checking for binaries..."
missing_binaries=0
for platform_config in "${PLATFORMS[@]}"; do
    IFS=':' read -r -a parts <<< "$platform_config"
    binary_name="${parts[1]}"
    if [ ! -f "${OUTPUT_DIR}/${binary_name}" ]; then
        echo "  ✗ Missing: ${binary_name}"
        missing_binaries=1
    else
        echo "  ✓ Found: ${binary_name}"
    fi
done

if [ $missing_binaries -eq 1 ]; then
    echo ""
    echo "ERROR: Some binaries are missing. Please run the build script first:"
    echo "  ./scripts/build.sh"
    exit 1
fi

echo ""

# Create packages for each platform
for platform_config in "${PLATFORMS[@]}"; do
    IFS=':' read -r -a parts <<< "$platform_config"
    platform_name="${parts[0]}"
    binary_name="${parts[1]}"
    archive_format="${parts[2]}"
    
    create_package "$platform_name" "$binary_name" "$archive_format"
done

echo ""
echo "Packaging complete! Release archives available in ${RELEASE_DIR}/"
ls -lh "${RELEASE_DIR}/"*.{zip,tar.gz} 2>/dev/null || echo "No archives created"

echo ""
echo "To create a GitHub release:"
echo "  1. Use the release script: ./scripts/release.sh v2.0.0 'Release notes'"
echo "  2. Or manually:"
echo "     - Create a new tag: git tag -a v2.0.0 -m 'Release v2.0.0'"
echo "     - Push the tag: git push origin v2.0.0"
echo "     - Upload files from ${RELEASE_DIR}/ to the GitHub release page"
