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

/-- The `Fin 101` result is always at most 100. This theorem is a consequence
    of the return type, not a runtime check. -/
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
