import LeanPort.Wallet.Policy

namespace LeanPort.Wallet

/-- Decode the three wire values, then ask the verified domain function.
    The result is 1 for allowed and 0 for rejected. -/
@[export leanport_wallet_allowed]
def allowedForC (balance amount fee : UInt64) : UInt8 :=
  if allowed balance.toNat amount.toNat fee.toNat then 1 else 0

/-- The balance view. On rejection it returns the original balance, so a caller
    must use `allowedForC` to distinguish "rejected" from "zero spent". -/
@[export leanport_wallet_balance_after]
def balanceAfterForC (balance amount fee : UInt64) : UInt64 :=
  (balanceAfter balance.toNat amount.toNat fee.toNat).toUInt64

end LeanPort.Wallet
