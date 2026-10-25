import LeanPort.Division.Policy

namespace LeanPort.Division

/-- The boundary projects the rich `Option` result to a scalar.

    `leanport build` does not generate `Option` yet, so this projection is
    explicit. The proofs in `Proofs.lean` describe both the `Option` function
    and this projection. -/
@[export leanport_division_safe_divide]
def safeDivideForC (n d : UInt64) : UInt64 :=
  divideOrZero n d

end LeanPort.Division
