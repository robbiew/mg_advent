#!/bin/bash
# Fails while placeholder art (from tools/mkplaceholders) is still embedded,
# so a release can't ship it. Set ALLOW_PLACEHOLDERS=1 for a test build.
cd "$(dirname "$0")/.." || exit 1
files=$(grep -rl --include='*.ANS' ADVENT-PLACEHOLDER internal/embedded/art 2>/dev/null)
[ -z "$files" ] && exit 0
count=$(echo "$files" | wc -l)
if [ "${ALLOW_PLACEHOLDERS:-0}" = "1" ]; then
    echo "WARNING: building with $count placeholder art file(s) (ALLOW_PLACEHOLDERS=1)."
    exit 0
fi
echo "ERROR: $count placeholder art file(s) are still embedded:"
echo "$files" | sed 's/^/  /' | head -40
echo "Replace them with the real art, or set ALLOW_PLACEHOLDERS=1 for a test build."
exit 1
