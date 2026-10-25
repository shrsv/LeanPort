import LeanPort.Basics.Policy

namespace LeanPort.Basics

/-- The C-facing boundary.

    LeanPort calls this symbol from generated C. It returns a scalar, so the
    archive is runtime-free: no Lean runtime is linked into the Go binary. -/
@[export leanport_basics_increment]
def incrementForC (n : UInt64) : UInt64 :=
  increment n

end LeanPort.Basics
