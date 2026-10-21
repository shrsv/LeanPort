# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Repository outline: root `README.md` describing the tool, motivation, planned
  usage, vocabulary and limitations.
- `tutorial/README.md` — the build-your-own-verified-CLI walkthrough.
- `tutorial/permctl/README.md` — specification of the worked example app.
- `docs/architecture.md`, `docs/cli-reference.md`, `docs/limitations.md`,
  `docs/agents.md`.
- `ROADMAP.md` describing the v0.1 vertical slice and deferred work.

### Notes

- Pre-alpha. The interface is specified; implementation is in progress.
- The core pipeline (Lean module + proofs → generated C → static archive →
  cgo → Go binary) has been validated locally with Lean 4.29.1 on linux/amd64.

### Changed

- Corrected the boundary specification after measuring the real link: strings,
  byte arrays, arrays, records, optionals and other boxed values **are**
  supported, via a runtime-linked archive (~30 MB) that embeds the Lean runtime,
  libc++ and GMP statically. Scalar-only exports keep the runtime-free archive
  (~4 KB). Consumers need no Lean installation or Lean headers in either mode.
- Added the annotated project layout to the root README, showing `lean/` (yours),
  `internal/verified/` (generated) and `cmd/` (application).

## [0.1.0] - Unreleased

### Added

- `leanport init`, `new`, `check`, `build`, `docs`, `doctor`, `version`.
- Generated Go bindings with a runtime-free scalar boundary.
- `tutorial/permctl` verified authorization CLI.
- `scripts/e2e.sh` acceptance run.
- Nix flake and GitHub Actions verify/release workflows.
- Binary releases for linux/darwin/windows × amd64/arm64.
