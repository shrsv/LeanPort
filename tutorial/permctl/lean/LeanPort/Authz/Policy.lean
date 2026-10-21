namespace LeanPort.Authz

/-- The five actions a caller can request. -/
inductive Action where
  | create
  | read
  | update
  | delete
  | viewAudit

/-- Effective level from *facts*, never from a client-supplied role.
    0 anonymous, 1 member, 2 owner, 3 admin. -/
def effectiveLevel (authenticated isAdmin ownsResource : Bool) : UInt8 :=
  if !authenticated then 0
  else if isAdmin then 3
  else if ownsResource then 2
  else 1

/-- The policy matrix. -/
def allows (level : UInt8) : Action → Bool
  | .create => level == 1 || level == 3
  | .read => level == 2 || level == 3
  | .update => level == 2 || level == 3
  | .delete => level == 2 || level == 3
  | .viewAudit => level == 3

/-- Authorization from facts and an action. -/
def authorize (authenticated isAdmin ownsResource : Bool) (action : Action) : Bool :=
  allows (effectiveLevel authenticated isAdmin ownsResource) action

end LeanPort.Authz
