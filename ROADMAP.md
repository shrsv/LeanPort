# Roadmap

LeanPort is deliberately being built as one complete vertical slice before any
generality. The acceptance criterion for v0.1 is narrow and testable:

> A developer or coding agent can create, prove, build, import and use one
> verified Lean module from Go without writing C glue or learning Lean's native
> linking machinery.

## v0.1.0 — the vertical slice

- [ ] `leanport init` — detects the Go module, pins Lean/Go/C toolchains, writes
      `lean/`, `leanport.toml`, `flake.nix`, `.envrc`, CI workflow, `AGENTS.md`.
- [ ] `leanport new <module>` — scaffolds policy, proofs, boundary and binding entry.
- [ ] `leanport check` — runs Lake, rejects `sorry`/`admit`, validates that every
      declared export exists in the generated C.
- [ ] `leanport build` — compiles generated C, selects runtime-free or
      runtime-linked mode from the export's actual symbol needs, archives
      `native/<goos>_<goarch>/lib<module>.a`, generates the Go API + cgo bridge,
      writes `MANIFEST.json`.
- [ ] Rich boundary: `String` ↔ `string`, `ByteArray` ↔ `[]byte`, arrays/lists
      ↔ slices, `Option` ↔ `(value, ok)`, records ↔ Go structs, in
      runtime-linked mode. Mechanism validated; generator support in progress.
- [ ] `leanport docs` — emits `SPEC.md` per module and syncs `AGENTS.md`.
- [ ] `leanport doctor` — reports toolchain presence, versions and target support.
- [ ] `tutorial/permctl` — a working command-line application whose authorization
      policy is written and proved in Lean, with an HTTP-free, stdlib-only Go CLI.
- [ ] `scripts/e2e.sh` — end-to-end acceptance run including a deliberate
      proof-hole failure.
- [ ] Nix flake pinning Lean 4.29.1, Go 1.24, and a C toolchain; CI validates it.
- [ ] Binary releases for linux/darwin/windows × amd64/arm64 with checksums.

## v0.2 — ergonomics

- [ ] Infer the boundary signature from `@[export]` declarations instead of
      requiring a `[[binding]]` table.
- [ ] `leanport watch` for a fast proof-check loop.
- [ ] Richer diagnostics: map Lean elaboration errors to source ranges with
      suggested fixes.
- [ ] Multi-module projects and shared Lean packages via Lake dependencies.

## v0.3 — ergonomics at scale

- [ ] Byte-buffer and string **borrowing** (zero-copy views) on top of the
      copying boundary.
- [ ] Streaming/iterator exports for large collections.
- [ ] Optional dynamically-linked runtime mode for very large generated modules.
- [ ] Cross-target artifact builds in CI, including darwin/arm64 and windows.

## Explicit non-goals

- Translating Lean source into Go source.
- Automatically proving arbitrary requirements. LeanPort automates scaffolding,
  proof checking and build integration. Someone — a person or an agent — must
  still write the proofs.
- Replacing Lean, Lake, or the Lean compiler.
- Making a cgo binding buildable with `CGO_ENABLED=0`.
- Arbitrary-precision `Nat`/`Int` or `IO` values crossing the boundary.

## Deferred with a decision recorded

- **`flake.lock`** requires network access to a Nix binary cache. It is committed
  by the first CI run on a machine with Nix, or in a generated `nix flake lock
  --commit-lock-file` step in `verify.yml`.
