package embedded

import (
	"embed"
)

// ArtFS contains all embedded art files. Each art/<year> directory is one
// season; art/common holds fallback screens. Note that go:embed skips files
// whose names start with '.' or '_', so a directory holding only a .gitkeep
// is not embedded.
//
//go:embed art
var ArtFS embed.FS
