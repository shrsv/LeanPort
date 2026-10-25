# LeanPort examples: from executable Lean to a verified CLI

Six examples, in increasing order of difficulty. Each one teaches one Lean
idea, shows the LeanPort boundary that exposes it to Go, and ends with an
ordinary Go call. The first five share one small project; the sixth is a
complete command-line application.

| # | Example | Lean idea | LeanPort idea | Callable from Go |
|---|---|---|---|---|
| 1 | [Executable Lean, called from Go](01-executable.md) | `def`, `#eval`, `main : IO Unit` | first `@[export]`, scalar binding | `learn.Increment(41) == 42` |
| 2 | [Pure functions and recursion](02-functions.md) | recursion, pattern matching, higher-order functions | several exports, `UInt64`/`Nat` conversion | `learn.SumTo(4) == 10` |
| 3 | [Specifications and proofs](03-specifications.md) | `Option`, `theorem`, `simp` | prove first, then project to a scalar | `learn.SafeDivide(12, 0) == 0` |
| 4 | [A business invariant](04-business-invariant.md) | invariants over a fee and a balance | two exports from one verified module | `learn.Allowed(...)`, `learn.BalanceAfter(...)` |
| 5 | [Types that enforce invariants](05-refined-types.md) | proof arguments, `Fin n`, `omega` | validate raw input once, at the boundary | `learn.CheckedDivide`, `learn.Percent` |
| 6 | [Capstone: a verified authorization CLI](06-authorization.md) | inductive types, decision tables, ten theorems | actions enum, runtime-free archive, full CLI | `authz.Authorize(..., authz.Update)` |

## The shape of every example

LeanPort does not translate Lean into Go. It compiles the Lean implementation to
native code and generates a narrow cgo wrapper. Every example therefore has the
same five pieces:

```
lean/LeanPort/<Module>/Policy.lean      the executable rule
lean/LeanPort/<Module>/Proofs.lean      what must be true about it (from Example 3)
lean/LeanPort/<Module>/Boundary.lean    the total C-facing adapter + @[export]
leanport.toml                           the binding: Go package, name, types
internal/verified/<package>/            generated Go API + native archive
```

The application imports only the generated Go package. It never sees
`lean_object`, Lean initialization, or a C symbol.

## Run the examples

Examples 1–5 live in [`foundations/`](foundations/README.md):

```bash
cd tutorial/foundations
leanport check
leanport build
go test ./...
go run ./cmd/learn
```

Example 6 lives in [`permctl/`](permctl/README.md):

```bash
cd tutorial/permctl
leanport check
leanport build
go test ./...
go build -o permctl ./cmd/permctl
./permctl verify
./permctl matrix
```

If LeanPort is not installed, start with [Install](../README.md#install). You
need Go 1.24+, a C compiler, cgo enabled, and Lean 4 via `elan` to run
`leanport check` or `leanport build`. `go test` only needs Go and a C compiler
when the generated archive is already present.

## Suggested pace

- **Examples 1 and 2** are about the toolchain. Type the code, run it, and look
  at what `leanport build` puts under `internal/verified/`.
- **Example 3** is the hinge. Spend most of your time here: write a function,
  state a property that should hold for every valid input, and ask Lean to
  check it.
- **Example 4** applies that habit to a rule you might actually ship: a balance,
  an amount and a fee.
- **Example 5** moves the precondition from a runtime check into a type. This is
  the idea that makes Lean different from a test suite.
- **Example 6** combines all of it in a CLI with ten theorems and 39 runtime
  checks.

Examples 1–5 are deliberately small. They are not a tour of Lean's theorem
proving language, tactics or Mathlib. The goal is the programming-first path:
executable code, then pure functions, then specifications, then proofs, then
types that make invalid states hard to build.

If you want a book to accompany the path, *Functional Programming in Lean* is
the programming-oriented one. It is reasonable to skip *Theorem Proving in
Lean* and *Mathematics in Lean* until you have a reason to formalise
mathematics. No Mathlib is required: the proofs here use `simp` and `omega`,
which ship with Lean.

## Where the examples live

```
tutorial/
├── README.md                  this index
├── 01-executable.md           Example 1
├── 02-functions.md            Example 2
├── 03-specifications.md       Example 3
├── 04-business-invariant.md   Example 4
├── 05-refined-types.md        Example 5
├── 06-authorization.md        Example 6: the full permctl walkthrough
├── foundations/               shared project for Examples 1-5
│   ├── lean/LeanPort/...
│   ├── leanport.toml
│   └── cmd/learn/
└── permctl/                   Example 6 project
```

## After the examples

- [`examples/rich-types/`](../examples/rich-types/) shows `String` and
  `ByteArray` crossing the boundary, and why that switches to a runtime-linked
  archive.
- [`docs/cli-reference.md`](../docs/cli-reference.md) is the authoritative
  reference for every command and binding option.
- [`docs/limitations.md`](../docs/limitations.md) lists what the boundary can
  and cannot carry in v0.1.1.
- [`docs/agents.md`](../docs/agents.md) explains how coding agents should edit
  a LeanPort project.
