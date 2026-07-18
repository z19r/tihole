# Releasing tihole

This file describes how a tihole release is cut and what each moving part does.

## Short answer

The supported release path is:

```text
just release patch|minor|major
```

That command runs the local checks, bumps `VERSION`, promotes the
`## [Unreleased]` section of `CHANGELOG.md` into a dated version section,
regenerates `site/src/changelog.js`, commits the result, and creates the
`vX.Y.Z` git tag.

Pushing that tag triggers GitHub Actions, which runs goreleaser to build the
binaries and publish the GitHub release.

## The pieces

| File | Role |
|------|------|
| `VERSION` | Single source of truth for the version (no `v` prefix). |
| `CHANGELOG.md` | Keep-a-Changelog history; `## [Unreleased]` at the top. |
| `site/src/changelog.js` | Generated browser global consumed by the site. |
| `cmd/tihole/changelog_sync.go` | `tihole changelog-sync` — regenerates the JS. |
| `justfile` | `release` and `changelog-sync` recipes. |
| `.goreleaser.yaml` | Build matrix, archives, checksums, GitHub release. |
| `.github/workflows/ci.yml` | Build + vet + test on push/PR. |
| `.github/workflows/release.yml` | goreleaser on tag push `v*`. |

## Local flow: `just release <level>`

Preconditions enforced by the recipe:

- clean working tree
- on the `main` branch
- `VERSION` and `CHANGELOG.md` present

Steps performed:

1. Run `go build ./...`, `go vet ./...`, and `go test ./...`.
2. Read `VERSION` and compute the next version for the requested level
   (`patch` / `minor` / `major`).
3. Write the new version to `VERSION`.
4. In `CHANGELOG.md`, insert a fresh empty `## [Unreleased]` above a new
   `## [X.Y.Z] - <today>` heading, moving the previously unreleased notes
   under the new version.
5. Run `tihole changelog-sync` to regenerate `site/src/changelog.js` from the
   now-promoted `CHANGELOG.md`.
6. `git add` `VERSION`, `CHANGELOG.md`, `site/src/changelog.js`.
7. Commit `release: vX.Y.Z`.
8. Create the annotated tag `vX.Y.Z`.

The recipe stops before pushing so you can review the commit and tag.

## Publishing

Push `main` and the tag:

```text
git push origin main
git push origin vX.Y.Z
```

The tag push starts `.github/workflows/release.yml`, which runs goreleaser
(`release --clean`) using the automatically provided `GITHUB_TOKEN`.
goreleaser:

1. builds `tihole` for linux and darwin on amd64 and arm64
2. stamps the version into the binary via `-ldflags -X main.version=...`
3. packs `.tar.gz` archives with `README.md`, `LICENSE*`, and `CHANGELOG.md`
4. generates `checksums.txt`
5. creates the GitHub release for `vX.Y.Z` against `z19r/tihole`

## Contributor checklist

1. checkout `main`, pull latest
2. make sure everything you want in the release is under `## [Unreleased]`
3. run `just release patch|minor|major`
4. review the release commit and tag
5. `git push origin main && git push origin vX.Y.Z`
6. watch the release workflow through the GitHub release page

## Just the changelog

To regenerate the site changelog without cutting a release:

```text
just changelog-sync
```

## Verifying goreleaser config locally

If goreleaser is installed:

```text
goreleaser check
goreleaser release --snapshot --clean   # dry run, no publish
```
