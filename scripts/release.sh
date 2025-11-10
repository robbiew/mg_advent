#!/bin/bash

# GitHub Release Script for Mistigris Advent Calendar
# Creates a GitHub release and uploads all packaged archives
# Note: 2025+ versions use embedded assets (no external art directory needed)
#
# Prerequisites: 
#   - GitHub CLI (gh) installed and authenticated
#   - Archives built using package.sh
#
# Usage: ./release.sh <version> [release-notes]  
# Example: ./release.sh v2.0.0 "2025 modernized release with embedded assets and multi-year support"

set -e

# Configuration
RELEASE_DIR="dist/releases"
REPO="robbiew/mg_advent"

# Check arguments
if [ $# -lt 1 ]; then
    echo "Usage: $0 <version> [release-notes]"
    echo ""
    echo "Examples:"
    echo "  $0 v2.0.0"
    echo "  $0 v2.0.0 '2025 modernized release with embedded assets and multi-year support'"
    echo "  $0 v2.1.0 'Bug fixes and performance improvements'"
    echo ""
    echo "Note: Version should start with 'v' (e.g., v1.0.0)"
    exit 1
fi

VERSION=$1
NOTES=${2:-"Release $VERSION"}

# Validate version format
if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+.*$ ]]; then
    echo "WARNING: Version should follow format v#.#.# (e.g., v1.0.0)"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if gh is installed
if ! command -v gh &> /dev/null; then
    echo "ERROR: GitHub CLI (gh) is not installed."
    echo ""
    echo "Install instructions:"
    echo "  Ubuntu/Debian: sudo apt install gh"
    echo "  macOS: brew install gh"
    echo "  Other: https://cli.github.com/manual/installation"
    echo ""
    echo "After installing, authenticate with: gh auth login"
    exit 1
fi

# Check if authenticated
if ! gh auth status &> /dev/null; then
    echo "ERROR: GitHub CLI is not authenticated."
    echo "Please run: gh auth login"
    exit 1
fi

# Check if release directory exists
if [ ! -d "$RELEASE_DIR" ]; then
    echo "ERROR: Release directory not found: $RELEASE_DIR"
    echo "Please run package.sh first to create release archives."
    exit 1
fi

# Find all archives
ARCHIVES=($(find "$RELEASE_DIR" -type f \( -name "*.zip" -o -name "*.tar.gz" \) 2>/dev/null))

if [ ${#ARCHIVES[@]} -eq 0 ]; then
    echo "ERROR: No archives found in $RELEASE_DIR"
    echo "Please run package.sh first to create release archives."
    exit 1
fi

echo "================================================"
echo "GitHub Release Creator"
echo "================================================"
echo "Repository: $REPO"
echo "Version:    $VERSION"
echo "Notes:      $NOTES"
echo ""
echo "Archives to upload:"
for archive in "${ARCHIVES[@]}"; do
    echo "  - $(basename "$archive")"
done
echo "================================================"
echo ""

# Confirm release
read -p "Create this release? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Release cancelled."
    exit 0
fi

# Check if tag exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo "Tag $VERSION already exists."
    read -p "Delete and recreate tag? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Deleting local tag..."
        git tag -d "$VERSION"
        echo "Deleting remote tag..."
        git push origin ":refs/tags/$VERSION" 2>/dev/null || echo "Remote tag doesn't exist, continuing..."
    else
        echo "Using existing tag."
    fi
else
    # Create and push tag
    echo "Creating tag $VERSION..."
    git tag -a "$VERSION" -m "$NOTES"
    echo "Pushing tag to GitHub..."
    git push origin "$VERSION"
fi

# Check if release exists
if gh release view "$VERSION" --repo "$REPO" &> /dev/null; then
    echo ""
    echo "Release $VERSION already exists."
    read -p "Delete and recreate release? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Deleting existing release..."
        gh release delete "$VERSION" --repo "$REPO" --yes
    else
        echo "Cancelled. Use a different version or delete the existing release first."
        exit 1
    fi
fi

# Create release
echo ""
echo "Creating GitHub release..."
gh release create "$VERSION" \
    --repo "$REPO" \
    --title "$VERSION" \
    --notes "$NOTES" \
    "${ARCHIVES[@]}"

if [ $? -eq 0 ]; then
    echo ""
    echo "================================================"
    echo "✓ Release $VERSION created successfully!"
    echo "================================================"
    echo ""
    echo "View release at:"
    echo "  https://github.com/$REPO/releases/tag/$VERSION"
    echo ""
    echo "Uploaded archives:"
    for archive in "${ARCHIVES[@]}"; do
        echo "  ✓ $(basename "$archive")"
    done
else
    echo ""
    echo "ERROR: Failed to create release."
    exit 1
fi
