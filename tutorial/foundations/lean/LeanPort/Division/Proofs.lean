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
