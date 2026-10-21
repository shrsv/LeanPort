# Tutorial: build a verified CLI with LeanPort

This tutorial builds **`permctl`**, a small command-line tool whose
authorization policy is written and proved in Lean 4, compiled to native code,
and called from Go through a generated LeanPort binding.

The point is not `permctl` itself. The point is the workflow: *write rules and
proofs in Lean, get an ordinary Go package, ship a Go binary.*

> **Status: complete.** The finished project is checked into
> [`permctl/`](permctl/), and [`../scripts/e2e.sh`](../scripts/e2e.sh) rebuilds
> and tests it on every run. Every *Expected* block below comes from that run
> (Lean 4.29.1, Go 1.24, linux/amd64).

---

## What you are building

`permctl` answers authorization questions against a policy with four access
levels and five actions.

| Access level | create | read | update | delete | viewAudit |
|---|---|---|---|---|---|
| unauthenticated | deny | deny | deny | deny | deny |
| member | **allow** | deny | deny | deny | deny |
| owner (of this resource) | deny | **allow** | **allow** | **allow** | deny |
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
| Nix *(optional)* | pinned Go + C toolchain | `nix --version` |
| LeanPort | this workflow | `leanport version` |

Install LeanPort from a release, or build it from this repository:

```bash
cd LeanPort
go build -o /usr/local/bin/leanport ./cmd/leanport   # or any directory on PATH
leanport version
```

## Path A — use the checked-in project

```bash
cd LeanPort/tutorial/permctl
leanport check
leanport build
go test ./...
go run ./cmd/permctl verify
```

