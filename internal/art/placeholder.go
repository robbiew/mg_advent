package art

import "bytes"

// PlaceholderMarker is written into every placeholder art file (in its SAUCE
// author field and on screen) by tools/mkplaceholders. Real art never
// contains it, so its presence means a file still needs replacing.
const PlaceholderMarker = "ADVENT-PLACEHOLDER"

// IsPlaceholder reports whether art file contents are generated placeholder art.
func IsPlaceholder(data []byte) bool {
	return bytes.Contains(data, []byte(PlaceholderMarker))
}
