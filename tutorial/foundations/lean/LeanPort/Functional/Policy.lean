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
