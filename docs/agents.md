# Guide for AI agents

This document is the contract for coding agents working in a LeanPort project.
It tells you where things live, what you may edit, how to verify your work, and
how to interpret failures.

## The one rule

**Edit Lean under `lean/`. Never edit generated output under
`internal/verified/`.**

Generated directories carry a header comment saying so. If generated output is
wrong, the fix belongs in the Lean source or in `leanport.toml`, followed by
`leanport build`.

## Layout

```
<project>/
├── leanport.toml                     # the contract: modules, exports, types, targets
├── lean/                             # YOU EDIT HERE
│   ├── lean-toolchain                # pinned Lean version; do not float it casually
│   ├── lakefile.toml
│   └── LeanPort/<Module>/
│       ├── Policy.lean               # specification + executable implementation
│       ├── Proofs.lean               # theorems about the implementation
│       ├── Boundary.lean             # runtime-free scalar boundary + @[export]
│       └── SPEC.md                   # generated summary of what is guaranteed
└── internal/verified/<pkg>/          # GENERATED — do not edit
    ├── <pkg>.go                      # public Go API: types and constants
    ├── <pkg>_cgo.go                  # cgo layer and Go wrappers
    ├── artifact.go                   # archive digest
    ├── native/<goos>_<goarch>/*.a    # native archive (+ Lean runtime if linked)
    └── MANIFEST.json                 # toolchain + export + artifact digests
```

## The loop

```bash
leanport check     # type-check proofs; fails on 'sorry'/'admit' and missing exports
leanport build     # generate + compile + archive + generate Go; runs check first
go test ./...      # verify the Go wiring
leanport docs      # refresh each module's SPEC.md
```

## Where a change belongs

| Change | Edit | Then run |
|---|---|---|
| Policy rule (e.g. who may delete) | `lean/.../Policy.lean` | `leanport build` |
| Missing/too-weak guarantee | `lean/.../Proofs.lean` | `leanport check` |
| New value passed across the boundary | `lean/.../Boundary.lean` + `leanport.toml` binding | `leanport build` |
| New action or level code | `leanport.toml` code table + `Policy.lean` | `leanport build` |
| Add a whole module | `leanport new <module>` | `leanport build` |
| Application wiring | Go code outside `internal/verified/` | `go test ./...` |

## Boundary constraints

There are two boundary modes; `leanport build` chooses automatically.

**Runtime-free** (scalar-only): only `Bool`, `UInt8/16/32/64`, `Int8/16/32/64`,
`USize`, `Float`/`Float32`, `Char`, and action codes. No `Option`, `Nat`,
strings, arrays, `Repr`, or IO on the exported path. Invalid input returns the
deny code, never panics or allocates.

**Runtime-linked** (`String` and `ByteArray` in v0.1.1): values are copied in
and out and ownership is handled by generated glue. Do not hand-write
conversions. Arrays, records, `Option` and `Except` have validated ABI mappings
but no generator yet; using them in `leanport.toml` fails the build.

The generated glue adds no locking. If Go may call a linked binding from
multiple goroutines, serialize the calls.

If `leanport build` reports an unexpected runtime symbol in a module you intended
to keep runtime-free, simplify `Boundary.lean` or set `runtime = "linked"` in the
binding. Do not add flags to suppress the check.

## Interpreting failures

- **`sorry`/`admit` found** — a proof obligation is open. It names the file and
  line. Close the proof; do not delete the theorem.
- **`export X not found in generated C`** — the `@[export X]` declaration is
  missing, misspelled, or the module did not build.
- **`result type <T> is not supported by the generator yet`** — the token is
  recognised but v0.1.1 cannot generate it. See `docs/limitations.md`.
- **Lean elaboration error** — the theorem or definition does not type-check.
  Fix the Lean source.

## Rules of engagement

1. Never weaken a specification to make a proof pass without recording why in
   the module's `SPEC.md`.
2. Never add `sorry` to reach a green build.
3. Never hand-edit generated files, and never commit a `leanport build` output
   without running `leanport check` in the same commit. Generated archives are
   normally committed for runtime-free modules and gitignored for linked ones.
4. When the policy changes, add or update a Go test that exercises the changed
   decision, in addition to the Lean proof.
5. Report the exact commands you ran and their results.
