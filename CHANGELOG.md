# Changelog

All notable changes to tihole will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-07-25

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

[Unreleased]: https://github.com/z19r/tihole/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/z19r/tihole/releases/tag/v0.1.0
