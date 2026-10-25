# Example 3 — Specifications and proofs

**What this teaches**

- The difference between an implementation and a specification.
- `Option` makes failure explicit in the return type instead of hiding it in a
  sentinel value.
- How to state a property that should hold for **every** valid input, and let
  Lean check the proof.
- `simp` is proof automation: you do not need to learn tactic programming to
  read or write these first proofs.
- How to project a rich domain result (`Option`) onto a scalar boundary when the
  generator does not support the rich type yet.

**Where this lives:** [`foundations/lean/LeanPort/Division/`](foundations/lean/LeanPort/Division/)

This is the hinge of the path. Examples 1 and 2 were about executable code.
From here on, every function gets a stated contract.

---

## 1. The implementation

Add the module:

```bash
cd tutorial/foundations
leanport new --package learn --go-name SafeDivide \
  --export leanport_division_safe_divide Division
```

Replace `lean/LeanPort/Division/Policy.lean` with:

```lean
namespace LeanPort.Division

/-- Failure is represented in the return type. `none` means "no quotient
    exists for this input". There is no silent zero. -/
def safeDivide (n d : UInt64) : Option UInt64 :=
  if d = 0 then none else some (n / d)

/-- The scalar view used at the C boundary. Rejected divisions become zero.
    This is a deliberate API choice, not a hidden fallback in `safeDivide`. -/
def divideOrZero (n d : UInt64) : UInt64 :=
  (safeDivide n d).getD 0

end LeanPort.Division
```

Evaluate it:

```lean
import LeanPort.Division.Policy

#eval LeanPort.Division.safeDivide 12 3   -- some 4
#eval LeanPort.Division.safeDivide 12 0   -- none
```

`Option` is a small inductive type with two constructors: `none` and
`some value`. A caller must handle both. That is different from returning `0` on
failure, where the caller cannot tell "the answer is zero" from "there is no
answer".

## 2. The specification

`lean/LeanPort/Division/Proofs.lean`:

```lean
import LeanPort.Division.Policy

namespace LeanPort.Division

/-- Every zero divisor is rejected. -/
theorem safeDivide_zero (n : UInt64) :
    safeDivide n 0 = none := by
  simp [safeDivide]

/-- Every nonzero divisor produces the quotient. The hypothesis `h` is a
    precondition: the theorem says nothing when it does not hold. -/
theorem safeDivide_nonzero (n d : UInt64) (h : d ≠ 0) :
    safeDivide n d = some (n / d) := by
  simp [safeDivide, h]

/-- The scalar boundary turns rejection into zero. -/
theorem divideOrZero_zero (n : UInt64) :
    divideOrZero n 0 = 0 := by
  simp [divideOrZero, safeDivide]

/-- On valid input, the boundary returns exactly the quotient. -/
theorem divideOrZero_nonzero (n d : UInt64) (h : d ≠ 0) :
    divideOrZero n d = n / d := by
  simp [divideOrZero, safeDivide, h]

end LeanPort.Division
```

Read the first theorem as a contract:

> For every natural number `n`, `safeDivide n 0` is `none`.

That is stronger than a test table. A test can check a few values of `n`; the
theorem covers every `UInt64`. The second theorem covers every pair `(n, d)`
with `d ≠ 0`, not just the examples you thought to write down.

The proofs are short because `simp [safeDivide]` unfolds the definition and
simplifies the `if`. You do not need to understand the simplifier's internals;
Lean checks the proof term that results. If the theorem were false, `simp`
would leave a goal open and the build would fail.

## 3. The boundary

`lean/LeanPort/Division/Boundary.lean`:

```lean
import LeanPort.Division.Policy

namespace LeanPort.Division

/-- The boundary projects the rich `Option` result to a scalar. The proofs in
    `Proofs.lean` describe both the `Option` function and this projection. -/
@[export leanport_division_safe_divide]
def safeDivideForC (n d : UInt64) : UInt64 :=
  divideOrZero n d

end LeanPort.Division
```

The v0.1.1 generator supports scalars, strings and byte arrays. It does not
generate `Option` yet. Rather than pretend that `Option` crossed the boundary,
this example makes the projection explicit:

- `safeDivide` is the domain API, with failure in the type;
- `divideOrZero` is the scalar view chosen for this Go API;
- the proofs cover both, so the projection is not an unverified afterthought.

If you later want `(value, ok bool)` in Go, the binding table in
[`docs/cli-reference.md`](../docs/cli-reference.md) shows the planned
`option:<T>` mapping. Until then, an explicit scalar view is honest and total.

## 4. The binding

The scaffold has placeholder argument types. Replace its `[[binding]]` with:

```toml
[[binding]]
  package = "learn"
  module = "Division"
  export = "leanport_division_safe_divide"
  go_name = "SafeDivide"
  args = ["uint64", "uint64"]
  result = "uint64"
  runtime = "auto"
```

Generated Go:

```go
func SafeDivide(n uint64, d uint64) uint64
```

## 5. Check the proofs and call the function

```bash
leanport check
leanport build
go test ./...
```

The check line reports the shared project:

```text
✓ checked 5 module(s); 10 export(s) verified
```

If a proof is unfinished, `leanport check` refuses to pass. To see the
discipline, temporarily append a hole:

```bash
printf '\ntheorem injected_hole : True := by\n  sorry\n' \
  >> lean/LeanPort/Division/Proofs.lean
leanport check
```

```text
leanport: unproved holes found (remove sorry/admit):
  /home/you/foundations/lean/LeanPort/Division/Proofs.lean:...: sorry
```

Restore the file before continuing. LeanPort does not repair or invent proofs;
it runs Lean's checker and rejects holes. The same failure path is shown in
detail in the [capstone](06-authorization.md#step-8--prove-that-a-broken-proof-fails-the-build).

From Go:

```go
fmt.Printf("safeDivide 12 3 = %d\n", learn.SafeDivide(12, 3))
fmt.Printf("safeDivide 12 0 = %d\n", learn.SafeDivide(12, 0))
```

```text
safeDivide 12 3 = 4
safeDivide 12 0 = 0
```

The second line is the boundary projection. In the domain, the result was
`none`; in the scalar API it is `0`. The theorem
`divideOrZero_zero` says exactly that.

## What is proved, and what is not

- **Proved:** the Lean functions behave as stated for every input in their
  types.
- **Tested in Go:** the generated wrapper passes values through correctly.
- **Not proved:** that `0` is the right failure value for your application. That
  is an API design choice. If the caller must distinguish failure from a real
  zero, expose a second scalar function (for example `IsDivisible`) or wait for
  the `Option` generator.

The scope of the guarantee is the specification you wrote, not the whole
application.

## Try this

- Change `safeDivide` to reject a zero dividend as well, then update every
  theorem. Which theorem statements are now false?
- Add `isDivisible (n d : UInt64) : Bool` and prove it agrees with
  `(safeDivide n d).isSome`.
- Replace `divideOrZero`'s fallback with `n` instead of `0`. Update the proofs
  and explain why the new API is worse (or better) for your use case.

Next: [Example 4 — a business invariant](04-business-invariant.md).
