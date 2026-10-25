# Example 1 — Executable Lean, called from Go

**What this teaches**

- Lean is an executable programming language, not only a proof assistant:
  `def` defines a function, `#eval` evaluates an expression, and
  `main : IO Unit` is an ordinary program entry point.
- LeanPort exports a **pure function**, not `main`. Your Go process keeps
  control; Lean is a native library behind it.
- The smallest useful LeanPort project has three Lean files, one binding entry,
  and a generated Go package.
- A scalar-only binding produces a **runtime-free** archive: no Lean runtime in
  your Go binary.

**Where this lives:** [`foundations/lean/LeanPort/Basics/`](foundations/lean/LeanPort/Basics/)

---

## 1. Run Lean by itself

Start a project and add one module:

```bash
mkdir foundations
cd foundations
go mod init example.com/foundations
leanport init --env local --ci none --name foundations --targets linux/amd64
leanport new --package learn --go-name Increment \
  --export leanport_basics_increment Basics
```

`leanport new` writes the three files and appends a binding with placeholder
types. Replace the Lean files with the versions below, then fix the binding.

> The checked-in version of this project is `tutorial/foundations`, and the rest
> of the path uses that path. If you created your own `foundations` directory,
> run the same commands from there; the names and code are identical.

`lean/LeanPort/Basics/Policy.lean`:

```lean
namespace LeanPort.Basics

/-- The smallest useful program: one pure function. -/
def increment (n : UInt64) : UInt64 :=
  n + 1

end LeanPort.Basics
```

You can evaluate it without any Go involvement. Add
`lean/Main.lean`:

```lean
import LeanPort.Basics.Policy

def main : IO Unit :=
  IO.println s!"increment 41 = {LeanPort.Basics.increment 41}"

#eval LeanPort.Basics.increment 41
```

Run it with the project's pinned toolchain:

```bash
cd lean
lake env lean --run Main.lean
```

```text
42
increment 41 = 42
```

The first line is `#eval` at compile time; the second is `main` at runtime.
Both execute the same Lean function. This is ordinary programming: no theorem
proving, no tactics, no Mathlib.

## 2. Add the C-facing boundary

LeanPort needs a symbol with a C-compatible signature. That is what
`Boundary.lean` is for.

`lean/LeanPort/Basics/Boundary.lean`:

```lean
import LeanPort.Basics.Policy

namespace LeanPort.Basics

/-- The C-facing boundary. Scalar in, scalar out: runtime-free. -/
@[export leanport_basics_increment]
def incrementForC (n : UInt64) : UInt64 :=
  increment n

end LeanPort.Basics
```

The `@[export ...]` name is the contract between Lean and LeanPort. The
boundary does not re-implement the rule; it adapts the domain function to the
wire types. `Proofs.lean` is empty for now:

```lean
import LeanPort.Basics.Policy

namespace LeanPort.Basics

-- Example 1 stops at executable code. Proofs start in Example 3.

end LeanPort.Basics
```

## 3. Describe the binding

Replace the scaffolded `[[binding]]` in `leanport.toml` with:

```toml
[[binding]]
  package = "learn"
  module = "Basics"
  export = "leanport_basics_increment"
  go_name = "Increment"
  args = ["uint64"]
  result = "uint64"
  runtime = "auto"
```

Each line is a promise:

| Field | Meaning here |
|---|---|
| `package` | the generated Go package name is `learn` |
| `module` | Lean lives under `lean/LeanPort/Basics/` |
| `export` | must match `@[export leanport_basics_increment]` |
| `go_name` | Go sees `learn.Increment` |
| `args` / `result` | one `UInt64` in, one `UInt64` out |
| `runtime` | `auto` chooses the cheapest mode |

## 4. Check, build, call

```bash
leanport check
leanport build
```

On this one-module project the important lines are:

```text
✓ checked 1 module(s); 1 export(s) verified
...
→ internal/verified/learn/native/linux_amd64/libBasics.a (none mode, 14.1 KB)
→ go build ./...
✓ build complete
```

`none mode` means no Lean runtime was linked. The generated package is tiny and
ordinary Go now:

```go
package main

import (
	"fmt"

	"example.com/foundations/internal/verified/learn"
)

func main() {
	fmt.Println(learn.Increment(41))
}
```

```bash
go run ./cmd/learn
```

```text
42
```

That is the whole loop. `leanport build` did the Lean-to-C work; `go build`
only sees a Go package and a static archive. There is no Lean installation on
the consumer side.

## What just happened

```
Policy.lean      executable Lean function
Boundary.lean    @[export] adapter with a stable C signature
leanport.toml    binding: Lean symbol -> Go name, types, package
leanport build   Lean -> C -> archive -> generated Go + cgo wrapper
go build         ordinary Go build; no Lean headers, no Lake
```

The checked-in [`foundations`](foundations/README.md) project keeps this module
and adds four more. Because later examples use `Nat`, `Option` and `Fin`, the
shared `learn` package becomes runtime-linked; see
[the project README](foundations/README.md#why-one-package-is-runtime-linked).
The single-module output above is what you get if you stop at Example 1.

## Try this

- Change `increment` to take two arguments (`n + step`) and update the binding.
- Add `decrement` as a second `@[export]` and a second `[[binding]]` in the same
  package. Run `leanport check && leanport build` and call both from Go.
- Deliberately misspell the `@[export]` name. Read the `leanport check` error:
  it tells you which binding could not find its symbol. Put it back.

Next: [Example 2 — pure functions and recursion](02-functions.md).
