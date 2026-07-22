# tihole — task runner
# Run `just` or `just --list` to see recipes.

bin := "tihole"
pkg := "./cmd/tihole"
out := "bin/tihole"

# Show available recipes
default:
    @just --list

# Build the binary into ./bin
build:
    go build -o {{ out }} {{ pkg }}

# Run the TUI
run *args:
    go run {{ pkg }} {{ args }}

# Install to $GOBIN / $GOPATH/bin
install:
    go install {{ pkg }}

# Run all tests
test *args:
    go test ./... {{ args }}

# Run tests with coverage summary (house standard: 80%+)
cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | tail -1

# Open the HTML coverage report
cover-html: cover
    go tool cover -html=coverage.out

# Format all Go source (gofmt + golines at 80 cols)
fmt:
    gofmt -w .
    PATH="$(go env GOPATH)/bin:$PATH" golines \
        --base-formatter=gofmt --max-len=80 \
        --shorten-comments -w . \
        || go run github.com/segmentio/golines@latest \
        --base-formatter=gofmt --max-len=80 \
        --shorten-comments -w .

# Verify formatting (fails if anything is unformatted)
fmt-check:
    #!/usr/bin/env bash
    set -euo pipefail
    bad="$(gofmt -l .)"
    if [[ -n "$bad" ]]; then
        echo "$bad"
        exit 1
    fi
    export PATH="$(go env GOPATH)/bin:$PATH"
    if ! command -v golines >/dev/null; then
        go install github.com/segmentio/golines@latest
    fi
    bad="$(golines --base-formatter=gofmt --max-len=80 \
        --shorten-comments -l .)"
    if [[ -n "$bad" ]]; then
        echo "$bad"
        exit 1
    fi

# go vet
vet:
    go vet ./...

# Full lint gate: formatting + vet
lint: fmt-check vet

# Tidy modules
tidy:
    go mod tidy

# Everything CI should enforce
check: lint test

# Remove build + coverage artifacts
clean:
    rm -rf bin coverage.out

# ─── Release ─────────────────────────────────────────────────────

# Regenerate site/src/changelog.js from repo-root CHANGELOG.md
changelog-sync:
    go run {{ pkg }} changelog-sync

# Bump VERSION, promote CHANGELOG Unreleased, sync site, commit + tag
# Usage: just release patch | minor | major
release LEVEL:
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ ! "{{ LEVEL }}" =~ ^(patch|minor|major)$ ]]; then
        echo "Usage: just release patch|minor|major"; exit 1
    fi
    if [[ -n "$(git status --porcelain)" ]]; then
        echo "Error: dirty working tree"; exit 1
    fi
    BRANCH=$(git rev-parse --abbrev-ref HEAD)
    if [[ "$BRANCH" != "main" ]]; then
        echo "Error: release must run from main (on $BRANCH)"; exit 1
    fi
    # Release quality gate.
    go build ./...
    go vet ./...
    go test ./...
    # Compute the next version from VERSION.
    OLD=$(tr -d '[:space:]' < VERSION)
    IFS='.' read -r MAJ MIN PAT <<< "$OLD"
    case "{{ LEVEL }}" in
        patch) PAT=$((PAT + 1)) ;;
        minor) MIN=$((MIN + 1)); PAT=0 ;;
        major) MAJ=$((MAJ + 1)); MIN=0; PAT=0 ;;
    esac
    NEW="${MAJ}.${MIN}.${PAT}"
    TODAY=$(date -u +%Y-%m-%d)
    echo "Releasing v${OLD} -> v${NEW}"
    printf '%s\n' "$NEW" > VERSION
    # Promote [Unreleased] into a dated version section, leaving a fresh
    # empty Unreleased at the top.
    awk -v ver="$NEW" -v today="$TODAY" '
        /^## \[Unreleased\]/ && !done {
            print "## [Unreleased]"
            print ""
            print "## [" ver "] - " today
            done = 1
            next
        }
        { print }
    ' CHANGELOG.md > CHANGELOG.md.tmp
    mv CHANGELOG.md.tmp CHANGELOG.md
    # Refresh the link-reference footer.
    REPO="https://github.com/z19r/tihole"
    if grep -q '^\[Unreleased\]:' CHANGELOG.md; then
        sed -i "s#^\[Unreleased\]:.*#[Unreleased]: ${REPO}/compare/v${NEW}...HEAD#" \
            CHANGELOG.md
        printf '[%s]: %s/releases/tag/v%s\n' "$NEW" "$REPO" "$NEW" \
            >> CHANGELOG.md
    fi
    # Regenerate the site changelog from the promoted CHANGELOG.md.
    go run {{ pkg }} changelog-sync
    git add VERSION CHANGELOG.md site/src/changelog.js
    git commit -m "release: v${NEW}"
    git tag "v${NEW}"
    echo ""
    echo "Committed and tagged v${NEW}."
    echo "Push with: git push origin main && git push origin v${NEW}"

# Cut a prerelease (beta) of the CURRENT VERSION: vX.Y.Z-beta.N.
# The beta number auto-increments from existing tags. VERSION is left
# untouched — betas lead up to the same X.Y.Z final. Usage: just prerelease
prerelease:
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ -n "$(git status --porcelain)" ]]; then
        echo "Error: dirty working tree"; exit 1
    fi
    BRANCH=$(git rev-parse --abbrev-ref HEAD)
    if [[ "$BRANCH" != "main" ]]; then
        echo "Error: prerelease must run from main (on $BRANCH)"; exit 1
    fi
    # Release quality gate.
    go build ./...
    go vet ./...
    go test ./...
    # Beta of the current base version; next N from existing beta tags.
    BASE=$(tr -d '[:space:]' < VERSION)
    LAST=$(git tag -l "v${BASE}-beta.*" \
        | sed "s/^v${BASE}-beta\.//" | sort -n | tail -1)
    if [[ -z "${LAST:-}" ]]; then N=1; else N=$((LAST + 1)); fi
    PRE="${BASE}-beta.${N}"
    if git rev-parse -q --verify "refs/tags/v${PRE}" >/dev/null; then
        echo "Error: tag v${PRE} already exists"; exit 1
    fi
    TODAY=$(date -u +%Y-%m-%d)
    echo "Cutting prerelease v${PRE}"
    # Promote [Unreleased] into a dated beta section, leaving a fresh
    # empty Unreleased at the top. VERSION is intentionally not changed.
    awk -v ver="$PRE" -v today="$TODAY" '
        /^## \[Unreleased\]/ && !done {
            print "## [Unreleased]"
            print ""
            print "## [" ver "] - " today
            done = 1
            next
        }
        { print }
    ' CHANGELOG.md > CHANGELOG.md.tmp
    mv CHANGELOG.md.tmp CHANGELOG.md
    # Refresh the link-reference footer.
    REPO="https://github.com/z19r/tihole"
    if grep -q '^\[Unreleased\]:' CHANGELOG.md; then
        sed -i "s#^\[Unreleased\]:.*#[Unreleased]: ${REPO}/compare/v${PRE}...HEAD#" \
            CHANGELOG.md
        printf '[%s]: %s/releases/tag/v%s\n' "$PRE" "$REPO" "$PRE" \
            >> CHANGELOG.md
    fi
    # Regenerate the site changelog from the promoted CHANGELOG.md.
    go run {{ pkg }} changelog-sync
    git add CHANGELOG.md site/src/changelog.js
    git commit -m "release: v${PRE}"
    git tag "v${PRE}"
    echo ""
    echo "Committed and tagged v${PRE} (VERSION stays ${BASE})."
    echo "Push with: git push origin main && git push origin v${PRE}"
