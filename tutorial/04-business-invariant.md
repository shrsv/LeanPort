# Example 4 — A business invariant

**What this teaches**

- How to turn a business rule into an executable function plus a stated
  invariant.
- The invariant here is arithmetic and exact: a successful withdrawal never
  creates money, and the remainder plus the amount plus the fee equals the
  original balance.
- Why a domain model uses `Nat` (unbounded) even when the wire type is
  `UInt64`: `amount + fee` must not wrap around.
- How to expose one verified module as two Go functions when the boundary cannot
  carry a tuple or `Option`.

**Where this lives:** [`foundations/lean/LeanPort/Wallet/`](foundations/lean/LeanPort/Wallet/)

This is the first example that looks like code you might ship. It is still
deliberately small: one balance, one withdrawal, one fee.

---

## 1. The rule

Add the module:

```bash
cd tutorial/foundations
leanport new --package learn --go-name Allowed \
  --export leanport_wallet_allowed Wallet
```

Replace `lean/LeanPort/Wallet/Policy.lean` with:

```lean
namespace LeanPort.Wallet

/-- Withdraw `amount` plus `fee` if, and only if, the balance covers both.

    The domain uses `Nat` (unbounded) so `amount + fee` cannot wrap around.
    The boundary converts the wire types `UInt64` to `Nat` before calling. -/
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
```

Three decisions are visible in the code:

1. **The fee is part of the check.** `amount + fee ≤ balance`, not
   `amount ≤ balance`. A withdrawal that leaves no room for the fee is rejected.
2. **Failure is an `Option`, not a negative balance.** There is no state in
   which the account goes below zero.
3. **Rejection leaves the balance unchanged.** `balanceAfter` uses `getD
   balance`, so a failed withdrawal cannot silently spend money.

The last choice is an API decision. `BalanceAfter` alone cannot distinguish
"rejected" from "spent everything and ended at zero", so the Go API also exposes
`Allowed`. Use the flag first, then read the balance.

## 2. The invariants

`lean/LeanPort/Wallet/Proofs.lean`:

```lean
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

/-- `allowed` agrees with the `Option` result: true exactly when the withdrawal
    would succeed. -/
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
```

The interesting theorem is `withdraw_charges_fee`. It says:

> If the withdrawal succeeds with remainder `remaining`, then
> `remaining + amount + fee = balance`.

That is conservation of money for this model. It rules out both "the bank
created money" and "the bank kept part of the fee without recording it". The
proof uses `omega` for the arithmetic after unfolding the definition. `omega`
is a decision procedure for linear integer arithmetic; you do not need to
manually chain inequalities.

`withdraw_never_exceeds_balance` is a consequence, but it is worth stating on
its own because it is the property a reviewer will look for first.

## 3. The boundary

`lean/LeanPort/Wallet/Boundary.lean`:

```lean
import LeanPort.Wallet.Policy

namespace LeanPort.Wallet

/-- Decode the three wire values, then ask the verified domain function. -/
@[export leanport_wallet_allowed]
def allowedForC (balance amount fee : UInt64) : UInt8 :=
  if allowed balance.toNat amount.toNat fee.toNat then 1 else 0

/-- The balance view. On rejection it returns the original balance, so a caller
    must use `allowedForC` to distinguish rejection from "zero left". -/
@[export leanport_wallet_balance_after]
def balanceAfterForC (balance amount fee : UInt64) : UInt64 :=
  (balanceAfter balance.toNat amount.toNat fee.toNat).toUInt64

end LeanPort.Wallet
```

Go's `uint64` is converted to `Nat` before any arithmetic. This is why the
domain function can use `amount + fee` without worrying about 64-bit overflow.
The conversion is total: every `UInt64` is a `Nat`.

## 4. Two bindings

Replace the scaffolded `[[binding]]` with both entries:

```toml
[[binding]]
  package = "learn"
  module = "Wallet"
  export = "leanport_wallet_allowed"
  go_name = "Allowed"
  args = ["uint64", "uint64", "uint64"]
  result = "bool"
  runtime = "auto"

[[binding]]
  package = "learn"
  module = "Wallet"
  export = "leanport_wallet_balance_after"
  go_name = "BalanceAfter"
  args = ["uint64", "uint64", "uint64"]
  result = "uint64"
  runtime = "auto"
```

Generated Go:

```go
func Allowed(balance, amount, fee uint64) bool
func BalanceAfter(balance, amount, fee uint64) uint64
```

## 5. Call it from Go

```go
fmt.Printf("allowed(100, 30, 2) = %v\n", learn.Allowed(100, 30, 2))
fmt.Printf("balanceAfter(100, 30, 2) = %d\n", learn.BalanceAfter(100, 30, 2))
fmt.Printf("allowed(100, 99, 2) = %v\n", learn.Allowed(100, 99, 2))
fmt.Printf("balanceAfter(100, 99, 2) = %d\n", learn.BalanceAfter(100, 99, 2))
```

```text
allowed(100, 30, 2) = true
balanceAfter(100, 30, 2) = 68
allowed(100, 99, 2) = false
balanceAfter(100, 99, 2) = 100
```

The second withdrawal is rejected because `99 + 2 = 101 > 100`, even though
`99 ≤ 100`. That is the fee-aware check. The balance is unchanged at `100`.

## What is proved, and what is not

- **Proved:** the fee is included in the check; successful withdrawals conserve
  value; rejected withdrawals leave the balance unchanged; `Allowed` agrees with
  the underlying `Option`.
- **Tested in Go:** the wrapper maps three `uint64`s and two results correctly.
- **Not proved:** that the balance passed in is the customer's real balance, or
  that the database applies the returned balance atomically. Those are
  application and infrastructure concerns.

## Try this

- Change the rule to charge the fee only on success but not require it to fit
  the balance. State the new invariant. Is it still true that the balance
  never goes below zero?
- Add a minimum-balance rule (`balance - amount - fee ≥ minimum`) and prove
  that a successful withdrawal preserves it.
- Add a third export that returns `1` when the fee alone exceeds the balance,
  so the CLI can give a better error. Prove it agrees with the domain rule.

Next: [Example 5 — types that enforce invariants](05-refined-types.md).
