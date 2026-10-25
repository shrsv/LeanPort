namespace LeanPort.Basics

/-- The smallest useful program: one pure function.

    Lean can evaluate this directly with `#eval`, and LeanPort can expose the
    same function to Go. There is no proof here yet; this example is about the
    path from Lean source to a working Go call. -/
def increment (n : UInt64) : UInt64 :=
  n + 1

end LeanPort.Basics
