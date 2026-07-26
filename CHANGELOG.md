# Changelog

All notable changes to tihole will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/z19r/tihole/compare/v1.0.0-beta.1...HEAD
[1.0.0-beta.1]: https://github.com/z19r/tihole/releases/tag/v1.0.0-beta.1
[1.0.0]: https://github.com/z19r/tihole/releases/tag/v1.0.0
