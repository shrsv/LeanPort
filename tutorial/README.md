# Tutorial: build a verified CLI with LeanPort

This tutorial builds **`permctl`**, a small command-line tool whose
authorization policy is written and proved in Lean 4, compiled to native code,
and called from Go through a generated LeanPort binding.

The point is not `permctl` itself. The point is the workflow: *write rules and
proofs in Lean, get an ordinary Go package, ship a Go binary.*

> **Status: pre-alpha outline.** The steps and expected results below are the
> acceptance script for `v0.1.0`. Each *Expected* block is filled in from a real
> run during implementation; until then, treat them as the target.

---

## What you are building

`permctl` answers authorization questions against a policy with four access
levels and five actions.

| Access level | create | read | update | delete | viewAudit |
|---|---|---|---|---|---|
| unauthenticated | deny | deny | deny | deny | deny |
| member | **allow** | deny | deny | deny | deny |
| owner (of this resource) | n/a | **allow** | **allow** | **allow** | deny |
| admin | **allow** | **allow** | **allow** | **allow** | **allow** |

Crucially, `owner` is a property of *this resource*, not an account-wide role.
The policy function therefore receives three facts — authenticated, admin,
owns-this-resource — not a single role string.

The Lean project proves properties such as:

- an unauthenticated caller is denied every action;
- an unauthenticated caller cannot become an admin by any flag combination;
- an owner cannot read the audit log;
- audit access is exactly equivalent to "authenticated **and** admin".

## Prerequisites

| Tool | Needed for | Check |
|---|---|---|
| Go 1.24+ | the application and the CLI | `go version` |
| a C compiler | cgo | `cc --version` |
| Lean 4 via elan | changing/proving Lean code | `lean --version` |
| Nix *(optional but recommended)* | pinned toolchains | `nix --version` |
| LeanPort | this workflow | `leanport version` |

## Path A — with Nix (recommended)

```bash
cd tutorial/permctl
nix develop
```

*Expected:* a shell with pinned Lean, Go and a C compiler; `lake --version`,
`go version` and `cc --version` all report the versions in `flake.nix`.

## Path B — without Nix

```bash
# Lean via elan
curl -fsSL https://elan.lean-lang.org/elan-init.sh | sh -s -- -y
export PATH="$HOME/.elan/bin:$PATH"
# Go and a C compiler must already be on PATH
```

The tracked `lean-toolchain` file makes elan select the pinned Lean version
automatically on first use.

---

## Steps

### 1. Create the project

```bash
mkdir permctl && cd permctl
go mod init example.com/permctl
leanport init --env local --targets linux/amd64 --ci none
```

*Expected:* `lean/`, `leanport.toml` and `AGENTS.md` are created; `leanport
doctor` reports Go, a C compiler and Lean as present.

### 2. Scaffold the verified module

```bash
leanport new authz
```

*Expected:* `lean/LeanPort/Authz/{Policy.lean,Proofs.lean,Boundary.lean,SPEC.md}`
exist, and `leanport.toml` has a new `[[binding]]` entry.

### 3. Read, then edit the Lean policy

Open `lean/LeanPort/Authz/Policy.lean`. It defines the level/action model and
the `authorize` function. `Proofs.lean` states the guarantees.

```bash
sed -n '1,60p' lean/LeanPort/Authz/Policy.lean
```

*Expected:* the policy is readable without any Lean build-system knowledge.

### 4. Check the proofs

```bash
leanport check
```

*Expected:* Lean elaborates `Policy.lean` and `Proofs.lean`; the command exits
`0`. Then, deliberately:

```bash
echo 'theorem broken : False := by sorry' >> lean/LeanPort/Authz/Proofs.lean
leanport check   # must fail
```

*Expected:* a non-zero exit naming `Proofs.lean` and the `sorry`. Remove the
line before continuing. This is the discipline the tool exists to enforce.

### 5. Build the native binding

```bash
leanport build
```

*Expected:* `.lake` C is generated, compiled and archived to
`internal/verified/authz/native/linux_amd64/libAuthz.a`; the Go API,
`authz_cgo.go`, `bridge.h`, `bridge.c` and `MANIFEST.json` appear; a final
`go build ./...` succeeds.

### 6. Inspect what was generated

```bash
ls internal/verified/authz
cat internal/verified/authz/authz.go
```

*Expected:* a small Go API (`type Action uint8`, `func Authorize(...) bool`) and
no Lean types, no `lean_object`, no manual C.

### 7. Write the CLI

`cmd/permctl/main.go` is stdlib-only and has three useful modes:

```bash
permctl decide --auth --admin=false --owner=false --action create   # -> allow
permctl matrix                                                      # -> the table above
permctl verify                                                      # -> all cells asserted
```

*Expected:* `decide` prints `allow`/`deny` and exits `0`/`1`; `verify` checks all
twenty `(level, action)` cells against the table and exits non-zero on any
mismatch.

### 8. Prove it through usage

```bash
go test ./...
go run ./cmd/permctl verify
go run ./cmd/permctl decide --auth --owner --action delete    # -> allow
go run ./cmd/permctl decide --auth --action viewAudit         # -> deny
```

*Expected:* tests pass; the CLI's answers match the table. This is the check
that the Go wiring is correct — distinct from the Lean proofs, which are about
the policy function itself.

### 9. What was proved, and what was not

- **Proved in Lean:** the decision function's behaviour for every modelled input.
- **Not proved:** that the CLI parses flags correctly, maps flags to the right
  facts, or exits with the intended code. That is what the Go tests are for.

---

## CI

`leanport init --ci github` writes a workflow that runs the same steps on every
push, so a policy change cannot merge without its proof and its binding
regenerating cleanly.

## Next

Read [`docs/agents.md`](../docs/agents.md) for how agents should work in this
project, and [`docs/limitations.md`](../docs/limitations.md) before widening the
boundary beyond scalars.
