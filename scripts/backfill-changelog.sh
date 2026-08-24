#!/usr/bin/env bash
# Finds released version headings in CHANGELOG.md that have no bullet
# content (the bug that let v1.1.0, v1.2.0, and v1.2.1 ship empty), drafts
# entries from the conventional-commit messages between that version's tag
# and the previous one, and — with --write — inserts the draft and
# regenerates site/src/changelog.js so both stay in sync.
#
# Usage:
#   scripts/backfill-changelog.sh            # dry run: print the draft
#   scripts/backfill-changelog.sh --write    # apply it and sync the site
#
# The draft is a starting point, not a final answer: review it (dry run
# output, or the diff after --write) and trim/reword bullets by hand.

set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

WRITE=0
if [[ "${1:-}" == "--write" ]]; then
    WRITE=1
elif [[ -n "${1:-}" ]]; then
    echo "Usage: $0 [--write]" >&2
    exit 1
fi

# Version headings in file order, newest first, excluding [Unreleased].
mapfile -t VERSIONS < <(grep -oP '^## \[\K[^]]+(?=\] - )' CHANGELOG.md)

section_is_empty() {
    awk -v ver="$1" '
        $0 == "## [" ver "]" || index($0, "## [" ver "] ") == 1 { found=1; next }
        found && /^## \[/ { exit }
        found && /^- / { exit 1 }
        END { if (!found) exit 1 }
    ' CHANGELOG.md
}

# Conventional-commit subject -> "Section|bullet text".
classify_commit() {
    local subject="$1" section body
    case "$subject" in
        feat*': '*) section="Added" ;;
        fix*': '*) section="Fixed" ;;
        perf*': '*|refactor*': '*) section="Changed" ;;
        *) section="Changed" ;;
    esac
    body="${subject#*: }"
    body="$(tr '[:lower:]' '[:upper:]' <<<"${body:0:1}")${body:1}"
    [[ "$body" == *. ]] || body="${body}."
    printf '%s|%s\n' "$section" "$body"
}

draft_for_version() {
    local ver="$1" this_tag="v$1" prev_tag="$2"
    local range
    if git rev-parse -q --verify "refs/tags/$this_tag" >/dev/null; then
        range="${prev_tag:+$prev_tag..}$this_tag"
    else
        range="${prev_tag:+$prev_tag..}HEAD"
    fi

    local -A bullets=()
    local order=()
    while IFS=$'\t' read -r subject; do
        [[ "$subject" == release:\ * ]] && continue
        [[ "$subject" == Merge\ * ]] && continue
        local pair section body
        pair="$(classify_commit "$subject")"
        section="${pair%%|*}"
        body="${pair#*|}"
        if [[ -z "${bullets[$section]:-}" ]]; then
            order+=("$section")
        fi
        bullets["$section"]+="- ${body}"$'\n'
    done < <(git log --no-merges --reverse --pretty='%s' "$range" -- . 2>/dev/null || true)

    [[ "${#order[@]}" -eq 0 ]] && return 1

    echo "## [$ver]"
    for section in Added Changed Fixed; do
        [[ -n "${bullets[$section]:-}" ]] || continue
        echo
        echo "### $section"
        echo
        printf '%s' "${bullets[$section]}"
    done
}

FOUND_EMPTY=0
DRAFTS=()
for i in "${!VERSIONS[@]}"; do
    ver="${VERSIONS[$i]}"
    [[ "$ver" == *-* ]] && continue # skip prereleases/betas
    section_is_empty "$ver" || continue
    FOUND_EMPTY=1

    prev_tag=""
    for ((j = i + 1; j < ${#VERSIONS[@]}; j++)); do
        candidate="v${VERSIONS[$j]}"
        if git rev-parse -q --verify "refs/tags/$candidate" >/dev/null; then
            prev_tag="$candidate"
            break
        fi
    done

    draft="$(draft_for_version "$ver" "$prev_tag" || true)"
    if [[ -z "$draft" ]]; then
        echo "warn: no commits found to draft [$ver] from (skipping)" >&2
        continue
    fi
    DRAFTS+=("$draft")
done

if [[ "$FOUND_EMPTY" -eq 0 ]]; then
    echo "No empty version sections in CHANGELOG.md — nothing to backfill."
    exit 0
fi

if [[ "$WRITE" -eq 0 ]]; then
    printf '\n%s\n\n' "----- draft (review, then re-run with --write) -----"
    for draft in "${DRAFTS[@]}"; do
        printf '%s\n\n' "$draft"
    done
    exit 0
fi

for draft in "${DRAFTS[@]}"; do
    ver="$(head -1 <<<"$draft" | sed -E 's/^## \[(.+)\]$/\1/')"
    section_is_empty "$ver" || continue # re-check: may have been filled since
    body="$(tail -n +3 <<<"$draft")" # drop the "## [ver]" line and its blank
    line_no=$(grep -n "^## \[$ver\]" CHANGELOG.md | head -1 | cut -d: -f1)
    insert_after=$((line_no + 1))
    tmp="$(mktemp)"
    printf '%s\n\n' "$body" >"$tmp"
    sed -i "${insert_after}r ${tmp}" CHANGELOG.md
    rm -f "$tmp"
    echo "backfilled [$ver] in CHANGELOG.md"
done

go run ./cmd/tihole changelog-sync
echo "Review the changes (git diff), then commit."
