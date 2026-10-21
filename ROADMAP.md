# Roadmap

LeanPort is deliberately built as one complete vertical slice before any
generality. The acceptance criterion for v0.1 is narrow and testable:

> A developer or coding agent can create, prove, build, import and use one
> verified Lean module from Go without writing C glue or learning Lean's native
> linking machinery.

## v0.1.0 — the vertical slice (released 2026-10-21)

- [x] `leanport init` — detects the Go module, pins Lean/Go/C toolchains, writes
      `lean/`, `leanport.toml`, optional `flake.nix`/`.envrc`, CI workflow and
      `AGENTS.md`.
- [x] `leanport new <module>` — scaffolds policy, proofs, boundary and binding
      entry.
- [x] `leanport check` — runs Lake, rejects `sorry`/`admit` in Lean code (comments
      stripped), validates that every declared export exists in the generated C.
- [x] `leanport build` — compiles generated C, selects runtime-free or
      runtime-linked mode from the binding types and actual symbol needs,
      archives `native/<goos>_<goarch>/lib<module>.a`, generates the Go API,
      cgo layer, archive digest and `MANIFEST.json`.
- [x] Rich boundary: `String` ↔ `string` and `ByteArray` ↔ `[]byte` in
      runtime-linked mode, including argument and result directions.
- [x] `leanport docs` — emits `SPEC.md` per module.
- [x] `leanport doctor` — reports toolchain presence, versions and project
      configuration.
- [x] `tutorial/permctl` — a working command-line application whose authorization
      policy is written and proved in Lean, with a stdlib-only Go CLI.
- [x] `scripts/e2e.sh` — end-to-end acceptance run including a deliberate
      proof-hole failure and a rich-types build.
- [x] Nix flake providing Go, gcc and helpers, with Lean pinned by
      `lean-toolchain` and fetched by elan; CI validates it.
- [x] Binary releases for linux/darwin/windows × amd64/arm64 with checksums.

## v0.2 — richer generated boundary

The ABI and bridge for these were validated while building v0.1; the generator
does not emit them yet.

- [ ] Sealed records ↔ Go structs (scalar fields stay runtime-free; boxed fields
      use runtime-linked mode).
- [ ] `Array α` / `List α` ↔ `[]α` for scalar element types.
- [ ] `Option α` ↔ `(value, ok)` and `Except ε α` ↔ `(value, error)`.
- [ ] Infer boundary signatures from `@[export]` declarations instead of
      requiring a `[[binding]]` table.
- [ ] Thread-safe glue (a mutex or a runtime lock around linked calls).
- [ ] `leanport watch` for a fast proof-check loop.
- [ ] Richer diagnostics: map Lean elaboration errors to source ranges with
      suggested fixes.
- [ ] Multi-module projects and shared Lean packages via Lake dependencies.

## v0.3 — ergonomics at scale

- [ ] Byte-buffer and string **borrowing** (zero-copy views) on top of the
      copying boundary.
- [ ] Streaming/iterator exports for large collections.
- [ ] Optional dynamically-linked runtime mode for very large generated modules.
- [ ] Cross-target artifact builds in CI, including darwin/arm64 and windows,
      with a target C toolchain.
- [ ] Commit `flake.lock` and validate the Nix path in CI with a pinned nixpkgs.

## Explicit non-goals

- Translating Lean source into Go source.
- Automatically proving arbitrary requirements. LeanPort automates scaffolding,
  proof checking and build integration. Someone — a person or an agent — must
  still write the proofs.
- Replacing Lean, Lake, or the Lean compiler.
- Making a cgo binding buildable with `CGO_ENABLED=0`.
- Arbitrary-precision `Nat`/`Int` or `IO` values crossing the boundary.

## Deferred with a decision recorded

- **`flake.lock`** requires network access to a Nix binary cache. It is
  intentionally not committed in v0.1; the first `nix develop` or CI run creates
  it. Commit it to pin nixpkgs for reproducible environments.
- **Cross-compilation** is limited to the host target in v0.1.0. Archives are
  per-target, and the config already records intended targets; a target C
  toolchain is required before `leanport build --target` can do the work.
