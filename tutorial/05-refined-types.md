# Example 5 — Types that enforce invariants

**What this teaches**

- A precondition can be an argument with a proof, not just a runtime check:
  `divideBy (n d : Nat) (h : d ≠ 0)`.
- A return type can carry an invariant: `Fin 101` is a natural number below 101,
  so "the result is at most 100" is true by construction.
- `Fin n` is also the right type for a bounded index; it is impossible to build
  `Fin 3` with value 3.
- The boundary is the one place where untrusted wire values are inspected and
  turned into these richer types. Invalid input fails closed.

**Where this lives:** [`foundations/lean/LeanPort/Refined/`](foundations/lean/LeanPort/Refined/)

This is the last teaching example before the capstone. It changes the question
from "did we test enough inputs?" to "can a caller even construct an invalid
input?".

---

## 1. Preconditions as evidence

Add the module:

```bash
cd tutorial/foundations
leanport new --package learn --go-name CheckedDivide \
  --export leanport_refined_checked_divide Refined
```

Replace `lean/LeanPort/Refined/Policy.lean` with:

```lean
namespace LeanPort.Refined

/-- Divide, but require *evidence* that the divisor is nonzero.
    There is no zero check in the body: a caller cannot supply `h` for `d = 0`. -/
def divideBy (n d : Nat) (_h : d ≠ 0) : Nat :=
  n / d

/-- The boundary wrapper. This is the one place that inspects a raw divisor and
    chooses what to do when the precondition fails. -/
def checkedDivide (n d : Nat) : Nat :=
  if h : d = 0 then 0 else divideBy n d h

/-- A percentage is a natural number below 101. The return type carries the
    invariant. `min n 100` clamps the input; `omega` proves the bound. -/
def percent (n : Nat) : Fin 101 :=
  ⟨min n 100, by omega⟩

/-- An index into three slots. `Fin 3` can only be 0, 1 or 2. -/
def slot (a b c : Nat) (i : Fin 3) : Nat :=
  match i with
  | ⟨0, _⟩ => a
  | ⟨1, _⟩ => b
  | _ => c

end LeanPort.Refined
```

Compare the two division functions:

| Function | Who checks `d ≠ 0`? | What a caller must supply |
|---|---|---|
| `checkedDivide` | the function, with `if h : d = 0` | a raw `Nat` |
| `divideBy` | nobody; the type requires proof | `n`, `d`, and `h : d ≠ 0` |

`divideBy` is the safer core API: a caller cannot invoke it with `d = 0`,
because it cannot produce the third argument. `checkedDivide` is the adapter for
untrusted input. It is the only place that needs a fallback.

`percent` shows the same idea on the result side. `Fin 101` is a subtype of
`Nat` with a proof that the value is `< 101`. A function returning `Fin 101`
cannot return 150. The `omega` tactic discharges the proof obligation
`min n 100 < 101`.

`slot` uses `Fin 3` for an index. `⟨3, by decide⟩` does not type-check, so an
out-of-range slot is not representable inside Lean.

## 2. Prove the wrapper agrees with the core

`lean/LeanPort/Refined/Proofs.lean`:

```lean
import LeanPort.Refined.Policy

namespace LeanPort.Refined

/-- The checked wrapper rejects a zero divisor instead of dividing. -/
theorem checkedDivide_zero (n : Nat) :
    checkedDivide n 0 = 0 := by
  simp [checkedDivide]

/-- With nonzero evidence, the wrapper is exactly the core function. -/
theorem checkedDivide_nonzero (n d : Nat) (h : d ≠ 0) :
    checkedDivide n d = n / d := by
  simp [checkedDivide, h, divideBy]

/-- The `Fin 101` result is always at most 100. This is a consequence of the
    return type, not a runtime check. -/
theorem percent_bounded (n : Nat) :
    (percent n).val ≤ 100 := by
  show min n 100 ≤ 100
  exact Nat.min_le_right _ _

/-- Inputs already in range are unchanged. -/
theorem percent_of_le (n : Nat) (h : n ≤ 100) :
    (percent n).val = n := by
  show min n 100 = n
  exact Nat.min_eq_left h

/-- Inputs above the range are clamped to 100. -/
theorem percent_of_gt (n : Nat) (h : 100 < n) :
    (percent n).val = 100 := by
  have h' : 100 ≤ n := Nat.le_of_lt h
  show min n 100 = 100
  exact Nat.min_eq_right h'

/-- Valid slots return the corresponding value. -/
theorem slot_zero (a b c : Nat) :
    slot a b c ⟨0, by decide⟩ = a := by
  rfl

theorem slot_one (a b c : Nat) :
    slot a b c ⟨1, by decide⟩ = b := by
  rfl

theorem slot_two (a b c : Nat) :
    slot a b c ⟨2, by decide⟩ = c := by
  rfl

end LeanPort.Refined
```

