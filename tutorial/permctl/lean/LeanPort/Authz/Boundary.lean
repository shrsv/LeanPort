import LeanPort.Authz.Policy

namespace LeanPort.Authz

/-- Apply the policy for one decoded action, encoding the result as 1/0. -/
def decideForC (authenticated isAdmin ownsResource : UInt8) (action : Action) : UInt8 :=
  if authorize (authenticated == 1) (isAdmin == 1) (ownsResource == 1) action then 1 else 0

/-- The C-facing boundary.

    Facts are 0/1 flags, never a client-supplied role. The result is 1 (allow)
    or 0 (deny). Unknown action codes fail closed. This is total and scalar-only,
    so the generated archive is runtime-free. -/
@[export leanport_authz_authorize]
def authorizeForC (authenticated isAdmin ownsResource actionCode : UInt8) : UInt8 :=
  if actionCode == 0 then decideForC authenticated isAdmin ownsResource .create
  else if actionCode == 1 then decideForC authenticated isAdmin ownsResource .read
  else if actionCode == 2 then decideForC authenticated isAdmin ownsResource .update
  else if actionCode == 3 then decideForC authenticated isAdmin ownsResource .delete
  else if actionCode == 4 then decideForC authenticated isAdmin ownsResource .viewAudit
  else 0

end LeanPort.Authz
