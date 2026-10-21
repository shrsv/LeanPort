# CLI reference

> Status: pre-alpha. This is the target contract for `v0.1.0`.

## Synopsis

```
leanport <command> [flags]
```

Run `leanport <command> --help` for command-specific help.

## Global flags

| Flag | Description |
|---|---|
| `-C, --dir <path>` | Run as if LeanPort was started in `<path>`. |
| `-h, --help` | Show help. |
| `--version` | Print version and exit. |
| `--verbose` | Print each external command (`lake`, `cc`, `ar`) and its arguments. |

## Commands

### `leanport init`

Set up LeanPort in an existing Go project. Non-destructive by default: refuses
to overwrite files it did not generate unless `--force` is given.

| Flag | Default | Description |
|---|---|---|
| `--env <nix\|local>` | `nix` | `nix` writes `flake.nix`, `.envrc`, `flake.lock` policy; `local` documents elan/Go/cc pinning instead. |
| `--lean <stable\|vX.Y.Z>` | `stable` | Lean toolchain written to `lean/lean-toolchain`. |
| `--go <detect\|X.Y>` | `detect` | Reads `go.mod`; `detect` preserves it. |
| `--targets <list>` | `linux/amd64` | Comma-separated `goos/goarch` pairs to build archives for. |
| `--ci <github\|none>` | `github` | Writes `.github/workflows/verify.yml`. |
| `--force` | `false` | Overwrite conflicting files. |

Writes: `lean/lean-toolchain`, `lean/lakefile.toml`, `leanport.toml`,
`AGENTS.md`, and (per flags) `flake.nix`, `.envrc`,
`.github/workflows/verify.yml`.

### `leanport new <module>`

Scaffold one verified module inside an existing LeanPort project.

Writes:

```
lean/LeanPort/<Module>/Policy.lean      # spec + implementation
lean/LeanPort/<Module>/Proofs.lean      # theorems, with one worked example
lean/LeanPort/<Module>/Boundary.lean    # runtime-free scalar boundary + @[export]
lean/LeanPort/<Module>/SPEC.md          # human/agent specification
```

and appends a `[[binding]]` entry to `leanport.toml`.

| Flag | Description |
|---|---|
| `--export <symbol>` | C symbol for the `@[export]` declaration. Defaults to `leanport_<module>_<go_name>`. |
| `--go-package <name>` | Generated Go package name. Defaults to the lower-cased module name. |

### `leanport check`

Runs `lake build` for every Lean library, then:

- rejects `sorry` and `admit` anywhere under `lean/`, reporting file and line;
- verifies every `[[binding]]`'s `export` appears in the generated C.

Exit `0` on success, `1` on any failure. Output is designed to be read by both
humans and agents: it names the unmet obligation.

### `leanport build`

The full pipeline. Equivalent to `check`, then:

1. locate generated C under `lean/.lake/build/ir/**/*.c`;
2. compile each translation unit with the C compiler and flags from
   `leanc --print-cflags`;
3. inspect undefined symbols: if only the known dead-path runtime entry points
   remain, emit fail-closed stubs (runtime-free); otherwise merge the Lean
   runtime, libc++, GMP and libuv into the archive (runtime-linked);
4. emit `bridge.c` with fail-closed stubs for known dead-path runtime symbols;
5. archive `native/<goos>_<goarch>/lib<Module>.a`;
6. generate the Go API, `_cgo.go`, `bridge.h` and `MANIFEST.json`;
7. optionally run `go build ./...` to prove the result links (`--verify`, default on).

| Flag | Default | Description |
|---|---|---|
| `--target <goos/goarch>` | from config | Build only this target. |
| `--verify` | `true` | Run `go build ./...` after generating. |
| `--clean` | `false` | Remove previous generated artifacts first. |

### `leanport docs`

Generates `SPEC.md` per module and refreshes the generated block in `AGENTS.md`:
signatures, error codes, preconditions, and a list of files not to edit.

### `leanport doctor`

Prints a report of `go`, `cc`, `lake`, `lean`, `elan`, `nix`, `CGO_ENABLED`,
detected `go.mod`, `lean-toolchain` and pinned config versions, plus which
targets can currently be built. Non-zero exit if a required tool is missing.

### `leanport version`

Prints version, commit and build date. `--json` for machine consumption.

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
toolchain = "leanprover/lean4:v4.29.1" # mirrors lean/lean-toolchain

[[binding]]
package  = "authz"                     # generated Go package name
module   = "Authz"                     # Lean module under LeanPort/
export   = "leanport_authz_authorize"  # C symbol produced by @[export]
go_name  = "Authorize"                 # Go function name

# Argument and result types map to Go through this table:
#   bool           <-> Bool            (runtime-free)
#   uint8/16/32/64 <-> UInt8/16/32/64  (runtime-free)
#   int8/16/32/64  <-> Int8/16/32/64   (runtime-free)
#   usize          <-> USize           (runtime-free)
#   float/float32  <-> Float/Float32   (runtime-free)
#   rune           <-> Char            (runtime-free)
#   string         <-> String          (runtime-linked, copied UTF-8)
#   bytes          <-> ByteArray       (runtime-linked, copied)
#   <name>:<Enum>  <-> UInt8 code declared in `actions`/`enums`
#   <Name>         <-> sealed record   (fields mapped recursively)
args     = ["bool", "bool", "bool", "action:Action"]
result   = "bool"

# Boundary mode. "auto" (default) chooses runtime-free when every export is
# scalar-only, and runtime-linked when boxed data is involved. Force with
# "none" or "linked".
runtime  = "auto"

# Named code tables become Go constant blocks and fail-closed decoders.
[bindings.actions]
Create    = 0
Read      = 1
Update    = 2
Delete    = 3
ViewAudit = 4
```

### Exit codes

| Code | Meaning |
|---|---|
| `0` | Success. |
| `1` | Check/build failure (proof hole, missing export, link failure). |
| `2` | Usage error or missing toolchain. |

### Environment

| Variable | Purpose |
|---|---|
| `LEANPORT_LEAN` | Override the `lean` executable path. |
| `LEANPORT_CC` | Override the C compiler. |
| `LEANPORT_NO_VERIFY` | Skip the post-build `go build ./...` check. |
