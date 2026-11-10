#!/bin/bash
# Publish the archives from build.sh + package.sh as a GitHub release.
#
# Usage: ./scripts/release.sh <version> [options]
#   --notes "text"      release notes (default: "Release <version>")
#   --notes-file FILE   release notes from a Markdown file
#   --title "text"      release title (default: the version)
#   --prerelease        mark as a pre-release (v1.0.0-style "Latest" is untouched)
#   --yes               don't ask for confirmation, e.g. in a script
#   --dry-run           check everything and show the commands; change nothing
#
# Examples:
#   VERSION=v2.0.0 ./scripts/build.sh && ./scripts/package.sh
#   ./scripts/release.sh v2.0.0 --notes-file notes.md
#   ./scripts/release.sh v2.0.0-beta.2 --prerelease --notes-file notes.md
#
# The tag goes on the current commit, which must be committed and pushed.
# With --yes an existing tag or release is never replaced: the script stops.

set -euo pipefail
cd "$(dirname "$0")/.."

RELEASE_DIR="dist/releases"
REPO="robbiew/mg_advent"

usage() {
    sed -n '2,19p' "$0" | sed 's/^# \{0,1\}//'
    exit "${1:-1}"
}
fail() {
    echo "ERROR: $*" >&2
    exit 1
}

VERSION="" NOTES="" NOTES_FILE="" TITLE=""
PRERELEASE=0 YES=0 DRY_RUN=0
while [ $# -gt 0 ]; do
    case "$1" in
        --notes) NOTES="${2:?--notes needs text}"; shift 2 ;;
        --notes-file) NOTES_FILE="${2:?--notes-file needs a file}"; shift 2 ;;
        --title) TITLE="${2:?--title needs text}"; shift 2 ;;
        --prerelease) PRERELEASE=1; shift ;;
        --yes | -y) YES=1; shift ;;
        --dry-run) DRY_RUN=1; shift ;;
        -h | --help) usage 0 ;;
        -*) echo "Unknown option: $1" >&2; usage ;;
        *)
            [ -z "$VERSION" ] || { echo "Unexpected argument: $1" >&2; usage; }
            VERSION="$1"; shift ;;
    esac
done
[ -n "$VERSION" ] || usage
TITLE="${TITLE:-$VERSION}"
NOTES="${NOTES:-Release $VERSION}"
[ -z "$NOTES_FILE" ] || [ -f "$NOTES_FILE" ] || fail "notes file not found: $NOTES_FILE"

# confirm "question": yes with --yes, otherwise ask.
confirm() {
    [ "$YES" = 1 ] && return 0
    local reply
    read -r -p "$1 (y/N) " reply
    [[ $reply =~ ^[Yy]$ ]]
}
# run cmd...: run it, or only print it with --dry-run.
run() {
    if [ "$DRY_RUN" = 1 ]; then
        printf '  would run:'
        printf ' %q' "$@"
        echo
    else
        "$@"
    fi
}

# --- Checks (all run, even with --dry-run) ---------------------------------

[[ $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]] ||
    fail "version must look like v1.2.3 or v1.2.3-beta.1 (got $VERSION)"
if [[ $VERSION == *-* && $PRERELEASE = 0 ]]; then
    echo "WARNING: $VERSION has a pre-release suffix, but --prerelease was not given."
    [ "$YES" = 0 ] || fail "pass --prerelease, or use a version without a suffix"
    confirm "Publish it as a normal (Latest) release anyway?" || exit 1
fi

./scripts/check-placeholders.sh # set ALLOW_PLACEHOLDERS=1 for a test release

command -v gh >/dev/null || fail "GitHub CLI (gh) is not installed: https://cli.github.com"
gh auth status >/dev/null 2>&1 || fail "gh is not signed in: run gh auth login"

[ -z "$(git status --porcelain)" ] || fail "the working tree has uncommitted changes"
COMMIT="$(git rev-parse HEAD)"
git fetch -q origin
[ -n "$(git branch -r --contains "$COMMIT")" ] || fail "commit ${COMMIT:0:7} is not on GitHub yet: push it first"

