namespace LeanPort.Division

/-- Domain-level division: failure is represented in the return type.

    `none` means "no quotient exists for this input". The caller cannot ignore
    that case without deciding what to do; there is no silent zero. -/
def safeDivide (n d : UInt64) : Option UInt64 :=
  if d = 0 then none else some (n / d)

/-- The scalar view used at the C boundary. Rejected divisions become zero.
    This is a deliberate API choice, not a hidden fallback inside `safeDivide`. -/
def divideOrZero (n d : UInt64) : UInt64 :=
  (safeDivide n d).getD 0

end LeanPort.Division
