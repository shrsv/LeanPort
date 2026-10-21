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

> **Status: pre-alpha.** The interface below is specified and under active
> implementation. The end-to-end pipeline (Lean 4 module + proofs → generated C
> → static archive → cgo → Go binary) has been validated locally, and this
> repository is being filled in against that validated design. Until the
> `v0.1.0` tag is cut, treat the commands and schemas here as the target
> contract, not as a released interface. Follow `CHANGELOG.md` and
> `ROADMAP.md`.

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
| Lean runtime initialization and linking | compiles and archives native artifacts |
| Proof-checking discipline in CI | one `leanport check` command, rejects `sorry` holes |
| Teaching agents where proofs live | generates `docs/agents.md` and per-module specs |
| Version drift across Lean/Go/C toolchains | Nix flake and `doctor` diagnostics |

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
leanport init                     # pin toolchains, set up lean/ and conventions
leanport new authz                # scaffold a module: spec, proofs, boundary, binding
# ... edit lean/LeanPort/Authz/*.lean ...
leanport check                    # run Lean's elaborator; fail on unproved holes
leanport build                    # compile Lean -> C -> static archive + Go binding
go build ./...                    # or: go run ./cmd/app
```

`leanport build` is run by maintainers when Lean code changes. Application
builds and CI for consumers simply run `go build ./...`.

---

## Quickstart

Build a working verified CLI end to end:

➡️ **[`tutorial/`](tutorial/README.md)** — a step-by-step walkthrough that builds
`permctl`, a small command-line tool whose authorization policy is written and
proved in Lean 4, compiled to native code, and called from Go through the
generated binding.

The tutorial covers both a Nix-pinned environment and a non-Nix environment
(`elan` + Go + a C compiler).

---

## Install

> Planned interfaces; releases are published once the tutorial passes its
> end-to-end acceptance run.

**Prebuilt binaries** (linux/darwin/windows × amd64/arm64) will be attached to
each release:

```bash
# example; assets are named leanport_<version>_<os>_<arch>.tar.gz
curl -fsSL https://github.com/shrsv/LeanPort/releases/latest/download/leanport_0.1.0_linux_amd64.tar.gz \
  | tar -xz -C /usr/local/bin leanport
```

**With Go:**

```bash
go install github.com/shrsv/LeanPort/cmd/leanport@latest
```

**From source:**

```bash
git clone https://github.com/shrsv/LeanPort
cd LeanPort
go build ./cmd/leanport
./leanport version
```

Requirements: Go 1.24+, a C compiler (`gcc` or `clang`), and — only for
maintainers changing Lean code — Lean 4 via [elan](https://github.com/leanprover/elan).
`leanport doctor` reports what is present and what is missing.

---

## Usage

```
LeanPort — bring Lean 4 proven modules into your Go projects

Usage:
  leanport <command> [flags]

Commands:
  init      Set up LeanPort in an existing Go project (pins toolchains)
  new       Scaffold a new verified module (spec, proofs, boundary, binding)
  check     Type-check proofs and reject unproved holes
  build     Compile Lean to C, archive it, and generate the Go binding
  docs      Generate module specs and agent-facing documentation
  doctor    Diagnose missing toolchains and version mismatches
  version   Print version information

Flags:
  -C, --dir <path>   Run as if LeanPort was started in <path>
  -h, --help         Help for leanport
      --version      Print version information
```

Full command and configuration reference: [`docs/cli-reference.md`](docs/cli-reference.md).

### Generated Go API

For a binding declared in `leanport.toml`, LeanPort emits a small, boring
package:

```go
package authz

type Action uint8

const (
    Create Action = iota
    Read
    Update
    Delete
    ViewAudit
)

// Authorize reports whether an authenticated principal may perform action.
func Authorize(authenticated, isAdmin, ownsResource bool, action Action) bool
```

The generated package also contains the cgo layer, the C bridge header, and
`native/<goos>_<goarch>/lib<module>.a`. Application code imports only the Go
API. It never sees `lean_object`, Lean initialization functions, or raw C
symbols.

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
| **bridge** | The generated C layer that adapts Lean's C ABI to a stable, narrow interface. |
| **binding** | The generated Go package the application imports. |
| **target** | A `goos/goarch` pair that a native archive is built for. |
| **consumer** | An application that imports a generated binding and never needs Lean. |
| **maintainer** | Someone who edits Lean code and must run `leanport check` and `leanport build`. |

---

## Limitations

State clearly, up front:

- **Scalar boundary only (v0.1).** Exported functions may take and return
  `Bool`, `UInt8/16/32/64`, `Int8/16/32/64` and must be total and fail-closed.
  Strings, arrays, optionals, inductive returns and IO in an exported path are
  rejected at `leanport build` time with a diagnostic. This keeps the generated
  bridge free of the Lean runtime.
- **Lean's C ABI is unstable.** LeanPort records the Lean toolchain version in
  the module manifest and regenerates artifacts on toolchain upgrades.
- **cgo is required.** A C compiler and compatible native libraries must exist
  at application build time. `CGO_ENABLED=0` pure-Go builds cannot link a
  generated binding.
- **Cross-compilation needs a target C toolchain**, and native archives are
  per-target. A linux/amd64 archive does not work on darwin/arm64.
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
- Each module's `SPEC.md` (generated by `leanport docs`) states exactly what is
  guaranteed.
- `leanport check` failures name the unsatisfied obligation.

---

## Repository layout

```
LeanPort/
├── cmd/leanport/          CLI entry point
├── internal/              CLI implementation
├── tutorial/              build a sample verified application, end to end
│   └── permctl/           verified-authorization command-line tool
├── docs/                  architecture, CLI reference, limitations, agent guide
├── scripts/e2e.sh         end-to-end acceptance run
├── flake.nix              pinned Lean + Go + C toolchain
├── ROADMAP.md             what is in v0.1 and what is deferred
└── .github/workflows/     verify + release automation
```

## Roadmap and license

See [`ROADMAP.md`](ROADMAP.md). MIT licensed — see [`LICENSE`](LICENSE).
