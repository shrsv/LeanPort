# Limitations

LeanPort is useful within a deliberate, narrow boundary. Knowing the edges is
part of using it honestly.

## 1. Two boundary modes

What differs between modes is how much of the Lean runtime the generated archive
carries, not whether consumers need Lean. In both modes the consumer needs only
Go, cgo and a C toolchain.

**Runtime-free mode** (scalar-only exports). The archive contains the module
objects plus a glue object with fail-closed stubs for runtime entry points that
appear in the module initializer's dead code path. No Lean runtime is linked.

| Lean | Go |
|---|---|
| `Bool` | `bool` |
| `UInt8/16/32/64`, `Int8/16/32/64` | `uint8/16/32/64`, `int8/16/32/64` |
| `USize` | `uint` |
| `Float`, `Float32` | `float64`, `float32` |
| `Char` | `rune` |
| `UInt8` action code | generated enum type |

Measured on Lean 4.29.1, linux/amd64: archive ≈ 12–28 KB; the capstone binary is
≈ 2.3 MB and `ldd` shows only libc. (The range is because proofs are compiled
into the same archive.)

**Runtime-linked mode** (`String` and `ByteArray`). LeanPort merges the module
objects with the Lean runtime, `libc++`, GMP and libuv into one relocatable
object. Only the members actually referenced are pulled in.

| Lean | Go |
|---|---|
| `String` | `string` (UTF-8, copied) |
| `ByteArray` | `[]byte` (copied both ways) |

Measured: merged archive ≈ 29 MB; a Go binary using all three rich exports is
≈ 14 MB, and `ldd` shows only libc/libm/libgcc. The `lean.h`-dependent glue is
compiled into the archive at `leanport build` time, so the consumer never needs
Lean headers.

`leanport build` picks the mode by inspecting undefined symbols: if the only
unresolved Lean names are the known dead-path set, it emits stubs; otherwise it
switches to the runtime-linked merge. A binding that uses `string`/`bytes`
always uses linked mode. The choice is recorded in `MANIFEST.json` and can be
forced with `runtime = "none" | "linked"` in `leanport.toml`.

Consequences worth knowing:

- runtime-linked archives are large, so prefer runtime-free boundaries where the
  domain allows it (codes and scalars are usually enough for decision logic);
- runtime-free generated packages can be linked together in one Go binary.
  Runtime-linked packages each embed the Lean runtime, so keep
  `String`/`ByteArray` exports in one package;
- the generated glue has no locking. Lean's runtime is initialized lazily on the
  first call; serialize calls (one goroutine, a mutex, or a worker queue) unless
  you add your own synchronization;
- copying semantics apply at the boundary. A `String` or `[]byte` is copied in
  and out; there are no shared mutable buffers;
- `String` arguments are passed as NUL-terminated C strings, so embedded NUL
  bytes are not representable; use `bytes` for arbitrary binary data.

## 2. The generator covers scalars, strings and byte arrays

The ABI and bridge have been validated for records, arrays/lists, `Option` and
`Except` (see [`architecture.md`](architecture.md)), but the v0.1.1 code
generator does not emit mappings for them. Using one of those tokens in
`leanport.toml` fails the build with a type error.

For richer values today, either:

- encode them as several scalar/`bytes` arguments at the boundary and rebuild
  the value in Lean; or
- wait for generator support (tracked in [`ROADMAP.md`](../ROADMAP.md)).

Records in runtime-free mode, and `Option`/`Except` mappings, are the next
generator targets.

## 3. `Nat`/`Int` and `IO` are not boundary types

Arbitrary-precision `Nat`/`Int` need allocation and are not supported across the
boundary. Convert to fixed-width integers. `IO` is not a return type; a boundary
must be a total function from inputs to a result. Use `Except`/`Option` inside
Lean and encode the outcome as a scalar if you need failure.

## 4. The Lean C ABI is unstable

Lean's FFI is documented as unstable. LeanPort pins the Lean toolchain version
in `lean/lean-toolchain` and records it in `MANIFEST.json`. Upgrading Lean means
regenerating all archives; the manifest digest change makes this visible in
review. Do not assume artifacts built with one Lean version are usable with
another.

## 5. cgo is required, and `CGO_ENABLED=0` cannot work

A generated binding is a cgo package. Consumers need:

- a C compiler at application build time;
- a native archive matching the target `goos/goarch`;
- a C toolchain available for that target when cross-compiling.

`CGO_ENABLED=0` pure-Go builds cannot link a generated binding. This is a
consequence of the design goal, not an oversight: verified logic in Lean means
native code in the binary.

## 6. Cross-compilation is not automated in v0.1.1

`leanport build --target` currently accepts only the host `goos/goarch`, because
the C objects are compiled with the host compiler. Cross-building needs a target
C toolchain and a target Lean toolchain, exactly as any other cgo project does.
The config records all intended targets, and archives are per-target:
`native/linux_amd64/libAuthz.a` does not work on `darwin/arm64`.

## 7. Proofs prove the model, not the application

`leanport check` proves theorems about the decision function under its modelled
inputs. It does not prove:

- that the Go layer authenticates the caller correctly;
- that `ownsResource` is computed from trustworthy data;
- that the action chosen for a request matches the route;
- that authorization and the database mutation are atomic under concurrency;
- anything about deployment.

Those remain ordinary engineering, and ordinary testing applies. The useful
split is: prove the decision rules once, in Lean; test the wiring in Go.

## 8. LeanPort cannot invent proofs

LeanPort automates scaffolding, proof checking and build integration. If a
required guarantee has no proof, `leanport check` fails and someone — a person
or an agent — must write the missing proof. This is the intended division of
labour.

## 9. Nix pins the build environment, not the universe

The flake provides Go, gcc, make and helpers. Lean is pinned by
`lean/lean-toolchain` and fetched by elan, because nixpkgs does not track every
Lean release. Go modules are pinned by `go.mod`/`go.sum`, and Lean packages by
Lake's manifest.

`flake.lock` is intentionally absent from the repository: it cannot be generated
without Nix. The first `nix develop` on a machine with Nix creates it; commit it
to pin nixpkgs.

## 10. Time and cold-start cost

`lake build` on a fresh Lean toolchain downloads packages and can take minutes.
`leanport build` is a maintainer action, not a consumer action. If your CI
rebuilds Lean artifacts on every commit, cache `lean/.lake`, `.leanport/` and
the elan toolchain.
