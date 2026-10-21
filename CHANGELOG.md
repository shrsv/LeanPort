# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
- `tutorial/permctl` — a verified authorization CLI with nine Lean theorems and
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
