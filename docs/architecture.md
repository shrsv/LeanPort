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
- For a scalar-only module, compiling that C with the flags from
  `leanc --print-cflags` needs only the Lean include directory; scalar exports
  do **not** require the Lean runtime to link.
- A boundary written with `Bool`/`UInt8` only leaves five core undefined
  symbols (`lean_notify_assert` may join them when an inline header assertion is
  used), all reached from the module initializer's dead code path:

  ```
  initialize_Init
  lean_inc_heartbeat
  mi_malloc_small
  lean_dec_ref_cold
  lean_internal_panic_out_of_memory
  ```

- The resulting archive links into a Go binary with plain `cgo` directives and
  runs correctly, with no Lean runtime present.

And for boxed data:

- `String`, `ByteArray`, `Array`, records and `Option` exports all compile to
  `lean_object*` signatures and reference real runtime functions (for example
  `lean_string_append`, `lean_mk_string`, `lean_alloc_sarray`, `mi_malloc_small`).
- The Lean toolchain ships static `libgmp.a`, `libuv.a`, `libc++.a` and
  `libc++abi.a`, so a fully static runtime link needs no system `-dev` packages.
- Merging the module objects and the pulled runtime members with
  `ld -r --start-group … --end-group` produces one relocatable object
  (measured ≈ 30 MB), whose only remaining undefined symbols are libc/libm.
- A Go program linking that archive through cgo builds, runs, and passes
  `String`, record and `ByteArray` values correctly: measured binary ≈ 14 MB,
  `ldd` shows only libc/libm/libgcc.
- The `lean.h`-dependent glue is compiled at `leanport build` time and placed in
  the archive, so the consumer's Go package contains only plain C prototypes and
  the consumer never needs Lean headers.

## Boundary modes

The boundary function is the only place where C values meet Lean values.
`leanport build` inspects the compiled object and chooses one of two modes.

### Runtime-free mode

Chosen when every export uses only scalars. The conventions keep it free of the
runtime:

- decode with `UInt8`/`Bool` comparisons, not `Option`;
- encode results as `UInt8` codes (`0` deny, `1` allow) and map them in Go;
- avoid `Nat` conversions, strings, arrays, `Repr` and IO on the exported path;
- make the boundary total: invalid input returns the deny code, never panics.

Five dead-path symbols (`initialize_Init`, `lean_inc_heartbeat`,
`mi_malloc_small`, `lean_dec_ref_cold`, `lean_internal_panic_out_of_memory`) are
satisfied by generated fail-closed stubs in `bridge.c`. They are reachable only
from the module initializer and allocation paths, which scalar exports never
invoke. (`lean_notify_assert` also appears when the Lean headers' inline
`assert()` macro is used; it fires only on a violated precondition.)

Archive size: approximately 4 KB. No Lean runtime in the consumer's binary.

### Runtime-linked mode

Chosen automatically when an export uses anything boxed — `String`,
`ByteArray`, arrays, records, `Option`, user inductives. The boundary can be
written naturally; LeanPort handles ownership:

- the generated glue owns the conversion and calls `lean_dec` on values it
  borrows;
- `String` and `ByteArray` are copied in and out, so there are no shared mutable
  buffers;
- the runtime is initialized once per process
  (`lean_initialize_runtime_module`, then each module initializer, then
  `lean_io_mark_end_initialization`).

LeanPort merges the module objects with the runtime members actually referenced
and with `libc++`/GMP/libuv, all statically. Archive size: approximately 30 MB.
The consumer needs no Lean headers and no Lean installation, and `ldd` on the
resulting binary shows only libc/libm/libgcc.

### Choosing

| | runtime-free | runtime-linked |
|---|---|---|
| Supported types | scalars, sealed records of scalars | everything Lean can compile |
| Archive | ~4 KB | ~30 MB |
| Go binary delta | ~0 | ~+12 MB |
| Consumer needs Lean | no | no |
| Preferred for | decision logic, policies, arithmetic | text, buffers, rich results |

Because decision logic rarely needs allocation, most verified modules will use
runtime-free mode. Rich types are available when the domain needs them, at a
size cost recorded in `MANIFEST.json`.

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
