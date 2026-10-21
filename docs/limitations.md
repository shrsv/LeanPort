# Limitations

LeanPort is useful within a deliberate, narrow boundary. Knowing the edges is
part of using it honestly.

## 1. Two boundary modes, chosen automatically

Every export is supported. What differs is how much of the Lean runtime the
generated archive carries.

**Runtime-free mode** (scalar-only exports). The archive contains the module's
object plus a few fail-closed stubs for the runtime entry points that appear in
the module initializer's dead code path. Nothing of the Lean runtime is linked.

| Lean | Go |
|---|---|
| `Bool` | `bool` |
| `UInt8/16/32/64`, `Int8/16/32/64` | `uint8/16/32/64`, `int8/16/32/64` |
| `USize` | `uint` |
| `Float`, `Float32` | `float64`, `float32` |
| `Char` | `rune` |
| sealed record of the above | Go `struct` (fields flattened across the ABI) |

Measured: archive ≈ 4 KB.

**Runtime-linked mode** (anything boxed). LeanPort merges the Lean runtime,
`libc++`, GMP and libuv into the archive as one relocatable object; only the
members actually referenced are pulled in.

| Lean | Go |
|---|---|
| `String` | `string` (UTF-8, copied) |
| `ByteArray` | `[]byte` (copied both ways) |
| `Array α`, `List α` (`α` scalar) | `[]α` |
| `Option α` | `(α, ok bool)` |
| `Except ε α` | `(α, error)` |
| user inductive with scalar fields | generated enum + `struct` |

Measured (Lean 4.29.1, linux/amd64): merged archive ≈ 30 MB; a hello-world Go
binary linking it is ≈ 14 MB, and `ldd` shows only libc/libm/libgcc — no Lean
shared library, and the consumer needs no Lean headers (the `lean.h`-dependent
glue is compiled into the archive at `leanport build` time, and the Go side sees
only plain C prototypes).

`leanport build` picks the mode by inspecting undefined symbols: if the only
unresolved names are the known dead-path set, it emits stubs; otherwise it
switches to the runtime-linked merge. The choice is recorded in `MANIFEST.json`
and can be overridden with `runtime = "none" | "linked"` in `leanport.toml`.

Consequences worth knowing:

- runtime-linked archives are large, so prefer runtime-free boundaries where the
  domain allows it (codes and scalars are usually enough for decision logic);
- the Lean runtime is a global, GC-adjacent subsystem: initialize once per
  process, and treat calls as not goroutine-safe unless the generated glue says
  otherwise;
- copying semantics apply at the boundary. A `String` or `[]byte` is copied in
  and out; there are no shared mutable buffers.

Not supported in either mode: `Nat`/`Int` (arbitrary precision) and `IO` as a
return type. Convert to fixed-width integers at the boundary.

## 2. The Lean C ABI is unstable

Lean's FFI is documented as unstable. LeanPort pins the Lean toolchain version
in `lean/lean-toolchain` and records it in `MANIFEST.json`. Upgrading Lean means
regenerating all archives; the manifest digest change makes this visible in
review. Do not assume artifacts built with one Lean version are usable with
another.

## 3. cgo is required, and `CGO_ENABLED=0` cannot work

A generated binding is a cgo package. Consumers need:

- a C compiler at application build time;
- a native archive matching the target `goos/goarch`;
- a C toolchain available for that target when cross-compiling.

`CGO_ENABLED=0` pure-Go builds cannot link a generated binding. This is a
consequence of the design goal, not an oversight: verified logic in Lean means
native code in the binary.

Cross-compiling requires an appropriate target C toolchain, exactly as any
other cgo project does.

## 4. Archives are per-target

`native/linux_amd64/libAuthz.a` does not work on `darwin/arm64`. Declare the
targets you support in `leanport.toml` and build (or fetch) each one. A target
with no archive is reported by `leanport doctor` and fails at `go build` with a
clear message.

## 5. Proofs prove the model, not the application

`leanport check` proves theorems about the decision function under its modelled
inputs. It does not prove:

- that the Go layer authenticates the caller correctly;
- that `ownsResource` is computed from trustworthy data;
- that the action chosen for a request matches the route;
- that authorization and the database mutation are atomic under concurrency;
- anything about deployment.

Those remain ordinary engineering, and ordinary testing applies. The useful
split is: prove the decision rules once, in Lean; test the wiring in Go.

## 6. LeanPort cannot invent proofs

LeanPort automates scaffolding, proof checking and build integration. If a
required guarantee has no proof, `leanport check` fails and someone — a person
or an agent — must write the missing proof. This is the intended division of
labour.

## 7. Nix pins the environment, not the artifacts

The flake pins Lean, Go and the C toolchain. It does not pin every Go module
(that is `go.mod`/`go.sum`) or every Lean package (that is
`lake-manifest.json`). `flake.lock` also cannot be generated without network
access to a Nix binary cache; the first CI run on a machine with Nix commits it.

## 8. Time and cold-start cost

`lake build` on a fresh Lean toolchain downloads packages and can take minutes.
`leanport build` is a maintainer action, not a consumer action. If your CI
rebuilds Lean artifacts on every commit, cache `lean/.lake` and the elan
toolchain.
