package embedded

import (
	"embed"
)

// ArtFS contains all embedded art files
// Includes .gitkeep files to ensure empty year directories are embedded
//
//go:embed art
var ArtFS embed.FS