`checkedDivide_nonzero` is the bridge between the two designs. It says the
runtime check and the type-level proof describe the same behaviour. That is the
kind of theorem that keeps an adapter honest.

## 3. Boundaries validate once

`lean/LeanPort/Refined/Boundary.lean`:

```lean
import LeanPort.Refined.Policy

namespace LeanPort.Refined

/-- The boundary is where raw wire input is validated. The core `divideBy`
    never sees a zero divisor. -/
@[export leanport_refined_checked_divide]
def checkedDivideForC (n d : UInt64) : UInt64 :=
  (checkedDivide n.toNat d.toNat).toUInt64

/-- Decode a raw number, clamp it into `Fin 101`, then return its value. -/
@[export leanport_refined_percent]
def percentForC (n : UInt64) : UInt8 :=
  (percent n.toNat).val.toUInt8

/-- Decode a raw slot code into `Fin 3`. Invalid codes fail closed to slot 0. -/
@[export leanport_refined_slot]
def slotForC (a b c : UInt64) (code : UInt8) : UInt64 :=
  if h : code.toNat < 3 then
    (slot a.toNat b.toNat c.toNat ⟨code.toNat, h⟩).toUInt64
  else
    a

end LeanPort.Refined
```

Two boundaries do different things:

- `percentForC` clamps: 250 becomes 100. The `Fin 101` proof is constructed by
  `percent`, so the conversion to `UInt8` cannot truncate a value above 100.
- `slotForC` fails closed: a code of 99 returns the first slot (10 in the demo),
  not an out-of-bounds read. The `if h : code.toNat < 3` is where the `Fin 3`
  evidence is manufactured.

This is the pattern to reuse: **rich types and proofs in the core, one total
decoder at the edge.**

## 4. Three bindings

Replace the scaffolded `[[binding]]` with all three entries:

```toml
[[binding]]
  package = "learn"
  module = "Refined"
  export = "leanport_refined_checked_divide"
  go_name = "CheckedDivide"
  args = ["uint64", "uint64"]
  result = "uint64"
  runtime = "auto"

[[binding]]
  package = "learn"
  module = "Refined"
  export = "leanport_refined_percent"
  go_name = "Percent"
  args = ["uint64"]
  result = "uint8"
  runtime = "auto"

[[binding]]
  package = "learn"
  module = "Refined"
  export = "leanport_refined_slot"
  go_name = "Slot"
  args = ["uint64", "uint64", "uint64", "uint8"]
  result = "uint64"
  runtime = "auto"
```

Generated Go:

```go
func CheckedDivide(n, d uint64) uint64
func Percent(n uint64) uint8
func Slot(a, b, c uint64, code uint8) uint64
```

## 5. Call it from Go

```go
fmt.Printf("checkedDivide 10 2 = %d\n", learn.CheckedDivide(10, 2))
fmt.Printf("checkedDivide 10 0 = %d\n", learn.CheckedDivide(10, 0))
fmt.Printf("percent 42 = %d\n", learn.Percent(42))
fmt.Printf("percent 250 = %d\n", learn.Percent(250))
fmt.Printf("slot 10 20 30 1 = %d\n", learn.Slot(10, 20, 30, 1))
fmt.Printf("slot 10 20 30 99 = %d\n", learn.Slot(10, 20, 30, 99))
```

```text
checkedDivide 10 2 = 5
checkedDivide 10 0 = 0
percent 42 = 42
percent 250 = 100
slot 10 20 30 1 = 20
slot 10 20 30 99 = 10
```

The last line is the fail-closed decoder: code 99 is not a valid `Fin 3`, so the
boundary returns the first slot. The core `slot` function never saw an invalid
index.

## What is proved, and what is not

- **Proved:** the checked wrapper agrees with the core division function; the
  percentage is always at most 100; valid slots select the right values.
- **Tested in Go:** clamping, zero-divisor fallback and invalid-slot fallback.
- **Not proved:** that clamping is the right policy for a percentage, or that
  slot 0 is the right fallback for an invalid code. Those are product decisions
  encoded at the boundary.

## Try this

- Write `def half (n : Nat) (h : Even n) : Nat` using a predicate as
  evidence. What does the caller need to supply?
- Change `percent` to reject out-of-range input instead of clamping. Which type
  should it return? Update the boundary and the proofs.
- Add a `slotOrZero` variant that returns `0` for an invalid code instead of
  slot 0, and prove both decoders return a value from the three slots.

Next: [Example 6 — the capstone](06-authorization.md).
