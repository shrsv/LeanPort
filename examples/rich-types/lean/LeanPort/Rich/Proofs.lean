import LeanPort.Rich.Policy

namespace LeanPort.Rich

theorem sum_empty : sumBytes ByteArray.empty = 0 := by rfl

theorem reverse_empty : reverse ByteArray.empty = ByteArray.empty := by
  simp [reverse]

theorem greet_world : greet "world" = "Hello, world!" := by rfl

end LeanPort.Rich
