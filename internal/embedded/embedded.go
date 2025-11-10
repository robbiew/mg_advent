package embedded

import (
	"embed"
)

// ArtFS contains all embedded art files
//
//go:embed art/**/*.ANS
var ArtFS embed.FS
