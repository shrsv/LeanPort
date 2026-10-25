# LeanPort

**Bring Lean 4 proven modules into your Go projects.**

LeanPort is a command-line tool that turns a folder of Lean 4 modules into a
cgo-consumable Go package. You write the logic and its proofs in Lean; LeanPort
checks the proofs, compiles the Lean implementation to C, packages a native
static archive, and generates the C/cgo bridge plus an idiomatic Go API.

The generated package behaves like any other Go dependency:

```go
import "example.com/app/internal/verified/authz"

allowed := authz.Authorize(authenticated, isAdmin, ownsResource, authz.Create)
```

Consumers need Go, cgo and a C toolchain. They do **not** need Lean, Lake, elan
or the LeanPort CLI to build the application.

> **Status: v0.1.1.** The pipeline is implemented and exercised end to end by
> [`scripts/e2e.sh`](scripts/e2e.sh): fresh scaffold, proof checking, runtime-free
> and runtime-linked archives, Go tests, and a deliberate `sorry` failure. The
> generator supports scalars, strings and byte arrays. Records, arrays,
> `Option` and `Except` have validated ABI mappings but no generator yet; see
> [`ROADMAP.md`](ROADMAP.md) and [`CHANGELOG.md`](CHANGELOG.md).

---

## Start here: examples, in order

The fastest way to understand LeanPort is to work through six small examples.
Each teaches one Lean idea, shows the LeanPort boundary that exposes it to Go,
and ends with an ordinary Go call:

| # | What it teaches |
|---|---|
| 1 | [Executable Lean, called from Go](tutorial/01-executable.md) |
| 2 | [Pure functions and recursion](tutorial/02-functions.md) |
| 3 | [Specifications and proofs](tutorial/03-specifications.md) |
| 4 | [A business invariant](tutorial/04-business-invariant.md) |
| 5 | [Types that enforce invariants](tutorial/05-refined-types.md) |
| 6 | [Capstone: a verified authorization CLI](tutorial/06-authorization.md) |

➡️ **[Open the example index](tutorial/README.md)** for the learning path,
what each example teaches, and how to run them.

