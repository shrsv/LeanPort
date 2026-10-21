# Tutorial: build a verified CLI with LeanPort

This tutorial builds **`permctl`**, a small command-line tool whose
authorization policy is written and proved in Lean 4, compiled to native code,
and called from Go through a generated LeanPort binding.

The point is not `permctl` itself. The point is the workflow: *write rules and
proofs in Lean, get an ordinary Go package, ship a Go binary.*

You start from an empty directory. By the end you have:

- a Lean 4 policy with ten machine-checked theorems;
- a generated Go package with no Lean headers and no hand-written C source;
- a CLI with 39 runtime checks;
- a 27 KB runtime-free native archive;
- a project where `go test ./...` and `go build ./...` need only Go and a C
  compiler.

> **Captured from a real run.** The project commands below were executed in
> order on Lean 4.29.1, Go 1.24.3 and gcc 13.3.0 on linux/amd64. Output blocks
> are the actual output. Timings and absolute paths vary on your machine. The
> commands use a POSIX shell (Linux, macOS or WSL).

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

## Before you start

| Tool | Needed for | Check |
|---|---|---|
| Go 1.24+ | the application and the CLI | `go version` |
| a C compiler | cgo | `cc --version` |
| Lean 4 via [elan](https://github.com/leanprover/elan) | changing, proving or rebuilding Lean code | `lean --version` inside the project |
| Nix *(optional)* | pinned Go + C toolchain | `nix --version` |
| LeanPort | this workflow | `leanport version` |

Check the Go and C toolchains before starting:

```bash
go version
cc --version
```

```text
go version go1.24.3 linux/amd64
cc (Ubuntu 13.3.0-6ubuntu2~24.04.1) 13.3.0
```

Your versions and the exact `cc` banner will differ. Lean is checked after the
project is created, because elan selects the toolchain from
`lean/lean-toolchain`.

The generated package needs cgo. Make sure it is enabled:

```bash
export CGO_ENABLED=1
go env CGO_ENABLED
```

```text
1
```

Install LeanPort from a release:

```bash
VERSION=v0.1.1
curl -fsSL -o leanport \
  "https://github.com/shrsv/LeanPort/releases/download/${VERSION}/leanport_${VERSION}_linux_amd64"
chmod +x leanport
sudo install -m755 leanport /usr/local/bin/leanport
```

Choose `darwin_arm64`, `windows_amd64.exe`, and so on for your platform. With
Go:

```bash
go install github.com/shrsv/LeanPort/cmd/leanport@v0.1.1
```

`go install` writes to `$(go env GOPATH)/bin`; make sure that directory is on
your `PATH`.

Or build it from this repository:

```bash
git clone https://github.com/shrsv/LeanPort
cd LeanPort
go build -o leanport ./cmd/leanport
sudo install -m755 leanport /usr/local/bin/leanport
```

Check it:

```bash
leanport version
```

```text
leanport v0.1.1 (commit ccc6686, built 2026-10-21T12:30:16Z)
```

A binary installed with `go install` or built with `go build` from source prints
`leanport dev` (plus commit/build placeholders) because the release script
injects the version; that is expected.

If Lean is not installed:

```bash
curl -fsSL https://elan.lean-lang.org/elan-init.sh | sh -s -- -y
export PATH="$HOME/.elan/bin:$PATH"
```

You do **not** need a global Lean default. The `lean/lean-toolchain` file
written by `leanport init` pins Lean 4.29.1 for this project, and elan selects it
automatically.

### Optional: use the Nix flake

The flake lives at the repository root, not inside the tutorial project. From
the repository root:

```bash
cd LeanPort
nix develop
```

The shell provides Go, gcc and helpers, and bootstraps elan into `.elan/` if
Lean is not already installed. The first entry downloads the Lean toolchain
pinned by `lean/lean-toolchain`. The shell does not install LeanPort itself, so
build it once from the repository root:

```bash
go build -o leanport ./cmd/leanport
export PATH="$PWD:$PATH"
```

Then run the same commands below from your project directory. Verify the shell
with `go version`, `cc --version` and `lean --version`; Nix's own
download/build log is environment-specific and is not reproduced here. CI
validates the Nix path with `nix develop -c ./scripts/e2e.sh`.

---

## Step 1 — create the Go project and initialise LeanPort

```bash
mkdir permctl
cd permctl
go mod init example.com/permctl
```

```text
go: creating new go.mod: module example.com/permctl
```

Now initialise LeanPort. `--env local` is the PATH-based setup used in this
tutorial: it expects Go, a C compiler and Lean on your PATH. (`--env nix`
instead writes a project-local `flake.nix` and `.envrc`.) `--ci none` skips
writing a CI workflow for this tutorial.

```bash
leanport init --env local --ci none --name permctl --targets linux/amd64
```

```text
LeanPort initialised in /home/you/permctl
  Lean toolchain : 4.29.1
  Go module      : example.com/permctl
  targets        : linux/amd64
  environment    : local

Next:
  leanport new <module>   # scaffold a verified module
  leanport check          # prove it
  leanport build          # generate the Go binding
```

The first line prints an absolute path; yours will match the directory you
created. On macOS or Windows, choose your host target instead, for example
`--targets darwin/arm64` or `--targets windows/amd64`. v0.1.1 builds the host
target; cross-compilation needs a target C toolchain.

`init` created:

```text
./.gitignore
./AGENTS.md
./go.mod
./lean/lakefile.toml
./lean/lean-toolchain
./leanport.toml
```

`lean/lean-toolchain` pins the compiler:

```text
leanprover/lean4:4.29.1
```

`go.mod` contains the module:

```text
module example.com/permctl

go 1.24.3
```

The `go` directive reflects your installed Go version. `AGENTS.md` is the
project guide for coding agents; `leanport.toml` is the contract that ties Lean
modules to generated Go packages.

## Step 2 — scaffold the verified module

```bash
leanport new --package authz --go-name Authorize \
  --export leanport_authz_authorize Authz
```

```text
created lean/LeanPort/Authz/{Policy,Proofs,Boundary}.lean
added binding authz.Authorize -> leanport_authz_authorize

Edit the Lean files, then: leanport check && leanport build
```

The module name (`Authz`) is the final argument and must come **after** the
flags. The project now contains:

```text
./.gitignore
./AGENTS.md
./go.mod
./lean/LeanPort/Authz/Boundary.lean
./lean/LeanPort/Authz/Policy.lean
./lean/LeanPort/Authz/Proofs.lean
./lean/LeanPort/Authz/SPEC.md
./lean/lakefile.toml
./lean/lean-toolchain
./leanport.toml
```

`new` appended a Lake library for the module:

```text
name = "permctl"

# One lean_lib is appended per module by `leanport new`.

[[lean_lib]]
name = "LeanPort.Authz"
srcDir = "."
globs = ["LeanPort.Authz.*"]
```

The scaffold is a starting point. The next four steps replace it with the
`permctl` policy, proofs, boundary and binding.

## Step 3 — write the policy

Replace `lean/LeanPort/Authz/Policy.lean` with:

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

/-- Authorization from facts and an action. -/
def authorize (authenticated isAdmin ownsResource : Bool) (action : Action) : Bool :=
  allows (effectiveLevel authenticated isAdmin ownsResource) action

end LeanPort.Authz
```

`Action` is a small inductive; the boundary decodes wire codes into it. Nothing
here uses `Nat`, strings or IO, so the eventual archive stays runtime-free.

## Step 4 — state the guarantees

Replace `lean/LeanPort/Authz/Proofs.lean` with:

```lean
import LeanPort.Authz.Policy

namespace LeanPort.Authz

/-- Anonymous callers are denied every action. -/
theorem unauthenticated_denied (action : Action) :
    authorize false false false action = false := by
  cases action <;> rfl

/-- No combination of flags turns an anonymous caller into an admin. -/
theorem unauthenticated_cannot_become_admin (ownsResource : Bool) (action : Action) :
    authorize false true ownsResource action = false := by
  cases ownsResource <;> cases action <;> rfl

/-- Members may create, but not read resources they do not own. -/
theorem member_can_create : authorize true false false .create = true := by rfl

theorem member_cannot_read_others : authorize true false false .read = false := by rfl

/-- Owners may read, update and delete their own resource. -/
theorem owner_can_read : authorize true false true .read = true := by rfl

theorem owner_can_update : authorize true false true .update = true := by rfl

theorem owner_can_delete : authorize true false true .delete = true := by rfl

/-- Ownership does not grant audit access. -/
theorem owner_cannot_view_audit : authorize true false true .viewAudit = false := by rfl

/-- Admins may do everything. -/
theorem admin_can_everything (action : Action) :
    authorize true true false action = true := by
  cases action <;> rfl

/-- The audit log is readable exactly by authenticated admins. -/
theorem audit_requires_admin (a adm own : Bool) :
    authorize a adm own .viewAudit = (a && adm) := by
  cases a <;> cases adm <;> cases own <;> rfl

end LeanPort.Authz
```

These are ordinary Lean theorems. LeanPort does not invent or repair proofs; it
runs Lean's checker and rejects unproved holes.

## Step 5 — write the C boundary

`Boundary.lean` is the only Lean file that knows about C. Replace
`lean/LeanPort/Authz/Boundary.lean` with:

```lean
import LeanPort.Authz.Policy

namespace LeanPort.Authz

/-- Apply the policy for one decoded action, encoding the result as 1/0. -/
def decideForC (authenticated isAdmin ownsResource : UInt8) (action : Action) : UInt8 :=
  if authorize (authenticated == 1) (isAdmin == 1) (ownsResource == 1) action then 1 else 0

/-- The C-facing boundary.

    Facts are 0/1 flags, never a client-supplied role. The result is 1 (allow)
    or 0 (deny). Unknown action codes fail closed. This is total and scalar-only,
    so the generated archive is runtime-free. -/
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

Every path is total. Unknown action codes (anything other than 0–4) return `0`,
so invalid input fails closed rather than panicking or reaching a partial
function.

## Step 6 — update the binding contract

`leanport new` wrote a placeholder binding with two scalar arguments. Replace
the whole of `leanport.toml` with the final contract:

```toml
# LeanPort project contract. See docs/cli-reference.md.
# Files under internal/verified/ are generated; edit lean/ instead.

schema = 1

[project]
  name = "permctl"
  lean_dir = "lean"
  go_module = "example.com/permctl"
  package_dir = "internal/verified"
  targets = ["linux/amd64"]

[lean]
  toolchain = "4.29.1"

[[binding]]
  package = "authz"
  module = "Authz"
  export = "leanport_authz_authorize"
  go_name = "Authorize"
  args = ["bool", "bool", "bool", "action:Action"]
  result = "bool"
  runtime = "auto"
  actions = { Create = 0, Read = 1, Update = 2, Delete = 3, ViewAudit = 4 }
```

The `targets` line must match the target you chose in Step 1. The captured
outputs below use `linux/amd64`; on another host, keep your own `goos/goarch`
pair and expect the `native/...` path to change.

The `args` line maps the C boundary's four `UInt8` parameters to idiomatic Go
types. `action:Action` tells LeanPort to generate a named `Action` type and
decode it through the `actions` table. `runtime = "auto"` lets the builder
choose the cheapest mode; this scalar-only binding chooses runtime-free.

## Step 7 — check the proofs

```bash
leanport check
```

On a fresh project, the first check creates a manifest and builds all three
modules:

```text
→ lake build LeanPort.Authz.Policy LeanPort.Authz.Proofs LeanPort.Authz.Boundary
info: permctl: no previous manifest, creating one from scratch
info: toolchain not updated; already up-to-date
✔ [2/4] Built LeanPort.Authz.Policy (2.3s)
✔ [3/4] Built LeanPort.Authz.Proofs (340ms)
✔ [4/4] Built LeanPort.Authz.Boundary (338ms)
Build completed successfully (4 jobs).
✓ checked 1 module(s); 1 export(s) verified
```

The timings will differ, and Lake may print the modules in a different order
because it builds them in parallel. The last line is the important one: one
module was checked and the export named in `leanport.toml` exists.

You can confirm which Lean elan selected. Run this from `lean/`, because that
is where elan finds `lean-toolchain`:

```bash
(cd lean && lean --version)
```

```text
Lean (version 4.29.1, x86_64-unknown-linux-gnu, commit f72c35b3f637c8c6571d353742168ab66cc22c00, Release)
```

Running `lean --version` from the project root would use your global elan
default instead of the project's pinned toolchain; `leanport check` and
`leanport build` always use the project toolchain.

## Step 8 — prove that a broken proof fails the build

This is the discipline LeanPort exists to enforce. Back up the proofs, add a
deliberate hole, and run the check:

```bash
cp lean/LeanPort/Authz/Proofs.lean /tmp/Proofs.lean
printf '\ntheorem injected_hole : True := by\n  sorry\n' >> lean/LeanPort/Authz/Proofs.lean
status=0
leanport check || status=$?
echo "exit=$status"
mv /tmp/Proofs.lean lean/LeanPort/Authz/Proofs.lean
```

```text
leanport: unproved holes found (remove sorry/admit):
  /home/you/permctl/lean/LeanPort/Authz/Proofs.lean:43: sorry
exit=1
```

LeanPort prints the absolute path; the suffix will match your project. The
command exits non-zero, so CI fails. The `status=0` / `|| status=$?` form keeps
the block running under `set -e`, and the `mv` restores the proved file. Confirm:

```bash
leanport check
```

```text
✓ checked 1 module(s); 1 export(s) verified
```

(On a warm build the `lake build` line is still printed; the per-module timings
may be zero or omitted.)

## Step 9 — build the native binding

```bash
leanport build
```

```text
→ lake build LeanPort.Authz.Policy LeanPort.Authz.Proofs LeanPort.Authz.Boundary
Build completed successfully (4 jobs).
→ internal/verified/authz/native/linux_amd64/libAuthz.a (none mode, 27.4 KB)
→ go build ./...
✓ build complete
```

`none mode` means no Lean runtime is linked into the archive. The bridge was
compiled into `libAuthz.a`; the generated Go package contains no Lean headers
and no separate C source file. The archive is about 27 KB.

The target path is derived from `targets = ["linux/amd64"]`; on another host it
would be `native/darwin_arm64`, `native/windows_amd64`, and so on.

## Step 10 — inspect what was generated

```bash
find internal/verified/authz -type f | sort
```

```text
internal/verified/authz/MANIFEST.json
internal/verified/authz/artifact.go
internal/verified/authz/authz.go
internal/verified/authz/authz_cgo.go
internal/verified/authz/native/linux_amd64/libAuthz.a
```

The public Go API is small:

```bash
cat internal/verified/authz/authz.go
```

```go
// Code generated by leanport. DO NOT EDIT.
package authz

type Action uint8

const (
	Create    Action = 0
	Read      Action = 1
	Update    Action = 2
	Delete    Action = 3
	ViewAudit Action = 4
)
```

The cgo wrapper is also small — it declares the stable C entry point and
converts Go booleans to `uint8`:

```bash
cat internal/verified/authz/authz_cgo.go
```

```go
// Code generated by leanport. DO NOT EDIT.
package authz

/*
#cgo LDFLAGS: ${SRCDIR}/native/linux_amd64/libAuthz.a -lm -lpthread -ldl -lrt
#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>

uint8_t lp_go_Authorize(uint8_t a0, uint8_t a1, uint8_t a2, uint8_t a3);
*/
import "C"

func Authorize(a0 bool, a1 bool, a2 bool, a3 Action) bool {
	return C.lp_go_Authorize(b2u(a0), b2u(a1), b2u(a2), C.uint8_t(a3)) != 0
}

func b2u(b bool) C.uint8_t {
	if b {
		return 1
	}
	return 0
}
```

`artifact.go` carries the archive digest so Go's build cache invalidates when
the native code changes; `MANIFEST.json` records the toolchain, export table,
digests and selected mode. No `lean_object` appears in the public API.

Commit `lean/`, `leanport.toml` and `internal/verified/`. The `.gitignore`
written by `init` excludes only caches (`.leanport/`, `.lake/`, `.elan/`,
`.direnv/`), so consumers can build the generated package without Lean.

## Step 11 — write the CLI

Create the command directory:

```bash
mkdir -p cmd/permctl
```

Write `cmd/permctl/main.go`:

```go
// Command permctl is the LeanPort tutorial application.
//
// The authorization policy and its proofs live in lean/LeanPort/Authz/.
// Everything under internal/verified/ is generated by `leanport build`.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"example.com/permctl/internal/verified/authz"
)

var version = "dev"

type level struct {
	name string
	auth bool
	adm  bool
	own  bool
}

var levels = []level{
	{"unauthenticated", false, false, false},
	{"member", true, false, false},
	{"owner", true, false, true},
	{"admin", true, true, false},
}

var actions = []struct {
	name string
	code authz.Action
}{
	{"create", authz.Create},
	{"read", authz.Read},
	{"update", authz.Update},
	{"delete", authz.Delete},
	{"viewAudit", authz.ViewAudit},
}

// expected is the policy table from the specification. verify asserts that the
// generated binding agrees with every cell.
var expected = map[string]map[string]bool{
	"unauthenticated": {"create": false, "read": false, "update": false, "delete": false, "viewAudit": false},
	"member":          {"create": true, "read": false, "update": false, "delete": false, "viewAudit": false},
	"owner":           {"create": false, "read": true, "update": true, "delete": true, "viewAudit": false},
	"admin":           {"create": true, "read": true, "update": true, "delete": true, "viewAudit": true},
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 0
	}
	switch args[0] {
	case "decide":
		return runDecide(args[1:], stdout, stderr)
	case "matrix":
		return runMatrix(args[1:], stdout, stderr)
	case "verify":
		return runVerify(args[1:], stdout, stderr)
	case "version", "--version":
		fmt.Fprintf(stdout, "permctl %s\n", version)
		return 0
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "permctl: unknown command %q\n", args[0])
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `permctl - verified authorization, powered by LeanPort

Usage:
  permctl decide --auth [--admin] [--owner] --action <action>
  permctl matrix
  permctl verify
  permctl version

Actions: create, read, update, delete, viewAudit

decide prints allow or deny and exits 0 or 1, so it composes in shell:
  permctl decide --auth --owner --action update && rm resource
`)
}

func runDecide(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("decide", flag.ContinueOnError)
	fs.SetOutput(stderr)
	auth := fs.Bool("auth", false, "caller is authenticated")
	adm := fs.Bool("admin", false, "caller is an admin")
	own := fs.Bool("owner", false, "caller owns this resource")
	action := fs.String("action", "", "action to check")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	code, ok := lookupAction(*action)
	if !ok {
		fmt.Fprintf(stderr, "permctl: unknown action %q\n", *action)
		return 2
	}
	if allowed(*auth, *adm, *own, code) {
		fmt.Fprintln(stdout, "allow")
		return 0
	}
	fmt.Fprintln(stdout, "deny")
	return 1
}

func runMatrix(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("matrix", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	fmt.Fprintf(stdout, "%-16s", "LEVEL")
	for _, a := range actions {
		fmt.Fprintf(stdout, " %-10s", a.name)
	}
	fmt.Fprintln(stdout)
	for _, l := range levels {
		fmt.Fprintf(stdout, "%-16s", l.name)
		for _, a := range actions {
			word := "deny"
			if allowed(l.auth, l.adm, l.own, a.code) {
				word = "allow"
			}
			fmt.Fprintf(stdout, " %-10s", word)
		}
		fmt.Fprintln(stdout)
	}
	return 0
}

func runVerify(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	failures := 0
	checks := 0
	check := func(what string, got, want bool) {
		checks++
		if got != want {
			failures++
			fmt.Fprintf(stderr, "FAIL %s: got %v, want %v\n", what, got, want)
		}
	}
	for _, l := range levels {
		for _, a := range actions {
			check(l.name+"/"+a.name, allowed(l.auth, l.adm, l.own, a.code), expected[l.name][a.name])
		}
	}
	// Ownership is a fact, not a role: a member who owns the resource gets the
	// owner row. A client-supplied admin flag without authentication must not.
	for _, a := range actions {
		check("member-who-owns/"+a.name, allowed(true, false, true, a.code), expected["owner"][a.name])
		check("anonymous-with-flags/"+a.name, allowed(false, true, true, a.code), false)
		check("admin-who-owns/"+a.name, allowed(true, true, true, a.code), true)
	}
	// Unknown wire codes fail closed. The binding exposes uint8-backed Actions.
	for _, code := range []authz.Action{5, 6, 99, 255} {
		check(fmt.Sprintf("invalid-code-%d", code), allowed(true, true, true, code), false)
	}
	if failures > 0 {
		fmt.Fprintf(stderr, "%d/%d checks failed\n", failures, checks)
		return 1
	}
	fmt.Fprintf(stdout, "verified %d checks\n", checks)
	return 0
}

func allowed(authenticated, admin, owns bool, action authz.Action) bool {
	return authz.Authorize(authenticated, admin, owns, action)
}

func lookupAction(s string) (authz.Action, bool) {
	s = strings.ToLower(strings.ReplaceAll(s, "-", ""))
	for _, a := range actions {
		if strings.ToLower(a.name) == s {
			return a.code, true
		}
	}
	return 0, false
}
```

The important line is:

```go
func allowed(authenticated, admin, owns bool, action authz.Action) bool {
	return authz.Authorize(authenticated, admin, owns, action)
}
```

The rest is standard-library argument parsing, a printable matrix, and a
`verify` mode that asserts all 20 policy cells plus ownership, invalid
client-flags and invalid wire-code variants.

## Step 12 — write the Go tests

Write `cmd/permctl/main_test.go`:

```go
package main

import (
	"bytes"
	"strings"
	"testing"

	"example.com/permctl/internal/verified/authz"
)

func TestMatrixMatchesSpecification(t *testing.T) {
	for _, l := range levels {
		for _, a := range actions {
			got := allowed(l.auth, l.adm, l.own, a.code)
			want := expected[l.name][a.name]
			if got != want {
				t.Errorf("%s/%s: got %v, want %v", l.name, a.name, got, want)
			}
		}
	}
}

func TestVerifyCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"verify"}, &out, &errOut); code != 0 {
		t.Fatalf("verify exited %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "verified") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestDecideExitCodes(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"decide", "--auth", "--owner", "--action", "update"}, &out, &errOut); code != 0 {
		t.Fatalf("owner/update should allow, got %d (%s)", code, out.String())
	}
	if !strings.Contains(out.String(), "allow") {
		t.Fatalf("expected allow, got %q", out.String())
	}
	out.Reset()
	if code := run([]string{"decide", "--auth", "--owner", "--action", "viewAudit"}, &out, &errOut); code != 1 {
		t.Fatalf("owner/viewAudit should deny with exit 1, got %d", code)
	}
	if !strings.Contains(out.String(), "deny") {
		t.Fatalf("expected deny, got %q", out.String())
	}
}

func TestUnknownActionIsUsageError(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"decide", "--auth", "--action", "drop-tables"}, &out, &errOut); code != 2 {
		t.Fatalf("want exit 2, got %d", code)
	}
}

func TestInvalidWireCodeFailsClosed(t *testing.T) {
	for _, code := range []authz.Action{5, 255} {
		if allowed(true, true, true, code) {
			t.Errorf("invalid action code %d must be denied", code)
		}
	}
}
```

## Step 13 — run the Go tests

```bash
go test ./...
```

```text
ok  	example.com/permctl/cmd/permctl	0.004s
?   	example.com/permctl/internal/verified/authz	[no test files]
```

The time varies. The generated package has no tests of its own; the CLI tests
call it.

## Step 14 — build and run the CLI

Build a named binary and inject a version:

```bash
go build -ldflags "-X main.version=v1.0.0" -o permctl ./cmd/permctl
./permctl version
```

```text
permctl v1.0.0
```

Run the built-in conformance check:

```bash
./permctl verify
```

```text
verified 39 checks
```

Print the whole matrix:

```bash
./permctl matrix
```

```text
LEVEL            create     read       update     delete     viewAudit
unauthenticated  deny       deny       deny       deny       deny
member           allow      deny       deny       deny       deny
owner            deny       allow      allow      allow      deny
admin            allow      allow      allow      allow      allow
```

Decide individual requests. `decide` prints `allow`/`deny` and exits 0/1:

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

An owner may update their resource but may not read the audit log. The exit code
makes `permctl decide … && do_something` work directly in shell.

## Step 15 — regenerate the specification

```bash
leanport docs
```

```text
wrote /home/you/permctl/lean/LeanPort/Authz/SPEC.md
```

The path is absolute; yours will differ. Read the generated spec:

```bash
cat lean/LeanPort/Authz/SPEC.md
```

```markdown
# Authz specification

> Generated by `leanport docs`. Edit `Policy.lean` and `Proofs.lean`,
> not this file.

## Exports

| C symbol | Go | Arguments | Result |
|---|---|---|---|
| `leanport_authz_authorize` | `authz.Authorize` | `bool, bool, bool, action:Action` | `bool` |

## Guarantees

See `Proofs.lean` for the machine-checked theorems.
```

Run `leanport docs` after changing `leanport.toml`, `Policy.lean` or
`Proofs.lean`. The spec is generated from the binding and the checked module,
not maintained by hand.

## Step 16 — verify the whole project

At this point the complete loop is:

```bash
leanport check
leanport build
go test ./...
./permctl verify
```

The key line from each command on a warm build is:

```text
✓ checked 1 module(s); 1 export(s) verified
✓ build complete
ok  	example.com/permctl/cmd/permctl	0.004s
verified 39 checks
```

If something fails, ask LeanPort what it can see:

```bash
leanport doctor
```

```text
LeanPort doctor

  go         /home/you/.go/bin/go  go version go1.24.3 linux/amd64
  cc         /usr/bin/cc  cc (Ubuntu 13.3.0-6ubuntu2~24.04.1) 13.3.0
  lean       /home/you/.elan/bin/lean  Lean (version 4.29.1, x86_64-unknown-linux-gnu, commit f72c35b3f637c8c6571d353742168ab66cc22c00, Release)
  lake       /home/you/.elan/bin/lake  Lake version 5.0.0-src+f72c35b (Lean version 4.29.1)
  nix        MISSING

  CGO_ENABLED  enabled

  project      permctl
  module       example.com/permctl
  lean_dir     /home/you/permctl/lean
  targets      linux/amd64
  lean prefix  /home/you/.elan/toolchains/leanprover--lean4---v4.29.1
  toolchain    leanprover/lean4:4.29.1
  bindings     1

✓ toolchains look usable
```

`nix MISSING` is fine if you are not using the flake. The important fields are
the Go, C and Lean paths, `CGO_ENABLED`, the pinned toolchain, and the project's
bindings.

## Step 17 — add a second module and call it from Go

`authz` is not special, and you do not edit generated files by hand. Add a
second verified module with `leanport new`:

```bash
leanport new --package quota --go-name AllowQuota \
  --export leanport_quota_allow Quota
```

```text
created lean/LeanPort/Quota/{Policy,Proofs,Boundary}.lean
added binding quota.AllowQuota -> leanport_quota_allow

Edit the Lean files, then: leanport check && leanport build
```

`new` does three things: it creates the four files under
`lean/LeanPort/Quota/`, appends a `[[lean_lib]]` to `lean/lakefile.toml`, and
appends this binding to `leanport.toml`:

```toml
[[binding]]
  package = "quota"
  module = "Quota"
  export = "leanport_quota_allow"
  go_name = "AllowQuota"
  args = ["bool", "uint8"]
  result = "bool"
  runtime = "auto"
```

The scaffold is already total, so you can build it before replacing it with
your own rule. The Lean policy is:

```lean
namespace LeanPort.Quota

def decide (enabled : Bool) (code : UInt8) : Bool :=
  enabled && code != 0

end LeanPort.Quota
```

and the boundary is:

```lean
import LeanPort.Quota.Policy

namespace LeanPort.Quota

@[export leanport_quota_allow]
def AllowQuotaForC (enabled : UInt8) (code : UInt8) : UInt8 :=
  if decide (enabled == 1) code then 1 else 0

end LeanPort.Quota
```

Now prove and rebuild:

```bash
leanport check
leanport build
```

```text
→ lake build LeanPort.Authz.Policy LeanPort.Authz.Proofs LeanPort.Authz.Boundary LeanPort.Quota.Policy LeanPort.Quota.Proofs LeanPort.Quota.Boundary
Build completed successfully (7 jobs).
✓ checked 2 module(s); 2 export(s) verified
...
→ internal/verified/quota/native/linux_amd64/libQuota.a (none mode, 36.1 KB)
→ go build ./...
✓ build complete
```

`build` regenerates every package, including `internal/verified/authz/`; the
new package is `internal/verified/quota/`. Use it from a small Go program:

```go
package main

import (
	"fmt"

	"example.com/permctl/internal/verified/authz"
	"example.com/permctl/internal/verified/quota"
)

func main() {
	fmt.Println(authz.Authorize(true, false, true, authz.Update))
	fmt.Println(quota.AllowQuota(true, 7))
	fmt.Println(quota.AllowQuota(false, 7))
}
```

```bash
mkdir -p cmd/quotademo
cat > cmd/quotademo/main.go <<'EOF'
package main

import (
	"fmt"

	"example.com/permctl/internal/verified/authz"
	"example.com/permctl/internal/verified/quota"
)

func main() {
	fmt.Println(authz.Authorize(true, false, true, authz.Update))
	fmt.Println(quota.AllowQuota(true, 7))
	fmt.Println(quota.AllowQuota(false, 7))
}
EOF
go run ./cmd/quotademo
```

```text
true
true
false
```

The first line is the existing `authz` package (an owner may update). The next
two are the new `quota` package: enabled with code 7 allows, disabled denies.

**This is the rule to remember:** after adding or changing Lean, run
`leanport check && leanport build`; then `go build` or `go run` as usual. There
is no watcher. `go build` never reads `lean/`, so changing Lean alone does not
change the Go API or the archive. `leanport new` is also the supported way to
add a module — if you wire one by hand, you must update both `lakefile.toml` and
`leanport.toml`.

`String` and `ByteArray` modules use runtime-linked archives. Keep those
exports in one package: each runtime-linked package embeds the Lean runtime, and
multiple runtime-linked packages are not supported in v0.1.1.

## What was proved, and what was not

- **Proved in Lean:** the decision function's behaviour for every modelled input,
  including the ten theorems in `Proofs.lean`.
- **Tested in Go:** flag parsing, fact mapping, exit codes, the full matrix, and
  fail-closed behaviour for unknown action codes.
- **Not proved:** that the CLI authenticates the caller correctly, that
  ownership is derived from trustworthy data, or anything about deployment.
  Those are ordinary engineering and ordinary testing.

Lean proves the policy. LeanPort checks that the named export exists and that
the generated Go package builds. Neither proves that your HTTP layer, database,
or deployment is correct.

## What next

- [`examples/rich-types/`](../examples/rich-types/) adds `String` and
  `ByteArray` exports and shows the runtime-linked archive (~29 MB).
- [`docs/agents.md`](../docs/agents.md) describes how coding agents should edit
  this project.
- [`docs/limitations.md`](../docs/limitations.md) lists the boundary types and
  the unsupported ones.
- [`docs/cli-reference.md`](../docs/cli-reference.md) is the authoritative
  command, configuration and exit-code reference.
- `leanport init --ci github` in a fresh project writes a workflow that runs the
  same checks on every push.
