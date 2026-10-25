import LeanPort.Refined.Policy

namespace LeanPort.Refined

/-- The boundary is where raw wire input is validated. The core `divideBy`
    never sees a zero divisor. -/
@[export leanport_refined_checked_divide]
def checkedDivideForC (n d : UInt64) : UInt64 :=
  (checkedDivide n.toNat d.toNat).toUInt64

/-- Decode a raw number, clamp it into `Fin 101`, then return its value. The
    result is guaranteed to be at most 100. -/
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
