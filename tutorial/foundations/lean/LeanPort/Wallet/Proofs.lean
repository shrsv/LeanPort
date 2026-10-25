import LeanPort.Wallet.Policy

namespace LeanPort.Wallet

/-- If the amount plus fee exceeds the balance, the withdrawal is rejected. -/
theorem withdraw_rejects_overdraw (balance amount fee : Nat)
    (h : ¬ amount + fee ≤ balance) :
    withdraw balance amount fee = none := by
  simp [withdraw, h]

/-- A successful withdrawal never leaves more money than it started with. -/
theorem withdraw_never_exceeds_balance (balance amount fee remaining : Nat)
    (h : withdraw balance amount fee = some remaining) :
    remaining ≤ balance := by
  unfold withdraw at h
  split at h
  · injection h with h'
    omega
  · contradiction

/-- A successful withdrawal accounts for every unit: the remainder plus the
    amount plus the fee is exactly the original balance. -/
theorem withdraw_charges_fee (balance amount fee remaining : Nat)
    (h : withdraw balance amount fee = some remaining) :
    remaining + amount + fee = balance := by
  unfold withdraw at h
  split at h
  · injection h with h'
    omega
  · contradiction

/-- `allowed` agrees with the `Option` result: it is true exactly when the
    withdrawal would succeed. -/
theorem allowed_iff (balance amount fee : Nat) :
    allowed balance amount fee = true ↔ amount + fee ≤ balance := by
  unfold allowed withdraw
  by_cases h : amount + fee ≤ balance <;> simp [h]

/-- On success, `balanceAfter` is the new balance. -/
theorem balanceAfter_allowed (balance amount fee : Nat)
    (h : amount + fee ≤ balance) :
    balanceAfter balance amount fee = balance - (amount + fee) := by
  simp [balanceAfter, withdraw, h]

/-- On rejection, `balanceAfter` is the unchanged balance. -/
theorem balanceAfter_rejected (balance amount fee : Nat)
    (h : ¬ amount + fee ≤ balance) :
    balanceAfter balance amount fee = balance := by
  simp [balanceAfter, withdraw, h]

end LeanPort.Wallet
