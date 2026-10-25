import LeanPort.Basics.Policy

/-- Run this with `lake env lean --run Main.lean` from `lean/`.

    `main` is ordinary executable Lean. LeanPort does not export `main`; it
    exports pure functions such as `increment`, which Go can call. -/
def main : IO Unit :=
  IO.println s!"increment 41 = {LeanPort.Basics.increment 41}"

#eval LeanPort.Basics.increment 41
