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

# Delete local branches already merged into origin/main. Uses a content
# check (cherry-pick equivalence) so squash- and merge-commit-merged PRs
# are caught — plain `git branch --merged` misses those, which is how
# stale branches pile up. Pass `just prune-branches dry` to preview only.
prune-branches MODE="":
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ "{{ MODE }}" != "" && "{{ MODE }}" != "dry" ]]; then
        echo "Usage: just prune-branches [dry]"; exit 1
    fi
    git fetch --prune origin >/dev/null 2>&1 || true
    # Resolve the repo's default branch instead of hardcoding "main".
    base=$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD \
        2>/dev/null || echo origin/main)
    if ! git rev-parse --verify "$base" >/dev/null 2>&1; then
        echo "Error: base ref '$base' does not resolve" >&2; exit 1
    fi
    base_local="${base#origin/}"
    current=$(git rev-parse --abbrev-ref HEAD)
    merged=()
    while IFS= read -r b; do
        [[ "$b" == "$base_local" ]] && continue
        [[ "$b" == "$current" ]] && continue
        # Empty output = every commit on $b is already in $base.
        # A nonzero exit means the comparison itself failed (e.g. bad
        # ref) — that must NOT be treated as "merged".
        out=$(git rev-list --cherry-pick --right-only \
            --no-merges "$base...$b") && status=0 || status=$?
        if [[ $status -ne 0 ]]; then
            echo "Error: failed to diff '$b' against '$base'" >&2
            exit 1
        fi
        if [[ -z "$out" ]]; then
            merged+=("$b")
        fi
    done < <(git for-each-ref --format='%(refname:short)' refs/heads/)
    if [[ ${#merged[@]} -eq 0 ]]; then
        echo "No merged branches to prune."; exit 0
    fi
    if [[ "{{ MODE }}" == "dry" ]]; then
        echo "Would delete:"; printf '  %s\n' "${merged[@]}"; exit 0
    fi
    deleted=()
    failed=()
    for b in "${merged[@]}"; do
        if git branch -D "$b"; then
            deleted+=("$b")
        else
            failed+=("$b")
        fi
    done
    echo ""
    echo "Deleted ${#deleted[@]} branch(es)."
    if [[ ${#failed[@]} -gt 0 ]]; then
        echo "Failed to delete:"; printf '  %s\n' "${failed[@]}"
        exit 1
    fi

# ─── Release ─────────────────────────────────────────────────────

# Regenerate site/src/changelog.js from repo-root CHANGELOG.md
changelog-sync:
    go run {{ pkg }} changelog-sync

# Release verification gate (same checks CI enforces)
release-check: lint
    go build ./...
    go test ./...

# Preview what a release would do without changing anything
release-dry-run LEVEL:
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ ! "{{ LEVEL }}" =~ ^(patch|minor|major)$ ]]; then
        echo "Usage: just release-dry-run patch|minor|major"; exit 1
    fi
    CURRENT=$(tr -d '[:space:]' < VERSION)
    echo "Current version: $CURRENT"
    echo "Bump level: {{ LEVEL }}"
    just release-check
    echo ""
    echo "All checks passed. Run: just release {{ LEVEL }}"

# Bump version, open a release PR, merge on green CI (requires: gh)
release LEVEL: release-check
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
        read -r -p "Not on main (currently on $BRANCH). Switch to main? [y/N] " REPLY || REPLY=""
        if [[ "$REPLY" =~ ^[Yy]$ ]]; then
            git checkout main
        else
            echo "Aborted: release must run from main"; exit 1
        fi
    fi
    git pull --ff-only origin main
    git fetch --tags origin
    OLD=$(tr -d '[:space:]' < VERSION)
    IFS='.' read -r MAJ MIN PAT <<< "$OLD"
    case "{{ LEVEL }}" in
        patch) PAT=$((PAT + 1)) ;;
        minor) MIN=$((MIN + 1)); PAT=0 ;;
        major) MAJ=$((MAJ + 1)); MIN=0; PAT=0 ;;
    esac
    VERSION="${MAJ}.${MIN}.${PAT}"
    echo "Releasing v${OLD} -> v${VERSION}"
    printf '%s\n' "$VERSION" > VERSION
    # Regenerate the marketing site's version constant from VERSION.
    # site/src/version.js is the single source of truth that Nav.jsx and
    # Footer.jsx read at runtime.
    printf '// Single source of truth — regenerated by `just release`.\nwindow.TIHOLE_VERSION = "%s";\n' \
      "$VERSION" > site/src/version.js
    # Promote [Unreleased] to [VERSION] with auto-generated entries from
    # conventional commits since the last tag.
    TODAY=$(date -u +%Y-%m-%d)
    PREV_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    if [[ -n "$PREV_TAG" ]]; then
        COMMIT_LOG=$(git log "${PREV_TAG}..HEAD" --pretty=format:"%s" --no-merges | grep -v '^release:' || true)
    else
        COMMIT_LOG=$(git log --pretty=format:"%s" --no-merges | grep -v '^release:' || true)
    fi
    ADDED=()
    FIXED=()
    CHANGED=()
    RE_FEAT='^feat(\([^)]*\))?!?:[[:space:]](.+)'
    RE_FIX='^fix(\([^)]*\))?!?:[[:space:]](.+)'
    RE_OTHER='^(refactor|perf|chore|ci|docs|test|build|style)(\([^)]*\))?!?:[[:space:]](.+)'
    while IFS= read -r msg; do
        [[ -z "$msg" ]] && continue
        if [[ "$msg" =~ $RE_FEAT ]]; then
            ADDED+=("${BASH_REMATCH[2]}")
        elif [[ "$msg" =~ $RE_FIX ]]; then
            FIXED+=("${BASH_REMATCH[2]}")
        elif [[ "$msg" =~ $RE_OTHER ]]; then
            CHANGED+=("${BASH_REMATCH[3]}")
        fi
    done <<< "$COMMIT_LOG"
    CL_SECTION=$(mktemp)
    trap 'rm -f "$CL_SECTION"' EXIT
    {
        echo "## [Unreleased]"
        echo ""
        echo "## [${VERSION}] - ${TODAY}"
        if (( ${#ADDED[@]} )); then
            echo ""
            echo "### Added"
            echo ""
            for b in "${ADDED[@]}"; do echo "- $b"; done
        fi
        if (( ${#FIXED[@]} )); then
            echo ""
            echo "### Fixed"
            echo ""
            for b in "${FIXED[@]}"; do echo "- $b"; done
        fi
        if (( ${#CHANGED[@]} )); then
            echo ""
            echo "### Changed"
            echo ""
            for b in "${CHANGED[@]}"; do echo "- $b"; done
        fi
        if (( ! ${#ADDED[@]} && ! ${#FIXED[@]} && ! ${#CHANGED[@]} )); then
            echo ""
            echo "### Changed"
            echo ""
            echo "- version bump only (no conventional commits since ${PREV_TAG})"
        fi
    } > "$CL_SECTION"
    awk -v section="$CL_SECTION" '
        /^## \[Unreleased\]/ {
            while ((getline line < section) > 0) print line
            next
        }
        { print }
    ' CHANGELOG.md > CHANGELOG.md.tmp
    mv CHANGELOG.md.tmp CHANGELOG.md
    # Refresh the link-reference footer.
    REPO="https://github.com/z19r/tihole"
    if grep -q '^\[Unreleased\]:' CHANGELOG.md; then
        sed -i "s#^\[Unreleased\]:.*#[Unreleased]: ${REPO}/compare/v${VERSION}...HEAD#" \
            CHANGELOG.md
        printf '[%s]: %s/releases/tag/v%s\n' "$VERSION" "$REPO" "$VERSION" \
            >> CHANGELOG.md
    fi
    # Regenerate site/src/changelog.js from the now-promoted CHANGELOG.md
    # so the marketing site's Releases section always matches what shipped.
    go run {{ pkg }} changelog-sync
    # Regenerate site/src/metadata.js — all dynamic site values that JSX
    # components read at runtime (release date, SHA, tagline, screens).
    SHA=$(git rev-parse --short HEAD)
    HUMAN_DATE=$(date -u +"%b %d %Y" | tr -s ' ' | tr '[:lower:]' '[:upper:]')
    TAGLINE=$(sed -n '/^## \[[0-9]/,/^## \[/{/^- /{ s/^- //; s/\*\*//g; p; q; }}' CHANGELOG.md | sed 's/"/\\"/g')
    : "${TAGLINE:=tihole v${VERSION}}"
    SCREENS=$(find internal/tui/screens -mindepth 1 -maxdepth 1 -type d ! -name placeholder | wc -l | tr -d ' ')
    printf '// Dynamic site metadata — regenerated by `just release`.\nwindow.TIHOLE_META = {\n  releaseDate: "%s",\n  releaseDateHuman: "%s",\n  sha: "%s",\n  tagline: "%s",\n  screens: %s,\n  install: {\n    goLine: "go install github.com/z19r/tihole/cmd/tihole@latest",\n    configLine: "~/.config/tihole/config.yaml",\n  },\n};\n' \
      "$TODAY" "$HUMAN_DATE" "$SHA" "$TAGLINE" "${SCREENS:-0}" \
      > site/src/metadata.js
    git checkout -b "release/v${VERSION}"
    git add VERSION CHANGELOG.md site/
    git commit -m "release: v${VERSION}"
    git push -u origin "release/v${VERSION}"
    gh pr create \
        --title "release: v${VERSION}" \
        --body "Bump to v${VERSION} ({{ LEVEL }} release)" \
        --base main

    echo "Waiting for CI checks to appear..."
    for i in $(seq 1 30); do
        if gh pr checks --json name 2>/dev/null | grep -q name; then break; fi
        sleep 2
    done
    echo "Watching CI checks..."
    gh pr checks --watch --fail-fast

    # Cooper&Wright may auto-merge the PR first; only merge if still open.
    if [[ "$(gh pr view --json state -q .state)" == "OPEN" ]]; then
        echo "CI passed. Merging..."
        gh pr merge --squash --delete-branch
    fi

    git checkout main
    git pull --ff-only origin main

    echo "Watching release workflow..."
    sleep 5
    RUN_ID=$(gh run list --workflow release.yml --branch main --limit 1 \
        --json databaseId -q '.[0].databaseId')
    gh run watch "$RUN_ID" --exit-status

    echo ""
    echo "Release v${VERSION} complete."

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
    # Refuse to promote an empty [Unreleased] section.
    UNRELEASED=$(awk '
        /^## \[Unreleased\]/ { found=1; next }
        found && /^## \[/ { exit }
        found && /^- / { print; exit }
    ' CHANGELOG.md)
    if [[ -z "$UNRELEASED" ]]; then
        echo "Error: [Unreleased] section in CHANGELOG.md is empty."
        echo "Add changelog entries before releasing."
        exit 1
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
    git push origin main
    git push origin "v${PRE}"
