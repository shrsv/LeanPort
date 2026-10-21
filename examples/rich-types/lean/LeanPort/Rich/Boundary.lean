import LeanPort.Rich.Policy

namespace LeanPort.Rich

@[export leanport_rich_greet]
def greetForC (name : String) : String :=
  greet name

@[export leanport_rich_sum]
def sumForC (data : ByteArray) : UInt64 :=
  sumBytes data

@[export leanport_rich_reverse]
def reverseForC (data : ByteArray) : ByteArray :=
  reverse data

end LeanPort.Rich
