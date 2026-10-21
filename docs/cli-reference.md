# CLI reference

This document describes `leanport` v0.1.1 and the `leanport.toml` schema it
reads. Everything here is exercised by [`scripts/e2e.sh`](../scripts/e2e.sh).

## Synopsis

```
leanport <command> [flags]
```

Run `leanport help` for the command list. Command flags follow the standard Go
`flag` package, so they must appear **before** positional arguments:
`leanport new --package authz Authz`.

## Global flags

| Flag | Description |
|---|---|
| `-C, --dir <path>` | Run as if LeanPort was started in `<path>`. Applies before the command. |
| `-h, --help` | Show help. |
| `--version` | Print version and exit. |

There is no global `--verbose` in v0.1.1; external commands are echoed when they
matter (for example `→ lake build …`).

## Commands

### `leanport init`

Set up LeanPort in an existing Go project. Refuses to overwrite existing files
unless `--force` is given.

| Flag | Default | Description |
|---|---|---|
| `--env <nix\|local>` | `nix` | `nix` also writes `flake.nix` and `.envrc`; `local` assumes elan/Go/cc are installed. |
| `--lean <vX.Y.Z\|stable>` | `v4.29.1` | Lean toolchain written to `lean/lean-toolchain`. The default is pinned deliberately; `stable` floats. |
| `--targets <list>` | `linux/amd64` | Comma-separated `goos/goarch` pairs. |
| `--ci <github\|none>` | `github` | `github` writes `.github/workflows/verify.yml`. |
| `--name <name>` | directory name | Project name used in `leanport.toml` and the Lake package. |
| `--go-module <path>` | from `go.mod` | Go module path recorded in `leanport.toml`. |
| `--force` | `false` | Overwrite conflicting files. |

Writes: `lean/lean-toolchain`, `lean/lakefile.toml`, `leanport.toml`,
`AGENTS.md`, and (per flags) `flake.nix`, `.envrc`,
`.github/workflows/verify.yml`. It also appends `.leanport/`, `.lake/`,
`.elan/` and `.direnv/` to `.gitignore` if they are missing; generated Go
packages are intentionally not ignored.

Re-running `init` on a configured project fails unless `--force` is given.
`--force` refreshes the scaffold but preserves existing `[[binding]]` entries
and any `[[lean_lib]]` blocks, so it does not delete modules.

### `leanport new <module>`

Scaffold one verified module inside an existing LeanPort project.

Writes:

```
lean/LeanPort/<Module>/Policy.lean      # spec + implementation
lean/LeanPort/<Module>/Proofs.lean      # theorems, with one worked example
lean/LeanPort/<Module>/Boundary.lean    # scalar boundary + @[export]
lean/LeanPort/<Module>/SPEC.md          # human/agent specification
```

and appends a `[[lean_lib]]` block to `lean/lakefile.toml` plus a `[[binding]]`
entry to `leanport.toml`. The scaffolded binding is
`bool, uint8 -> bool`; edit `Boundary.lean` and the binding together.

Running `new` again appends another module; it does not overwrite existing
files. After adding or changing a module, run `leanport check` and
`leanport build` (and `leanport docs` for a refreshed `SPEC.md`).

| Flag | Description |
|---|---|
| `--package <name>` | Generated Go package name. Defaults to the lower-cased module name. |
| `--export <symbol>` | C symbol for the `@[export]` declaration. Defaults to `leanport_<module>_<go-name>`. |
| `--go-name <Name>` | Generated Go function name. Defaults to `Decide`. |

### `leanport check`

Runs `lake build` for each binding's `Policy`, `Proofs` and `Boundary` targets,
then:

- rejects `sorry` and `admit` in Lean code under `lean/`, reporting file and
  line. Line and block comments are stripped first, so documentation may mention
  the words;
- verifies every `[[binding]]`'s `export` appears in the generated C.

Exit `0` on success, `1` on any failure.

### `leanport build`

The full pipeline. Equivalent to `check`, then:

1. locate generated C under `lean/.lake/build/ir/**/*.c`;
2. compile each translation unit with `cc` and the flags from
   `leanc --print-cflags`;
3. subtract symbols defined elsewhere in the project, then inspect the remaining
   undefined symbols;
4. choose a boundary mode (see below) and generate one glue file per binding in
   `.leanport/build/`;
5. compile the glue; in runtime-linked mode merge it with the module objects and
   the Lean runtime (`libc++`, GMP, libuv) into one relocatable object;
6. archive `native/<goos>_<goarch>/lib<Module>.a`;
7. generate `<pkg>.go`, `<pkg>_cgo.go`, `artifact.go` and `MANIFEST.json`;
8. run `go build ./...` to prove the result links (`--verify`, default on).

