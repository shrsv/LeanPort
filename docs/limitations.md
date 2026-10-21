# Limitations

LeanPort is useful within a deliberate, narrow boundary. Knowing the edges is
part of using it honestly.

## 1. The exported boundary is scalar-only (v0.1)

Exported functions may take and return:

| Lean | Go | Notes |
|---|---|---|
| `Bool` | `bool` | Encoded as `UInt8` (`1`/`0`) |
| `UInt8/16/32/64` | `uint8/16/32/64` | |
| `Int8/16/32/64` | `int8/16/32/64` | |
| code tables | typed constants | e.g. `Action` ↔ `UInt8` |

Not supported in an exported path in v0.1:

- strings, `ByteArray`, arrays, lists;
- `Option`, `Sum`, `Except` and other inductive returns;
- `Nat` and `Int` (arbitrary precision);
- `IO`, tasks, or anything that allocates.

LeanPort rejects these at `leanport build` time by inspecting the undefined
symbols of the compiled object. You get the offending symbol names and a
pointer to this document, not a linker error three steps later.

Rationale: this is what keeps the archive free of the Lean runtime, which is
what makes `go build` boring for consumers. A wider boundary is planned
(v0.3) with explicit ownership rules.

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