If you only run one thing, run the capstone, **`permctl`**: a small command-line
tool whose authorization policy is written and proved in Lean 4, compiled to
native code, and called from Go through a generated binding. If LeanPort is not
installed, jump to [Install](#install) and come back. `leanport check` and
`leanport build` need Lean; the rest need only Go and cgo:

```bash
git clone https://github.com/shrsv/LeanPort
cd LeanPort/tutorial/permctl
leanport check
leanport build
go test ./...
go build -o permctl ./cmd/permctl
./permctl verify
./permctl matrix
```

Key output on Lean 4.29.1, Go 1.24.3, linux/amd64 (timings vary):

```text
✓ checked 1 module(s); 1 export(s) verified
...
→ internal/verified/authz/native/linux_amd64/libAuthz.a (none mode, 27.4 KB)
→ go build ./...
✓ build complete
ok  	example.com/permctl/cmd/permctl	0.004s
verified 39 checks
LEVEL            create     read       update     delete     viewAudit
unauthenticated  deny       deny       deny       deny       deny
member           allow      deny       deny       deny       deny
owner            deny       allow      allow      allow      deny
admin            allow      allow      allow      allow      allow
```

The checked-in project includes the generated archive, so `go test` and
`go build` need only Go and cgo. `leanport check` and `leanport build` need Lean
because they run the prover and compile the native code.

Decide individual requests:

```bash
./permctl decide --auth --owner --action update; echo "exit=$?"
./permctl decide --auth --owner --action viewAudit; echo "exit=$?"
```

```text
allow
exit=0
deny
exit=1
```

That is the whole loop: **Lean owns the rules and proofs; `leanport build`
produces a native archive and Go API; `go build` stays an ordinary Go build.**
The examples also show how to start from scratch, how to make a broken proof
fail CI, and how to inspect the generated package.

A separate example shows `String` and `ByteArray` crossing the boundary:
[`examples/rich-types/`](examples/rich-types/).

---

## Install

**Prebuilt binaries** (linux/darwin/windows × amd64/arm64) are attached to each
release together with `checksums.txt`:

```bash
VERSION=v0.1.1
curl -fsSL -o leanport \
  "https://github.com/shrsv/LeanPort/releases/download/${VERSION}/leanport_${VERSION}_linux_amd64"
chmod +x leanport
sudo install -m755 leanport /usr/local/bin/leanport
```

Choose `darwin_arm64`, `windows_amd64.exe`, and so on for your platform.

**With Go:**

```bash
go install github.com/shrsv/LeanPort/cmd/leanport@v0.1.1
```

**From source:**

```bash
git clone https://github.com/shrsv/LeanPort
cd LeanPort
go build ./cmd/leanport
./leanport version
```

> A binary installed with `go install` or built with `go build` from source
> prints `leanport dev`; the release script injects the version. That is
> expected.

Requirements: Go 1.24+, a C compiler (`gcc` or `clang`), and — only for
maintainers changing Lean code — Lean 4 via [elan](https://github.com/leanprover/elan).
`leanport doctor` reports what is present and what is missing.

---

## Motivation

Formal verification is valuable exactly where Go code is hardest to test: pure
decision logic with many boundary cases, such as authorization policies, quota
and billing rules, state machines, and protocol invariants. But bringing Lean
into a Go project usually means learning Lake, Lean's native toolchain, the
(undocumented and unstable) Lean FFI, C symbol mangling, runtime initialization,
and platform-specific linking. That is enough friction that most teams never
start.

LeanPort's principle is simple:

> **`leanport build` is allowed to be complicated. `go build` must be boring.**

LeanPort owns the tedious integration machinery:

| Painful without LeanPort | LeanPort does it |
|---|---|
| Learning Lake's target model | scaffolds `lakefile.toml` and module layout |
| Writing C ABI shims by hand | generates the boundary bridge and Go wrapper |
| Lean runtime initialization and linking | compiles, initializes and archives native artifacts |
| Proof-checking discipline in CI | one `leanport check` command, rejects `sorry` holes |
| Teaching agents where proofs live | writes `AGENTS.md` and per-module `SPEC.md` |
| Version drift across Lean/Go/C toolchains | Nix flake, pinned toolchain, `doctor` diagnostics |

**Non-goal:** LeanPort does not translate Lean source into Go source. Lean
compiles the implementation to native code. LeanPort generates the C ABI
adapter and the Go/cgo wrapper around it. The Lean implementation remains the
single source of truth; generated files are never a second implementation.

---

## How it works

```
        you write                        leanport build                go build
┌────────────────────────┐   ┌───────────────────────────────┐   ┌──────────────────┐
│ lean/LeanPort/Authz/   │   │ 1. leanport check (proofs)    │   │ your application │
│   Policy.lean          │   │ 2. lake build  ──► Authz.c    │   │                  │
│   Proofs.lean          │──►│ 3. cc -c       ──► Authz.o    │──►│  imports         │
│   Boundary.lean        │   │ 4. ar rcs      ──► libAuthz.a │   │  internal/       │
│   (@[export] entry)    │   │ 5. generate Go + cgo bridge   │   │  verified/authz  │
└────────────────────────┘   └───────────────────────────────┘   └──────────────────┘
      source of truth                generated, committed            ordinary import
```

Commands, in order:

```bash
leanport init --env local --ci none --name myapp --targets linux/amd64
leanport new --package authz --go-name Authorize \
  --export leanport_authz_authorize Authz
# ... edit lean/LeanPort/Authz/*.lean and the [[binding]] in leanport.toml ...
leanport check                    # run Lean's elaborator; fail on unproved holes
leanport build                    # compile Lean -> C -> static archive + Go binding
go build ./...                    # or: go run ./cmd/app
```

`leanport build` is run by maintainers when Lean code changes. Application
builds and CI for consumers simply run `go build ./...`.

### Project layout

The separation is the whole idea: `lean/` is yours, `internal/verified/` is
generated, and `cmd/` is ordinary application code that never touches C.

```
myapp/
├── go.mod
├── leanport.toml                     # the contract: modules, exports, types, targets
├── lean/
│   ├── lean-toolchain                # pinned Lean 4 version
│   ├── lakefile.toml
│   └── LeanPort/
│       └── Authz/
│           ├── Policy.lean           # specification + executable implementation
│           ├── Proofs.lean           # machine-checked theorems about the policy
│           ├── Boundary.lean         # decode C scalars, call the policy, encode the result
│           └── SPEC.md               # generated: what is guaranteed
├── internal/
│   └── verified/
│       └── authz/                    # GENERATED by `leanport build` — do not edit
│           ├── authz.go              # public Go API: type Action, func Authorize
│           ├── authz_cgo.go          # cgo layer, C prototypes and Go wrappers
│           ├── artifact.go           # archive digest (invalidates Go's build cache)
│           ├── native/
│           │   └── linux_amd64/
│           │       └── libAuthz.a    # compiled Lean implementation + glue
│           └── MANIFEST.json         # Lean toolchain, export table, digests, mode
└── cmd/
    └── app/                          # your application: imports internal/verified/authz
```

The `lean.h`-dependent glue is compiled at `leanport build` time and lives
inside the archive. It is not a source file in the generated Go package, so
`go build` never needs Lean headers.

A `[[binding]]` entry in `leanport.toml` ties the two sides together:

```toml
[[binding]]
package  = "authz"                       # generated Go package
module   = "Authz"                       # Lean module under lean/LeanPort/
export   = "leanport_authz_authorize"    # C symbol from @[export]
go_name  = "Authorize"
args     = ["bool", "bool", "bool", "action:Action"]
result   = "bool"
runtime  = "auto"
actions  = { Create = 0, Read = 1, Update = 2, Delete = 3, ViewAudit = 4 }
```

---

## Adding another module

`authz` is just the first module. Adding another one is a command, not a
convention: `leanport new` writes the Lean scaffold, appends the Lake library,
and appends the binding. Run it from the project root:

```bash
leanport new --package quota --go-name AllowQuota \
  --export leanport_quota_allow Quota
```

```text
created lean/LeanPort/Quota/{Policy,Proofs,Boundary}.lean
added binding quota.AllowQuota -> leanport_quota_allow

Edit the Lean files, then: leanport check && leanport build
```

The generated scaffold is already total. Its default binding has
`args = ["bool", "uint8"]` and `result = "bool"`, so the generated Go API is:

```go
func AllowQuota(enabled bool, code uint8) bool
```

Replace the scaffold's `decide` and theorems with your own rules, and adjust
`args`/`result` in the new `[[binding]]` if needed. `--package` is the generated
Go package directory, `--go-name` is the Go function, `--export` must match the
`@[export]` name in `Boundary.lean`, and the last argument (`Quota`) is the Lean
module.

Then run both commands:

```bash
leanport check
leanport build
```

`check` verifies every binding in the project. `build` regenerates **all**
bindings, not just the new one, and writes `internal/verified/quota/` beside
`internal/verified/authz/`. Import the new package from ordinary Go code:

```go
import (
	"example.com/permctl/internal/verified/authz"
	"example.com/permctl/internal/verified/quota"
)

fmt.Println(authz.Authorize(true, false, true, authz.Update))
fmt.Println(quota.AllowQuota(true, 7))
```

There is no watcher and no automatic update. Editing Lean and running
`go build` does nothing: `go build` only compiles the already-generated package.
After any change to `lean/` or `leanport.toml`, rerun
`leanport check && leanport build`. Run `leanport docs` if you also want the
generated `SPEC.md` refreshed. Commit the Lean sources, `leanport.toml`, and the
regenerated `internal/verified/` tree.

Multiple runtime-free modules can be linked into one binary. Runtime-linked
packages (`String`/`ByteArray`) embed Lean's runtime in each archive, so keep
those exports in one package; multiple runtime-linked packages are not
supported in v0.1.1. Adding a module by hand is possible, but you must edit both
`lean/lakefile.toml` and `leanport.toml`; `leanport new` is the supported path.

---

## Usage

```
LeanPort - bring Lean 4 proven modules into your Go projects

Usage:
  leanport <command> [flags]

Commands:
  init      Set up LeanPort in an existing Go project (pins toolchains)
  new       Scaffold a new verified module (spec, proofs, boundary, binding)
  check     Type-check proofs and reject unproved holes
  build     Compile Lean to C, archive it, and generate the Go binding
  docs      Generate per-module specification files
  doctor    Diagnose missing toolchains and version mismatches
  version   Print version information

Global flags:
  -C, --dir <path>   Run as if LeanPort was started in <path>
  -h, --help         Show help
      --version      Print version
```

Full command and configuration reference: [`docs/cli-reference.md`](docs/cli-reference.md).

### Generated Go API

For a binding declared in `leanport.toml`, LeanPort emits a small, boring
package:

```go
package authz

type Action uint8

const (
	Create    Action = 0
	Read      Action = 1
	Update    Action = 2
	Delete    Action = 3
	ViewAudit Action = 4
)

// Authorize reports whether an authenticated principal may perform action.
func Authorize(authenticated, isAdmin, ownsResource bool, action Action) bool
```

The generated package also contains the cgo layer, an archive digest file, the
native archive and `MANIFEST.json`. Application code imports only the Go API. It
never sees `lean_object`, Lean initialization functions, or raw C symbols.

### Supported types

`leanport build` inspects the compiled objects and chooses the cheapest mode
that can carry each binding.

| Lean | Go | Mode | Generator |
|---|---|---|---|
| `Bool` | `bool` | runtime-free | yes |
| `UInt8/16/32/64` | `uint8/16/32/64` | runtime-free | yes |
| `Int8/16/32/64` | `int8/16/32/64` | runtime-free | yes |
| `USize` | `uint` | runtime-free | yes |
| `Float`, `Float32` | `float64`, `float32` | runtime-free | yes |
| `Char` | `rune` | runtime-free | yes |
| `UInt8` action code | generated enum type | runtime-free | yes |
| `String` | `string` | runtime-linked | yes |
| `ByteArray` | `[]byte` | runtime-linked | yes |
| sealed record | Go `struct` | either | planned |
| `Array α`, `List α` | `[]α` | runtime-linked | planned |
| `Option α` | `(α, ok bool)` | runtime-linked | planned |
| `Except ε α` | `(α, error)` | runtime-linked | planned |

*Runtime-free* means the archive contains the verified module's own objects plus
a small glue object with fail-closed stubs; the Lean runtime is not linked at
all. *Runtime-linked* means Lean's runtime is merged into the archive
statically, so the consumer still needs nothing but a C toolchain.

Measured on Lean 4.29.1, linux/amd64: the runtime-free capstone archive is about
27 KB and its Go binary about 2.3 MB. The runtime-linked rich-types archive is
about 29 MB and its demo binary about 14 MB. `ldd` shows only ordinary system
libraries in both cases.

Everything on the boundary is **total and fail-closed**: an unrecognised code or
an invalid input yields the deny/zero result, never a panic.

---

## Vocabulary

LeanPort uses a deliberately small vocabulary.

| Term | Meaning |
|---|---|
| **module** | One verified unit: its specification, Lean implementation, proofs and boundary. Lives under `lean/`. |
| **specification** | What the operation must guarantee: preconditions, postconditions, invariants, and failure behaviour. |
| **implementation** | The executable Lean definition. The source of truth. |
| **proofs** | Machine-checked theorems about the implementation. `leanport check` fails if required guarantees are unproved. |
| **boundary** | The small, total function that decodes C-friendly scalars into domain values and encodes a result. Fails closed on invalid input. |
| **bridge** | The generated C layer that adapts Lean's C ABI to a stable, narrow interface. Compiled into the archive. |
| **binding** | The generated Go package the application imports. |
| **target** | A `goos/goarch` pair that a native archive is built for. |
| **consumer** | An application that imports a generated binding and never needs Lean. |
| **maintainer** | Someone who edits Lean code and must run `leanport check` and `leanport build`. |

---

## Limitations

State clearly, up front:

- **Two boundary modes, selected automatically.** A scalar-only export compiles
  to a runtime-free archive (12–28 KB) with no Lean runtime in your binary. An
  export that uses `String` or `ByteArray` switches to a runtime-linked archive
  (~29 MB) that embeds Lean's runtime statically. Either way, consumers need no
  Lean installation and no Lean headers. The tradeoff is size, not consumer
  setup.
- **The generator covers scalars, strings and byte arrays.** Records, arrays,
  `Option` and `Except` are ABI-validated but not generated yet; a build using
  them fails with a clear type error rather than emitting wrong code.
- **Lean's C ABI is unstable.** LeanPort records the Lean toolchain version in
  the manifest and regenerates artifacts on toolchain upgrades.
- **cgo is required.** A C compiler and compatible native libraries must exist
  at application build time. `CGO_ENABLED=0` pure-Go builds cannot link a
  generated binding.
- **Multiple modules are supported, with one runtime-linked package limitation.**
  Runtime-free modules can be imported together. Each runtime-linked package
  embeds Lean's runtime, so keep `String`/`ByteArray` exports in one package in
  v0.1.1.
- **v0.1.1 builds the host target.** Cross-compilation needs a target C
  toolchain; archives are per-target.
- **Linked glue does not add locking.** Serialize calls if Go calls a
  runtime-linked binding from multiple goroutines.
- **LeanPort proves the decision rules, not the whole application.** Unless you
  model them, authentication, resource lookup, route-to-action mapping,
  database consistency and deployment remain outside the verified boundary.

Details: [`docs/limitations.md`](docs/limitations.md).

---

## For AI agents

LeanPort is designed so that coding agents can find the contract, make verified
changes, and interpret proof failures without improvising the build.

- [`docs/agents.md`](docs/agents.md) — where implementation, proofs and
  generated files live; which files must never be edited by hand; how to run
  checks and read failures.
- Each module's `SPEC.md` (generated by `leanport docs`) states what is
  guaranteed.
- `leanport check` failures name the file and the unsatisfied obligation.

---

## Repository layout

```
LeanPort/
├── cmd/leanport/          CLI entry point
├── internal/app/          CLI implementation
├── tutorial/              six examples, from executable Lean to a verified CLI
│   ├── foundations/       Examples 1-5: one small shared project
│   └── permctl/           Example 6: verified-authorization command-line tool
├── examples/
│   └── rich-types/        String/ByteArray boundary, runtime-linked archive
├── docs/                  architecture, CLI reference, limitations, agent guide
├── media/                 README images (LiveReview banner)
├── scripts/
│   ├── e2e.sh             end-to-end acceptance run
│   └── release.sh         six cross-built binaries + checksums + gh release
├── flake.nix              pinned Go + C toolchain (Lean via elan)
├── .goreleaser.yaml       release configuration for future tags
├── ROADMAP.md             what is in v0.1 and what is deferred
└── .github/workflows/     verify + release automation
```

## Resources

- [Introducing LeanPort](https://journal.hexmos.com/introducing-leanport/) —
  the announcement article: motivation, design and an end-to-end walkthrough.

## Roadmap and license

See [`ROADMAP.md`](ROADMAP.md). MIT licensed — see [`LICENSE`](LICENSE).

---

## See More

Your team's attention is limited. Spend review effort where **business risk is highest** — not spread evenly across every diff.

If LeanPort is about proving the rules that ship, [**LiveReview**](https://hexmos.com/livereview) does the analogous thing for your *code changes*: instead of reviewing every diff with equal effort, it scores each change by blast radius — how far its impact reaches through your call graph — so review attention goes where it actually matters.

[![LiveReview — Risk-Aware AI Code Review for Business-Critical Systems](media/livereview-banner.png)](https://hexmos.com/livereview)
