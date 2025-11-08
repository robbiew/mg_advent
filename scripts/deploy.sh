#!/bin/bash

# Deployment script for Mistigris Advent Calendar
# Prepares the application for production deployment

set -e

echo "Preparing Mistigris Advent Calendar for deployment..."

# Build the application
echo "Building application..."
./scripts/build.sh

# Create deployment package
echo "Creating deployment package..."
DEPLOY_DIR="deploy"
mkdir -p "$DEPLOY_DIR"

# Copy binaries
cp dist/advent-linux-amd64 "$DEPLOY_DIR/advent"
cp dist/*.sha256 "$DEPLOY_DIR/"

# Copy configuration template
cp config/config.yaml "$DEPLOY_DIR/config.yaml.example"

# Copy art directory (without git history)
cp -r art "$DEPLOY_DIR/"

# Copy documentation
cp README.md LICENSE "$DEPLOY_DIR/"

# Copy launch script
cp scripts/launch.sh "$DEPLOY_DIR/"

# Create version file
echo "2.0.0" > "$DEPLOY_DIR/version.txt"
date > "$DEPLOY_DIR/build-date.txt"

# Create archive
echo "Creating deployment archive..."
tar -czf "advent-calendar-2.0.0.tar.gz" "$DEPLOY_DIR/"

echo "Deployment package created: advent-calendar-2.0.0.tar.gz"
echo ""
echo "Deployment contents:"
ls -la "$DEPLOY_DIR/"
echo ""
echo "To deploy:"
echo "1. Extract advent-calendar-2.0.0.tar.gz on target system"
echo "2. Copy config.yaml.example to config.yaml and customize"
echo "3. Ensure art/ directory has appropriate permissions"
echo "4. Test with: ./advent --local"