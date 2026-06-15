# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

This repository publishes the core `espyna-golang` module **and** the provider
adapter modules under `contrib/*`, each released as its own Go module. Contrib
modules are tagged with the subdirectory-prefixed convention
(`contrib/<name>/vX.Y.Z`) and share this changelog.

## [Unreleased]

## [0.1.0-alpha] - 2026-06-15

First published alpha of the core business framework and its provider adapters.

### Added
- Core framework module `github.com/erniealice/espyna-golang`: hexagonal
  architecture, entity use cases across the domain set, provider system,
  CEL-based workflow engine, and the composition/DI container.
- Provider adapter modules under `contrib/` (each independently versioned):
  `postgres`, `mysql`, `sqlserver`, `google`, `azure`, `aws`, `gin`, `fiber`,
  `grpc`, `asiapay`, `maya`, `paypal`, `calendly`, `microsoft`.

### Changed
- `go.mod` files now reference published module tags (`v0.1.0-alpha`) instead of
  local `replace` directives. Local multi-module development continues via the
  workspace `go.work` file.

[Unreleased]: https://github.com/erniealice/espyna-golang/compare/v0.1.0-alpha...HEAD
[0.1.0-alpha]: https://github.com/erniealice/espyna-golang/releases/tag/v0.1.0-alpha
