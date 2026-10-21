# Architecture

This document describes how LeanPort moves from Lean source to a Go package,
and why each step exists.

## The pipeline

```
lean/                             leanport build                        generated
──────────────────────────────    ──────────────────────────────────    ──────────────────────────────
LeanPort/Authz/Policy.lean        lake build                            internal/verified/authz/
LeanPort/Authz/Proofs.lean   ──►  .lake/build/ir/**/Policy.c       ──►  authz.go          (public Go API)
LeanPort/Authz/Boundary.lean      cc -c        ──► Policy.o             authz_cgo.go      (cgo layer)
                                  nm -u check  ──► stub policy         bridge.h          (C ABI)
@[export leanport_authz_…]        ar rcs       ──► libAuthz.a         bridge.c          (stubs + init)
                                                           │           native/linux_amd64/libAuthz.a
                                                           │           MANIFEST.json
                                                           ▼
                                                    go build ./...  links the archive
```

Three distinct kinds of check, each doing a different job:

1. **Lean proofs** (`leanport check`) — machine-checked theorems about the
   policy function's behaviour under its modelled inputs.
2. **Go tests** (`go test ./...`) — check that the Go application wires the
   identity, resource lookup and action mapping to the verified policy
   correctly.
3. **Native link** (`leanport build` then `go build`) — confirms the generated C
   compiles and links into an ordinary Go binary.

Integration tests are not redundant with the proofs: the theorems say nothing
about HTTP routes, session handling, or database consistency.

## Why Lean compiles to C, and why we archive it

Lean's compiler emits C for each module, and the official FFI is documented as
unstable. LeanPort therefore isolates the FFI behind a generated, narrow
adapter and pins the Lean toolchain version in the module manifest so artifacts
can be regenerated deterministically.

Go's `cgo` can link prebuilt static archives with package-level directives. A
per-target static archive plus generated cgo directives means the consumer runs
an ordinary `go build` with no Lean toolchain installed.

### Validated behaviour (Lean 4.29.1, linux/amd64)

These facts were established by building the tutorial's policy before writing
this document, and they drive the implementation:

- `lake build` writes generated C to `.lake/build/ir/<Module/Path>.c`.
- `@[export name]` on `def f (a b : UInt8) : UInt8` emits a C function
  `uint8_t name(uint8_t, uint8_t)` with unmangled linkage.
- Compiling that C with the flags from `leanc --print-cflags` needs only the
  Lean include directory; it does **not** require the Lean runtime to link.
- A boundary written with `Bool`/`UInt8` only leaves exactly five undefined
  symbols, all reached from the module initializer's dead code path:

  ```
  initialize_Init
  lean_inc_heartbeat
  mi_malloc_small
  lean_dec_ref_cold
  lean_internal_panic_out_of_memory
  ```

- The resulting archive links into a Go binary with plain `cgo` directives and
  runs correctly, with no Lean runtime present.

## The runtime-free boundary

The boundary function is the only place where C scalars meet Lean values. The
convention keeps it free of the Lean runtime:

- Decode with `UInt8`/`Bool` comparisons, not `Option`.
- Encode results as `UInt8` codes (`0` deny, `1` allow) and map them in Go.
- Do not `deriving Repr` on types used by an exported path.
- Make the boundary total: invalid input must return the deny code, never panic.

Using `Option`, `Nat` conversions, strings, arrays, `Repr`, or IO in a exported
path pulls live runtime symbols into the archive. `leanport build` detects this
by inspecting undefined symbols and fails with the offending names rather than
producing an archive that fails to link later.

The five dead-path symbols above are satisfied by a generated `bridge.c` that
provides fail-closed stubs. They are reachable only from the module initializer
and allocation paths, which scalar exports never invoke. If an export ever
needs them at runtime, it is out of the v0.1 boundary — by design.

## Generated artifacts

For a module `authz` targeting `linux/amd64`:

```
internal/verified/authz/
├── authz.go                                  # public Go API (hand-worthy, generated)
├── authz_cgo.go                              # cgo directives + nativeAuthorize
├── bridge.h                                  # extern declaration of the export
├── bridge.c                                  # runtime stubs
├── native/linux_amd64/libAuthz.a             # Lean implementation + runtime-free object
├── MANIFEST.json                             # lean toolchain, export table, checksums
├── SPEC.md                                   # generated specification summary
└── AGENTS.md                                 # generated agent notes for this module
```

Everything under `internal/verified/` is generated. Maintainers edit `lean/`.
A header comment in every generated file states this, and CI can verify the
working tree is clean after running `leanport build`.

## `MANIFEST.json`

```json
{
  "schema": 1,
  "generator": "leanport 0.1.0",
  "lean": { "toolchain": "leanprover/lean4:v4.29.1", "compiler": "4.29.1" },
  "target": { "goos": "linux", "goarch": "amd64" },
  "exports": [
    { "c_symbol": "leanport_authz_authorize", "go": "Authorize",
      "args": ["bool", "bool", "bool", "action:Action"], "result": "bool" }
  ],
  "artifacts": [
    { "path": "native/linux_amd64/libAuthz.a", "sha256": "…" }
  ]
}
```

The manifest is what makes an accidental Lean upgrade visible in review: a
toolchain change alters the digest of every archive.

## Why Nix

Lean, Go and a C toolchain must agree on versions for the artifacts to be
reproducible. Nix pins all three plus the generated archive inputs. Nix pins the
*build environment*; Go modules continue to pin Go dependencies and Lake
manifests pin Lean packages. These layers are complementary.

Nix is never required to consume a binding.