*Expected:* the proof check succeeds, the archive is runtime-free (about 27 KB),
tests pass, and `verify` prints `verified 39 checks`. Jump to
[Run the CLI](#run-the-cli) to try it.

## Path B — with Nix (pinned environment)

The flake lives at the repository root, not inside the tutorial project:

```bash
cd LeanPort
nix develop
cd tutorial/permctl
```

The shell provides Go, gcc and helpers, and bootstraps elan into `.elan/` if
Lean is not already installed. The first entry downloads the Lean toolchain
pinned by `lean/lean-toolchain`.

*Expected:* `go version`, `cc --version` and `lean --version` succeed inside the
shell. `flake.lock` is created on first use; commit it if you want to pin
nixpkgs.

## Path C — without Nix

```bash
# Lean via elan
curl -fsSL https://elan.lean-lang.org/elan-init.sh | sh -s -- -y
export PATH="$HOME/.elan/bin:$PATH"
# Go and a C compiler must already be on PATH
```

The tracked `lean-toolchain` file makes elan select the pinned Lean version
automatically on first use.

---

## Build it from scratch

### 1. Create the project

```bash
mkdir permctl && cd permctl
go mod init example.com/permctl
leanport init --env local --ci none --name permctl --targets linux/amd64
```

*Expected:* `lean/`, `leanport.toml` and `AGENTS.md` are created; `leanport
doctor` reports Go, a C compiler and Lean as present.

### 2. Scaffold the verified module

```bash
leanport new --package authz --go-name Authorize \
  --export leanport_authz_authorize Authz
```

Flags must come **before** the module name. *Expected:*
`lean/LeanPort/Authz/{Policy.lean,Proofs.lean,Boundary.lean,SPEC.md}` exist,
`lean/lakefile.toml` has a `[[lean_lib]]` block for `LeanPort.Authz`, and
`leanport.toml` has a new `[[binding]]` entry.

### 3. Write the policy

Replace `lean/LeanPort/Authz/Policy.lean` with the model below. `Action` is a
small inductive; the boundary decodes wire codes into it. Nothing here uses
`Nat`, strings or IO, so the eventual archive stays runtime-free.

```lean
namespace LeanPort.Authz

/-- The five actions a caller can request. -/
inductive Action where
  | create
  | read
  | update
  | delete
  | viewAudit

/-- Effective level from *facts*, never from a client-supplied role.
    0 anonymous, 1 member, 2 owner, 3 admin. -/
def effectiveLevel (authenticated isAdmin ownsResource : Bool) : UInt8 :=
  if !authenticated then 0
  else if isAdmin then 3
  else if ownsResource then 2
  else 1

/-- The policy matrix. -/
def allows (level : UInt8) : Action → Bool
  | .create => level == 1 || level == 3
  | .read => level == 2 || level == 3
  | .update => level == 2 || level == 3
  | .delete => level == 2 || level == 3
  | .viewAudit => level == 3

def authorize (authenticated isAdmin ownsResource : Bool) (action : Action) : Bool :=
  allows (effectiveLevel authenticated isAdmin ownsResource) action

end LeanPort.Authz
```

### 4. State the guarantees

Replace `Proofs.lean` with theorems about the matrix. A representative subset:

```lean
import LeanPort.Authz.Policy

namespace LeanPort.Authz

theorem unauthenticated_denied (action : Action) :
    authorize false false false action = false := by
  cases action <;> rfl

theorem unauthenticated_cannot_become_admin (ownsResource : Bool) (action : Action) :
    authorize false true ownsResource action = false := by
  cases ownsResource <;> cases action <;> rfl

theorem member_can_create : authorize true false false .create = true := by rfl

theorem owner_cannot_view_audit : authorize true false true .viewAudit = false := by rfl

theorem audit_requires_admin (a adm own : Bool) :
    authorize a adm own .viewAudit = (a && adm) := by
  cases a <;> cases adm <;> cases own <;> rfl

end LeanPort.Authz
```

The checked-in file proves all nine properties listed above. Proofs are ordinary
Lean; `leanport` does not invent them.

### 5. Write the boundary

`Boundary.lean` is the only Lean file that knows about C. It decodes action
codes, calls the policy, and returns `1`/`0`. Unknown codes fail closed.

```lean
import LeanPort.Authz.Policy

namespace LeanPort.Authz

def decideForC (authenticated isAdmin ownsResource : UInt8) (action : Action) : UInt8 :=
  if authorize (authenticated == 1) (isAdmin == 1) (ownsResource == 1) action then 1 else 0

@[export leanport_authz_authorize]
def authorizeForC (authenticated isAdmin ownsResource actionCode : UInt8) : UInt8 :=
  if actionCode == 0 then decideForC authenticated isAdmin ownsResource .create
  else if actionCode == 1 then decideForC authenticated isAdmin ownsResource .read
  else if actionCode == 2 then decideForC authenticated isAdmin ownsResource .update
  else if actionCode == 3 then decideForC authenticated isAdmin ownsResource .delete
  else if actionCode == 4 then decideForC authenticated isAdmin ownsResource .viewAudit
  else 0

end LeanPort.Authz
```

Update the binding in `leanport.toml` so the signature and action table match:

```toml
[[binding]]
package  = "authz"
module   = "Authz"
export   = "leanport_authz_authorize"
go_name  = "Authorize"
args     = ["bool", "bool", "bool", "action:Action"]
result   = "bool"
runtime  = "auto"
actions  = { Create = 0, Read = 1, Update = 2, Delete = 3, ViewAudit = 4 }
```

`leanport new` initially writes `args = ["bool", "uint8"]`; replace that line.

### 6. Check the proofs

```bash
leanport check
```

*Expected:*

```
→ lake build LeanPort.Authz.Policy LeanPort.Authz.Proofs LeanPort.Authz.Boundary
Build completed successfully.
✓ checked 1 module(s); 1 export(s) verified
```

Then deliberately break it:

```bash
echo 'theorem broken : False := by sorry' >> lean/LeanPort/Authz/Proofs.lean
leanport check   # must fail
```

*Expected:* a non-zero exit naming `Proofs.lean` and the line. Remove the line
before continuing. This is the discipline the tool exists to enforce.

### 7. Build the native binding

```bash
leanport build
```

*Expected:*

```
→ internal/verified/authz/native/linux_amd64/libAuthz.a (none mode, 27.4 KB)
→ go build ./...
✓ build complete
```

`none mode` means no Lean runtime is linked. The bridge was compiled into the
archive; the Go package has no Lean headers and no C source of its own.

### 8. Inspect what was generated

```bash
ls internal/verified/authz
cat internal/verified/authz/authz.go
```

*Expected:* `authz.go`, `authz_cgo.go`, `artifact.go`, `MANIFEST.json` and the
archive. The public API is `type Action uint8`, five constants, and
`func Authorize(authenticated, isAdmin, ownsResource bool, action Action) bool`.
No `lean_object` appears.

### 9. Write the CLI

`cmd/permctl/main.go` is stdlib-only. The important line is:

```go
func allowed(authenticated, admin, owns bool, action authz.Action) bool {
	return authz.Authorize(authenticated, admin, owns, action)
}
```

The rest is argument parsing, a printable matrix, and a `verify` mode that
asserts all 20 policy cells plus ownership and invalid-code variants. See the
checked-in file for the full command.

---

## Run the CLI

```bash
go test ./...
go run ./cmd/permctl verify
go run ./cmd/permctl matrix
go run ./cmd/permctl decide --auth --owner --action update    # allow, exit 0
go run ./cmd/permctl decide --auth --owner --action viewAudit # deny, exit 1
```

*Expected output:*

```
verified 39 checks
LEVEL            create     read       update     delete     viewAudit
unauthenticated  deny       deny       deny       deny       deny
member           allow      deny       deny       deny       deny
owner            deny       allow      allow      allow      deny
admin            allow      allow      allow      allow      allow
allow
deny
```

`go run` reports the non-zero exit as `exit status 1`; a built binary simply
exits `1`, so `permctl decide … && do_something` works in shell.

## What was proved, and what was not

- **Proved in Lean:** the decision function's behaviour for every modelled input,
  including the nine theorems in `Proofs.lean`.
- **Tested in Go:** flag parsing, fact mapping, exit codes, the full matrix, and
  fail-closed behaviour for unknown action codes.
- **Not proved:** that the CLI authenticates the caller correctly, that
  ownership is derived from trustworthy data, or anything about deployment.
  Those are ordinary engineering and ordinary testing.

## What next

- [`examples/rich-types/`](../examples/rich-types/) adds `String` and
  `ByteArray` exports and shows the runtime-linked archive (~29 MB).
- [`docs/agents.md`](../docs/agents.md) describes how coding agents should edit
  this project.
- [`docs/limitations.md`](../docs/limitations.md) lists the boundary types and
  the unsupported ones.
- `leanport init --ci github` in a fresh project writes a workflow that runs the
  same checks on every push.
