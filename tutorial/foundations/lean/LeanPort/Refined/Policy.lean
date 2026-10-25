namespace LeanPort.Refined

/-- Divide, but require *evidence* that the divisor is nonzero.

    There is no zero check in the body: a caller cannot construct the argument
    `h : d ≠ 0` for `d = 0`. The precondition lives in the type. -/
def divideBy (n d : Nat) (_h : d ≠ 0) : Nat :=
  n / d

/-- The boundary wrapper. This is the one place that inspects a raw divisor and
    chooses what to do when the precondition fails. -/
def checkedDivide (n d : Nat) : Nat :=
  if h : d = 0 then 0 else divideBy n d h

/-- A percentage is a natural number below 101. The return type carries the
    invariant: a `Fin 101` cannot be 101 or larger. `min n 100` clamps the
    input, and the proof obligation `min n 100 < 101` is discharged by `omega`. -/
def percent (n : Nat) : Fin 101 :=
  ⟨min n 100, by omega⟩

/-- An index into three slots. `Fin 3` can only be 0, 1 or 2. -/
def slot (a b c : Nat) (i : Fin 3) : Nat :=
  match i with
  | ⟨0, _⟩ => a
  | ⟨1, _⟩ => b
  | _ => c

end LeanPort.Refined
