namespace LeanPort.Rich

/-- Build a greeting. Strings cross the boundary as boxed Lean values. -/
def greet (name : String) : String :=
  "Hello, " ++ name ++ "!"

/-- Sum a byte array without leaving Lean. -/
def sumBytes (data : ByteArray) : UInt64 :=
  data.foldl (fun acc b => acc + b.toUInt64) 0

/-- Reverse a byte array, returning a fresh one. -/
def reverse (data : ByteArray) : ByteArray :=
  data.toList.reverse.toByteArray

end LeanPort.Rich
