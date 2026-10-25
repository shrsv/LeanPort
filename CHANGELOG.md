# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- A six-example learning path under `tutorial/`: executable Lean, pure
  functions, specifications and proofs, business invariants, refined types, and
  the existing `permctl` capstone. `tutorial/README.md` is now the index.
- `tutorial/foundations/` — a checked-in project for Examples 1–5, exercised by
  `scripts/e2e.sh`.

### Changed

- The original `tutorial/README.md` walkthrough is now
  `tutorial/06-authorization.md` (Example 6).

## [0.1.1] - 2026-10-21

Multi-module linking fix.

### Fixed

- Runtime-free archives now emit weak Lean runtime stubs. Importing several
  scalar modules into one Go binary no longer fails with duplicate
  `initialize_Init` and related symbols. Multiple bindings in one package are
  covered too.

### Added

- README and tutorial sections explaining how to add a module after `authz`,
  which commands are required, and how to import the generated package from Go.
- A two-module build and import check in `scripts/e2e.sh`.

### Known limits

- Multiple runtime-linked packages (`String`/`ByteArray`) are not supported.
  Each runtime-linked package embeds the Lean runtime, so keep those exports in
  one package.

## [0.1.0] - 2026-10-21

First working release: the full vertical slice, exercised end to end.

### Added

- `leanport init` — detects the Go module, pins Lean (`lean-toolchain`), writes
  `lean/`, `leanport.toml`, `AGENTS.md`, and optional `flake.nix`, `.envrc` and
  GitHub Actions workflow.
- `leanport new <Module>` — scaffolds `Policy.lean`, `Proofs.lean`,
  `Boundary.lean` and `SPEC.md`, appends a `[[lean_lib]]` to the Lake file and a
  `[[binding]]` to `leanport.toml`.
- `leanport check` — builds every declared module target, rejects `sorry`/`admit`
  in Lean code (line/block comments are stripped first), and verifies each
  `@[export]` symbol appears in the generated C.
- `leanport build` — compiles Lean-generated C, deducts project-defined symbols,
  chooses a boundary mode, generates and compiles the C glue, archives it, and
  writes the Go package, cgo layer, archive digest and `MANIFEST.json`.
- Runtime-free boundary for scalars and action enums; measured archive 12–28 KB
  and no Lean runtime in the Go binary.
- Runtime-linked boundary for `String` and `ByteArray`; measured archive ~29 MB
  and demo binary ~14 MB, with no Lean installation or headers needed by
  consumers.
- `leanport docs` and `leanport doctor`.
- `tutorial/permctl` — a verified authorization CLI with ten Lean theorems and
  39 Go-side checks.
- `examples/rich-types` — `String`/`ByteArray` exports and the runtime-linked
  archive.
- `scripts/e2e.sh` — fresh scaffold, tutorial build/test, deliberate proof-hole
  failure, rich-types build and CLI checks.
- `flake.nix` / `.envrc` — Go, gcc and helper tooling, with Lean pinned by elan.
- `scripts/release.sh`, `.goreleaser.yaml` and a tag-triggered release workflow.
- Six binaries (linux/darwin/windows × amd64/arm64) plus `checksums.txt`.

### Changed

- Corrected the capability description after measuring the real boundary:
  strings and byte arrays are supported through the runtime-linked archive;
  records, arrays, `Option` and `Except` are ABI-validated but their generator
  mappings are deferred to v0.2. The docs no longer claim generator support that
  does not exist.
- Clarified that runtime-free archives are tens of kilobytes (they include the
  compiled proofs), not ~4 KB.
- Documented the absence of locking in linked glue: serialize concurrent calls.

### Known limits

- The generator supports scalars, `String` and `ByteArray`. Records, arrays,
  `Option` and `Except` fail the build with a type error until their mappings
  land.
- `leanport build --target` accepts only the host `goos/goarch` in v0.1.0.
- `flake.lock` is not committed (it cannot be generated without Nix); the first
  Nix run creates it.
