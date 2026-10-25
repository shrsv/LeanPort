namespace LeanPort.Wallet

/--Withdraw `amount` plus `fee` if, and only if, the balance covers both.

    The domain uses `Nat` (unbounded) so `amount + fee` cannot wrap around.
    The boundary converts the wire types `UInt64` to `Nat` before calling this
    function. -/
def withdraw (balance amount fee : Nat) : Option Nat :=
  if amount + fee ≤ balance then
    some (balance - (amount + fee))
  else
    none

/-- The question a Go caller normally asks: may this withdrawal proceed? -/
def allowed (balance amount fee : Nat) : Bool :=
  (withdraw balance amount fee).isSome

/-- The balance after a successful withdrawal; unchanged after a rejection. -/
def balanceAfter (balance amount fee : Nat) : Nat :=
  (withdraw balance amount fee).getD balance

end LeanPort.Wallet
