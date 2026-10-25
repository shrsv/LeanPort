# Example 2 — Pure functions and recursion

**What this teaches**

- Functional programming in Lean: functions are values, recursion is ordinary,
  and pattern matching replaces loops.
- Lean checks that recursive functions terminate.
- The boundary is where wire types become domain types. Here `UInt64` from Go is
  converted to `Nat` for computation, then back.
- One Lean module can expose several Go functions.
- Runtime mode is chosen per generated package: adding `Nat`, `Option`, `Fin`,
  `String` or `ByteArray` makes the package runtime-linked.

**Where this lives:** [`foundations/lean/LeanPort/Functional/`](foundations/lean/LeanPort/Functional/)

---

## 1. The Lean functions

Add the module:

```bash
cd tutorial/foundations
leanport new --package learn --go-name SumTo \
  --export leanport_functional_sum_to Functional
```

Replace `lean/LeanPort/Functional/Policy.lean` with:

```lean
namespace LeanPort.Functional

/-- Sum 0 + 1 + ... + n by structural recursion. -/
def sumTo : Nat → Nat
  | 0 => 0
  | n + 1 => (n + 1) + sumTo n

/-- A higher-order helper: apply `f` twice. -/
def twice (f : Nat → Nat) (n : Nat) : Nat :=
  f (f n)

def double (n : Nat) : Nat :=
  n + n

/-- Composition without any mutation: `double` applied twice is `×4`. -/
def quadruple (n : Nat) : Nat :=
  twice double n

/-- Pattern matching on a natural number. -/
def isEven : Nat → Bool
  | 0 => true
  | n + 1 => !isEven n

end LeanPort.Functional
```

Three ideas are doing the work:

1. `sumTo` recurses on the *structure* of `Nat`: a zero case and a successor
   case. Lean accepts it because the recursive call is on a smaller argument.
2. `twice` takes a function as an argument. `quadruple` builds a new function
   from `double` without naming any intermediate value.
3. `isEven` matches on `0` and `n + 1`, then negates the recursive result.

You can evaluate all of them without Go:

```lean
import LeanPort.Functional.Policy

#eval LeanPort.Functional.sumTo 4      -- 10
#eval LeanPort.Functional.quadruple 3  -- 12
#eval LeanPort.Functional.isEven 7     -- false
```

Put those lines in a scratch file under `lean/` and run
`lake env lean Scratch.lean`, or add them to `Main.lean` from Example 1.

## 2. Boundaries and conversions

Go's generated API uses fixed-width scalars. `Nat` is unbounded, so the boundary
converts explicitly. This is a feature: the conversion is visible and total.

`lean/LeanPort/Functional/Boundary.lean`:

```lean
import LeanPort.Functional.Policy

namespace LeanPort.Functional

@[export leanport_functional_sum_to]
def sumToForC (n : UInt64) : UInt64 :=
  (sumTo n.toNat).toUInt64

/-- A `Bool` result crosses the C ABI as 1/0; the binding says `bool`. -/
@[export leanport_functional_is_even]
def isEvenForC (n : UInt64) : UInt8 :=
  if isEven n.toNat then 1 else 0

@[export leanport_functional_quadruple]
def quadrupleForC (n : UInt64) : UInt64 :=
  (quadruple n.toNat).toUInt64

end LeanPort.Functional
```

`Proofs.lean` is still empty; this example is about computation, not
verification:

```lean
import LeanPort.Functional.Policy

namespace LeanPort.Functional

-- Example 2 is still about executable code. Example 3 introduces proofs.

end LeanPort.Functional
```

## 3. Three bindings, one package

`leanport new` added one placeholder binding. Replace it with these three; all
share the generated package `learn`:

```toml
[[binding]]
  package = "learn"
  module = "Functional"
  export = "leanport_functional_sum_to"
  go_name = "SumTo"
  args = ["uint64"]
  result = "uint64"
  runtime = "auto"

[[binding]]
  package = "learn"
  module = "Functional"
  export = "leanport_functional_is_even"
  go_name = "IsEven"
  args = ["uint64"]
  result = "bool"
  runtime = "auto"

[[binding]]
  package = "learn"
  module = "Functional"
  export = "leanport_functional_quadruple"
  go_name = "Quadruple"
  args = ["uint64"]
  result = "uint64"
  runtime = "auto"
```

`leanport check && leanport build` now generate three Go functions in the same
package:

```go
func SumTo(n uint64) uint64
func IsEven(n uint64) bool
func Quadruple(n uint64) uint64
```

## 4. Call them from Go

```go
fmt.Printf("sumTo 4 = %d\n", learn.SumTo(4))
fmt.Printf("isEven 7 = %v\n", learn.IsEven(7))
fmt.Printf("quadruple 3 = %d\n", learn.Quadruple(3))
```

```text
sumTo 4 = 10
isEven 7 = false
quadruple 3 = 12
```

Nothing in the Go call reveals that `sumTo` is recursive or that
`quadruple` was built from a higher-order function. That is the point: verified
or not, the consumer sees a normal Go API.

## Why the archive changes

If `Functional` were the only module and used only `UInt64`, LeanPort could
choose a runtime-free archive. `Nat` uses Lean's arbitrary-precision arithmetic,
and `Option` values need runtime support, so a module that uses them may force
linked mode.

LeanPort chooses one mode **per generated Go package**. The `foundations`
project puts all five examples in `learn`, so by the time Examples 3–5 are
present, `learn` is runtime-linked (about 29 MB). Example 6 takes the other
route: every value stays scalar, so `permctl` gets a ~27 KB runtime-free
archive. If package size matters, split modules into packages by their runtime
needs.

## Try this

- Write `triangular` using `sumTo`: the `n`th triangular number is
  `sumTo n`.
- Write `power (base exp : Nat) : Nat` by recursion on `exp`, expose it, and
  call it from Go. What happens for `exp = 0`?
- Add a proof later: after Example 3, come back and prove
  `sumTo 0 = 0` and `sumTo (n + 1) = sumTo n + (n + 1)`.

Next: [Example 3 — specifications and proofs](03-specifications.md).
