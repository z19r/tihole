# Releasing tihole

This file describes how a tihole release is cut and what each moving part does.

## Short answer

The supported release path is:

```text
just release patch|minor|major
```

That command runs the release gate, bumps `VERSION`, generates the
changelog section from conventional commits, regenerates the site, pushes a
`release/vX.Y.Z` branch, opens a release PR, waits for CI, squash-merges
it, and watches the release workflow.

The merge changes `VERSION` on `main`, which starts
`.github/workflows/release.yml`. It re-runs the release gate, creates the
`vX.Y.Z` tag, and runs goreleaser to build the binaries and publish the
GitHub release.

## The pieces

| File | Role |
|------|------|
| `VERSION` | Single source of truth for the version (no `v` prefix). |
| `CHANGELOG.md` | Keep-a-Changelog history; `## [Unreleased]` at the top. |
| `site/src/changelog.js` | Generated browser global consumed by the site. |
| `site/src/version.js` | Generated `TIHOLE_VERSION` for the site. |
| `site/src/metadata.js` | Generated release date, SHA, tagline, screen count. |
| `cmd/tihole/changelog_sync.go` | `tihole changelog-sync` — regenerates the JS. |
| `justfile` | `release-check`, `release-dry-run`, `release`, `changelog-sync`. |
| `.goreleaser.yaml` | Build matrix, archives, checksums, GitHub release. |
| `.github/workflows/ci.yml` | Format + build + vet + test on push/PR. |
| `.github/workflows/release.yml` | Tag + goreleaser when `VERSION` changes on `main` (or a `v*` tag is pushed). |
| `scripts/backfill-changelog.sh` | Drafts entries for versions that shipped with no changelog content. |

## Commit messages drive the changelog

Release notes are generated from the subjects of non-merge commits since
the previous tag (PR titles, since PRs are squash-merged):

| Subject prefix | Section |
|----------------|---------|
| `feat:` / `feat(scope):` | Added |
| `fix:` / `fix(scope):` | Fixed |
| `refactor` `perf` `chore` `ci` `docs` `test` `build` `style` | Changed |

Anything else (no conventional prefix) is left out. If nothing matches,
the section reads "version bump only". Entries already under
`## [Unreleased]` are kept and land under the new version too.

## Local flow: `just release <level>`

`just release-dry-run <level>` runs the same gate without changing anything.

Preconditions:

- clean working tree
- on `main` (offers to switch if not)
- `main` fast-forwards to `origin/main`

Steps:

1. `just release-check`: gofmt/golines, `go vet`, `go build`, `go test`.
2. Pull `main`, fetch tags, and compute the next version from `VERSION`.
3. Write `VERSION` and `site/src/version.js`.
4. Replace `## [Unreleased]` with a fresh empty one above
   `## [X.Y.Z] - <today>` and the generated sections; update the link
   footer.
5. `tihole changelog-sync` regenerates `site/src/changelog.js`; regenerate
   `site/src/metadata.js`.
6. Commit `release: vX.Y.Z` on `release/vX.Y.Z`, push, open the PR.
7. Wait for CI (`gh pr checks --watch --fail-fast`), then squash-merge
   unless it was already auto-merged.
8. Return to `main`, pull, and `gh run watch` the release workflow.

If CI fails the recipe stops with the PR open; fix it on the branch and
merge, and the release workflow runs on merge as usual.

## Publishing (CI)

`.github/workflows/release.yml`:

1. reads the tag from `VERSION` (or the pushed tag)
2. re-runs `just release-check`
3. creates and pushes `vX.Y.Z` if it does not exist yet
4. runs goreleaser (`release --clean`), which builds `tihole` for linux and
   darwin on amd64 and arm64, stamps the version via
   `-ldflags -X main.version=...`, packs `.tar.gz` archives with
   `README.md`, `LICENSE*`, and `CHANGELOG.md`, generates `checksums.txt`,
   and creates the GitHub release
5. attests build provenance over `dist/checksums.txt`

Re-running a failed release workflow is safe: an existing tag is reused.

## Contributor checklist

1. use conventional PR titles (`feat:`, `fix:`, `chore:` …)
2. `just release-dry-run patch|minor|major`
3. `just release patch|minor|major`
4. review the generated notes in the release PR

## Just the changelog

To regenerate the site changelog without cutting a release:

```text
just changelog-sync
```

## Recovering a missed changelog entry

If a version was tagged with an empty section in `CHANGELOG.md` (and is
therefore also missing from `site/src/changelog.js`, since
`changelog-sync` skips sections with no bullets), draft and backfill it:

```text
scripts/backfill-changelog.sh            # dry run: prints a draft per
                                          # empty version, built from the
                                          # conventional-commit messages
                                          # between that version's tag and
                                          # the previous one
scripts/backfill-changelog.sh --write    # insert the draft and re-run
                                          # changelog-sync
```

Review the draft (or the diff after `--write`) before committing — it's a
starting point assembled from raw commit subjects, not curated prose.

## Verifying goreleaser config locally

If goreleaser is installed:

```text
goreleaser check
goreleaser release --snapshot --clean   # dry run, no publish
```

## Verifying release provenance

The release workflow signs a SLSA build-provenance attestation over
`dist/checksums.txt`, which covers every published archive. Anyone can
verify a downloaded artifact:

```
gh attestation verify tihole_1.2.3_linux_amd64.tar.gz --repo z19r/tihole
```

Attestations only exist for tags released after this workflow step landed.
