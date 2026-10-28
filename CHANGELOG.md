# Changelog

All notable changes to tihole will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.3.0] - 2026-10-28

### Changed

- cut releases through a PR like whetstone (#26)
- bump charm.land/bubbletea/v2 from 2.0.10 to 2.1.0 (#23)
- bump charm.land/bubbletea/v2 from 2.0.9 to 2.0.10 (#22)
- bump actions/attest-build-provenance from 3 to 4 (#21)
- attest release build provenance (#20)
- bump goreleaser/goreleaser-action from 6 to 7 (#13)
- bump actions/checkout from 4 to 7 (#16)
- bump charm.land/bubbles/v2 from 2.1.1 to 2.2.1 (#15)
- bump actions/setup-go from 5 to 7 (#14)
- bump charm.land/lipgloss/v2 from 2.0.5 to 2.0.6 (#17)
- bump charm.land/bubbletea/v2 from 2.0.8 to 2.0.9 (#19)

## [1.2.1] - 2026-08-24

### Changed

- Release tooling now pushes `main` and the release tag automatically as
  part of `just release`.

## [1.2.0] - 2026-08-24

### Changed

- Dashboard and query log now poll four times a second for snappier
  updates.

### Fixed

- Health gauge bars on the dashboard no longer wrap onto a second line.

## [1.1.0] - 2026-08-06

### Added

- Rich per-setting detail panel on the settings screen.

### Fixed

- Dashboard panels now size to their exact content width instead of
  over- or under-sizing.

## [1.0.0-beta.1] - 2026-07-26

First public beta. Ships the full 1.0.0 feature set (below) as prebuilt
binaries for linux and darwin on amd64/arm64, for testing ahead of the
1.0.0 final.

### Changed

- Canonicalized the module path to `github.com/z19r/tihole` so
  `go install github.com/z19r/tihole/cmd/tihole` resolves.

### Added

- MIT license.

## [1.0.0] - 2026-07-25

### Added

- Keyboard-driven terminal UI for managing Pi-hole v6 instances.
- Dashboard with a host-health strip: CPU, memory, and temperature gauges
  plus FTL diagnostics.
- Query log browser with one-key allow/block actions.
- Blocking pane promoted to a first-class screen for enabling and disabling
  Pi-hole blocking.
- Management screens for domains, groups, clients, adlists, and local DNS
  records.
- System, messages, and network screens for at-a-glance instance state.
- Multiple color themes, including the signature Gloss default with its
  multi-stop gradient treatment.
- Background-bleed-free rendering so styled text never leaks the terminal
  background under or after it.
- Config editor (`tihole config`) that opens automatically when an instance
  is misconfigured, with in-app error surfacing instead of hard startup
  failures.
- ASCII boot splash shown before the dashboard.

[Unreleased]: https://github.com/z19r/tihole/compare/v1.3.0...HEAD
[1.0.0-beta.1]: https://github.com/z19r/tihole/releases/tag/v1.0.0-beta.1
[1.0.0]: https://github.com/z19r/tihole/releases/tag/v1.0.0
[1.1.0]: https://github.com/z19r/tihole/releases/tag/v1.1.0
[1.2.0]: https://github.com/z19r/tihole/releases/tag/v1.2.0
[1.2.1]: https://github.com/z19r/tihole/releases/tag/v1.2.1
[1.3.0]: https://github.com/z19r/tihole/releases/tag/v1.3.0
