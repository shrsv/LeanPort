# Agent guide

This project is the shared home of Examples 1–5 from the tutorial. Later
examples use `Nat`, `Option` and `Fin`, so the single generated package `learn`
is runtime-linked; that is intentional. See `README.md`.

This project uses [LeanPort](https://github.com/shrsv/LeanPort): verified logic
is written in Lean and exposed to Go as an ordinary package.

## The one rule

Edit `lean/`. Never edit generated output under `internal/verified/`.
If generated code is wrong, fix the Lean source or `leanport.toml` and run
`leanport build`.

## Loop

    leanport check     # type-check proofs; rejects sorry/admit and missing exports
    leanport build     # Lean -> C -> static archive -> generated Go binding
    go test ./...      # verify the Go wiring
    leanport docs      # refresh SPEC.md files

## Where a change belongs

| Change | Edit |
|---|---|
| A policy rule | `lean/LeanPort/<Module>/Policy.lean` |
| A guarantee | `lean/LeanPort/<Module>/Proofs.lean` |
| What crosses the boundary | `lean/LeanPort/<Module>/Boundary.lean` + the `[[binding]]` entry |
| Application wiring | Go code outside `internal/verified/` |

## Boundary

Exports are either scalar-only (runtime-free archive) or use `String`/`ByteArray`
(runtime-linked archive). `leanport build` chooses automatically. Records,
arrays and `Option` are ABI-validated but not generated yet. Boundaries must
be total: invalid input returns the deny/zero result, never panics.

## Failure meanings

- **sorry/admit found** — a proof obligation is open; close the proof.
- **export X not found** — a missing or misspelled `@[export X]`, or the module did not build.
- **Lean build failed** — the definition or theorem does not type-check.

Never add `sorry` to reach a green build. Never hand-edit generated files.
