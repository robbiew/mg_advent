#!/bin/bash

# Setup script for local testing of Mistigris Advent Calendar
# Copies all necessary files to ~/retrograde/doors/advent for testing
# This script is for local development testing only

set -e

TEST_DIR="$HOME/retrograde/doors/advent"
echo "Setting up local test environment in: $TEST_DIR"

# Clean up any existing test directory
if [ -d "$TEST_DIR" ]; then
    echo "Removing existing test directory..."
    rm -rf "$TEST_DIR"
fi

# Create test directory
mkdir -p "$TEST_DIR"

# Copy the built binary
if [ -f "./advent" ]; then
    echo "Copying application binary..."
    cp ./advent "$TEST_DIR/"
else
    echo "Building application first..."
    go build -o advent ./cmd/advent
    cp ./advent "$TEST_DIR/"
fi

# Copy configuration
echo "Copying configuration..."
cp -r config "$TEST_DIR/"

# Copy art directory
echo "Copying art assets..."
cp -r art "$TEST_DIR/"

# Copy documentation
echo "Copying documentation..."
cp README.md LICENSE "$TEST_DIR/"

# Copy BBS info files
echo "Copying BBS info files..."
cp FILE_ID.ANS INFOFILE.ANS MEMBERS.ANS "$TEST_DIR/"

# Create a sample door32.sys for local testing
echo "Creating sample door32.sys for local testing..."
cat > "$TEST_DIR/door32.sys" << 'EOF'
Sysop Name
Test BBS
Test Location
555-123-4567
0
120
1
SysOp
120
1
1
EOF

# Create a sample config for local testing
echo "Creating sample config.yaml for local testing..."
cat > "$TEST_DIR/config.yaml" << 'EOF'
app:
  name: "Mistigris Advent Calendar - Local Test"
  version: "2.0.0"
  timeout_idle: "5m"
  timeout_max: "120m"

display:
  mode: "utf8"
  theme: "classic"
  scrolling:
    enabled: true
    indicators: true
    keyboard_shortcuts: true
  columns:
    handle_80_column_issue: true
    auto_detect_width: true
  performance:
    cache_enabled: true
    cache_size_mb: 50
    preload_lines: 100

logging:
  level: "info"
  format: "text"

art:
  base_dir: "art"

bbs:
  dropfile_path: "door32.sys"
EOF

# Make binary executable
chmod +x "$TEST_DIR/advent"

echo ""
echo "✅ Local test environment setup complete!"
echo ""
echo "Test Directory: $TEST_DIR"
echo "Contents:"
ls -la "$TEST_DIR"
echo ""
echo "To test locally:"
echo "  cd $TEST_DIR"
echo "  ./advent --local"
echo ""
echo "To test with BBS simulation:"
echo "  cd $TEST_DIR"
echo "  ./advent --path door32.sys"
echo ""
echo "Note: This directory is gitignored and only for your local testing."