shopt -s nullglob
ASSETS=("$RELEASE_DIR"/*.zip "$RELEASE_DIR"/*.tar.gz)
[ ${#ASSETS[@]} -gt 0 ] || fail "no archives in $RELEASE_DIR: run scripts/build.sh and scripts/package.sh"
[ ! -f "$RELEASE_DIR/checksums.sha256" ] || ASSETS+=("$RELEASE_DIR/checksums.sha256")

# The binaries must carry this version (build.sh stamps VERSION).
if [ "$(uname -sm)" = "Linux x86_64" ] && [ -x dist/advent-linux-amd64 ]; then
    built="$(dist/advent-linux-amd64 -version | head -1)"
    [[ $built == *" $VERSION" ]] ||
        fail "binaries report \"$built\"; rebuild with: VERSION=$VERSION ./scripts/build.sh && ./scripts/package.sh"
else
    echo "Note: can't run the Linux x86-64 binary here to check its version."
fi

# Tags and releases. Destructive steps are only planned here and carried out
# after the final confirmation, so cancelling there changes nothing.
DELETE_LOCAL_TAG=0 DELETE_REMOTE_TAG=0 DELETE_RELEASE=0 CREATE_TAG=1 PUSH_TAG=1

# The commit a tag on GitHub marks (peeled, so annotated tags give their commit).
remote_tag_commit() {
    local refs
    refs="$(git ls-remote --tags origin "refs/tags/$VERSION" "refs/tags/$VERSION^{}")"
    awk -v peeled="refs/tags/$VERSION^{}" '$2 == peeled { print $1; found = 1 } END { if (!found) exit 1 }' <<<"$refs" ||
        awk '{ print $1 }' <<<"$refs"
}
REMOTE_TAG="$(remote_tag_commit)"
LOCAL_TAG="$(git rev-parse -q --verify "refs/tags/$VERSION^{commit}" || true)"

if [ -n "$REMOTE_TAG" ]; then
    if [ "$REMOTE_TAG" = "$COMMIT" ]; then
        echo "Tag $VERSION on GitHub already marks ${COMMIT:0:7}; reusing it."
        CREATE_TAG=0 PUSH_TAG=0
        [ -z "$LOCAL_TAG" ] || [ "$LOCAL_TAG" = "$COMMIT" ] || DELETE_LOCAL_TAG=1
    else
        [ "$YES" = 0 ] || fail "tag $VERSION on GitHub marks ${REMOTE_TAG:0:7}, not ${COMMIT:0:7}"
        confirm "Tag $VERSION on GitHub marks ${REMOTE_TAG:0:7}. Replace it with ${COMMIT:0:7}?" || exit 1
        DELETE_REMOTE_TAG=1
        [ -z "$LOCAL_TAG" ] || DELETE_LOCAL_TAG=1
    fi
elif [ -n "$LOCAL_TAG" ]; then
    if [ "$LOCAL_TAG" = "$COMMIT" ]; then
        echo "Local tag $VERSION marks ${COMMIT:0:7}; it will be pushed."
        CREATE_TAG=0
    else
        [ "$YES" = 0 ] || fail "local tag $VERSION marks ${LOCAL_TAG:0:7}, not ${COMMIT:0:7}"
        confirm "Local tag $VERSION marks ${LOCAL_TAG:0:7}. Replace it with ${COMMIT:0:7}?" || exit 1
        DELETE_LOCAL_TAG=1
    fi
fi

if gh release view "$VERSION" --repo "$REPO" >/dev/null 2>&1; then
    [ "$YES" = 0 ] || fail "release $VERSION already exists"
    confirm "Release $VERSION already exists. Replace it?" || exit 1
    DELETE_RELEASE=1
fi

# --- Publish ---------------------------------------------------------------

kind="release"
[ "$PRERELEASE" = 0 ] || kind="PRE-RELEASE"
echo "================================================"
echo "Repository: $REPO"
echo "Version:    $VERSION ($kind) on ${COMMIT:0:7}"
echo "Title:      $TITLE"
echo "Notes:      ${NOTES_FILE:-$NOTES}"
echo "Files:"
for a in "${ASSETS[@]}"; do echo "  - $(basename "$a")"; done
[ "$DELETE_RELEASE" = 0 ] || echo "Replaces:   the existing release $VERSION (deleted first)"
[ "$DELETE_REMOTE_TAG" = 0 ] || echo "Replaces:   tag $VERSION on GitHub at ${REMOTE_TAG:0:7}"
echo "================================================"
[ "$DRY_RUN" = 1 ] || confirm "Create this $kind?" || { echo "Cancelled; nothing was changed."; exit 0; }

[ "$DELETE_RELEASE" = 0 ] || run gh release delete "$VERSION" --repo "$REPO" --yes
[ "$DELETE_REMOTE_TAG" = 0 ] || run git push origin ":refs/tags/$VERSION"
[ "$DELETE_LOCAL_TAG" = 0 ] || run git tag -d "$VERSION"
[ "$CREATE_TAG" = 0 ] || run git tag -a "$VERSION" -m "$TITLE" "$COMMIT"
[ "$PUSH_TAG" = 0 ] || run git push origin "refs/tags/$VERSION"

args=(release create "$VERSION" --repo "$REPO" --title "$TITLE" --verify-tag)
[ "$PRERELEASE" = 0 ] || args+=(--prerelease)
if [ -n "$NOTES_FILE" ]; then args+=(--notes-file "$NOTES_FILE"); else args+=(--notes "$NOTES"); fi
run gh "${args[@]}" "${ASSETS[@]}"

if [ "$DRY_RUN" = 1 ]; then
    echo "Dry run: nothing was changed."
else
    echo "Published: https://github.com/$REPO/releases/tag/$VERSION"
fi
