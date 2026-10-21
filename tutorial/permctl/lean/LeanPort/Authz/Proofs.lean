import LeanPort.Authz.Policy

namespace LeanPort.Authz

/-- Anonymous callers are denied every action. -/
theorem unauthenticated_denied (action : Action) :
    authorize false false false action = false := by
  cases action <;> rfl

/-- No combination of flags turns an anonymous caller into an admin. -/
theorem unauthenticated_cannot_become_admin (ownsResource : Bool) (action : Action) :
    authorize false true ownsResource action = false := by
  cases ownsResource <;> cases action <;> rfl

/-- Members may create, but not read resources they do not own. -/
theorem member_can_create : authorize true false false .create = true := by rfl

theorem member_cannot_read_others : authorize true false false .read = false := by rfl

/-- Owners may read, update and delete their own resource. -/
theorem owner_can_read : authorize true false true .read = true := by rfl

theorem owner_can_update : authorize true false true .update = true := by rfl

theorem owner_can_delete : authorize true false true .delete = true := by rfl

/-- Ownership does not grant audit access. -/
theorem owner_cannot_view_audit : authorize true false true .viewAudit = false := by rfl

/-- Admins may do everything. -/
theorem admin_can_everything (action : Action) :
    authorize true true false action = true := by
  cases action <;> rfl

/-- The audit log is readable exactly by authenticated admins. -/
theorem audit_requires_admin (a adm own : Bool) :
    authorize a adm own .viewAudit = (a && adm) := by
  cases a <;> cases adm <;> cases own <;> rfl

end LeanPort.Authz
