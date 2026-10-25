import LeanPort.Functional.Policy

namespace LeanPort.Functional

/-- `Nat` is unbounded and convenient for computation. The boundary converts
    the wire type `UInt64` to `Nat` and back. -/
@[export leanport_functional_sum_to]
def sumToForC (n : UInt64) : UInt64 :=
  (sumTo n.toNat).toUInt64

/-- A `Bool` result crosses the C ABI as 1/0. The binding declares it as
    `bool`, so Go sees `true`/`false`. -/
@[export leanport_functional_is_even]
def isEvenForC (n : UInt64) : UInt8 :=
  if isEven n.toNat then 1 else 0

/-- A second export from the same module. Both functions are ordinary Lean
    definitions; the boundary only adapts their types to C. -/
@[export leanport_functional_quadruple]
def quadrupleForC (n : UInt64) : UInt64 :=
  (quadruple n.toNat).toUInt64

end LeanPort.Functional