| Flag | Default | Description |
|---|---|---|
| `--target <goos/goarch>` | host | Build only this target. Must match the host in v0.1.1. |
| `--verify` | `true` | Run `go build ./...` after generating (`--verify=false` to skip). |
| `--clean` | `false` | Remove `.leanport/` build intermediates first. |
| `--runtime <auto\|none\|linked>` | `auto` | Force a boundary mode. `none` fails if the binding needs boxed types. |

Boundary modes:

- **`none`** — runtime-free. Scalars only. The archive contains the module
  objects and a glue object with fail-closed stubs for dead-path runtime
  symbols. Measured 12–28 KB.
- **`linked`** — runtime-linked. Supports `String` and `ByteArray` in addition
  to scalars. The Lean runtime is merged into the archive. Measured ~29 MB.
- **`auto`** — chooses `linked` if any argument/result is `string`/`bytes`, or
  if a live Lean runtime symbol remains after linking the project's own objects;
  otherwise `none`.

`--clean` does **not** delete generated Go packages or archives; it only clears
intermediate objects.

### `leanport docs`

Writes/refreshes `lean/LeanPort/<Module>/SPEC.md` for every module, listing all
of its exports. It is idempotent and never edits `Policy.lean` or `Proofs.lean`.
`AGENTS.md` is written once by `leanport init`; `docs` does not modify it.

### `leanport doctor`

Prints paths and versions for `go`, `cc`, `lean`, `lake`, `nix`, the
`CGO_ENABLED` setting, and — inside a project — the config, toolchain and Lean
prefix. Exits `1` if a required tool (`go`, `cc`, `lean`, `lake`) is missing or
cgo is disabled.

### `leanport version`

Prints `leanport <version> (commit <sha>, built <date>)`. Release binaries are
built with these values injected by `-ldflags`.

## `leanport.toml`

```toml
schema = 1

[project]
name        = "docsvc"                 # informational
lean_dir    = "lean"                   # location of the Lean project
go_module   = "example.com/docsvc"     # read from go.mod on init
package_dir = "internal/verified"      # generated bindings go here
targets     = ["linux/amd64"]          # default build targets

[lean]
toolchain = "4.29.1"                   # mirrors lean/lean-toolchain

[[binding]]
package  = "authz"                     # generated Go package name
module   = "Authz"                     # Lean module under LeanPort/
export   = "leanport_authz_authorize"  # C symbol produced by @[export]
go_name  = "Authorize"                 # Go function name
args     = ["bool", "bool", "bool", "action:Action"]
result   = "bool"

# "auto" (default) chooses none/linked; force with "none" or "linked".
runtime  = "auto"

# Named code tables become Go constant blocks.
actions  = { Create = 0, Read = 1, Update = 2, Delete = 3, ViewAudit = 4 }
```

`actions` is an inline table so it stays attached to its `[[binding]]` entry.
LeanPort's writer emits the equivalent `[binding.actions]` sub-table; both forms
parse.

### Boundary types

The generator in v0.1.1 supports the following tokens:

| Token | Lean | Go | Mode | Generator |
|---|---|---|---|---|
| `bool` | `Bool` | `bool` | runtime-free | yes |
| `uint8/16/32/64` | `UInt8/16/32/64` | `uint8/16/32/64` | runtime-free | yes |
| `int8/16/32/64` | `Int8/16/32/64` | `int8/16/32/64` | runtime-free | yes |
| `usize` | `USize` | `uint` | runtime-free | yes |
| `float` | `Float` | `float64` | runtime-free | yes |
| `float32` | `Float32` | `float32` | runtime-free | yes |
| `rune` | `Char` | `rune` | runtime-free | yes |
| `string` | `String` | `string` | runtime-linked | yes |
| `bytes` | `ByteArray` | `[]byte` | runtime-linked | yes |
| `action:<Enum>` | `UInt8` code | `<Enum>` | runtime-free | yes |
| sealed record | record | `struct` | either | planned |
| `array:<T>` / `list:<T>` | `Array`/`List` | `[]T` | linked | planned |
| `option:<T>` | `Option` | `(T, bool)` | linked | planned |
| `except:<E>:<T>` | `Except` | `(T, error)` | linked | planned |

Records, arrays, `Option` and `Except` have been validated at the ABI level (see
[`architecture.md`](architecture.md)); the code generator does not emit them
yet. Unknown tokens fail `leanport build` with a type error, never with silently
wrong code.

### Exit codes

| Code | Meaning |
|---|---|
| `0` | Success. |
| `1` | Check/build failure, link failure, or missing toolchain. |
| `2` | Usage error (unknown command, bad flag, malformed config type). |

### Environment

| Variable | Purpose |
|---|---|
| `LEANPORT_CC` | Override the C compiler used for module and glue objects. |
| `CGO_ENABLED` | Must not be `0`; generated bindings are cgo packages. |
