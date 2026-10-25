// Command learn runs the five small examples from the LeanPort learning path.
//
// Every number below comes from Lean code in lean/LeanPort/, compiled to a
// native archive and called through the generated package
// internal/verified/learn. The Go code never sees C, cgo or Lean.
package main

import (
	"fmt"

	"example.com/foundations/internal/verified/learn"
)

func main() {
	// Example 1: executable code.
	fmt.Printf("increment 41 = %d\n", learn.Increment(41))

	// Example 2: pure functions and recursion.
	fmt.Printf("sumTo 4 = %d\n", learn.SumTo(4))
	fmt.Printf("isEven 7 = %v\n", learn.IsEven(7))
	fmt.Printf("quadruple 3 = %d\n", learn.Quadruple(3))

	// Example 3: specifications and proofs, projected to a scalar.
	fmt.Printf("safeDivide 12 3 = %d\n", learn.SafeDivide(12, 3))
	fmt.Printf("safeDivide 12 0 = %d\n", learn.SafeDivide(12, 0))

	// Example 4: a business invariant with a fee.
	fmt.Printf("allowed(100, 30, 2) = %v\n", learn.Allowed(100, 30, 2))
	fmt.Printf("balanceAfter(100, 30, 2) = %d\n", learn.BalanceAfter(100, 30, 2))
	fmt.Printf("allowed(100, 99, 2) = %v\n", learn.Allowed(100, 99, 2))
	fmt.Printf("balanceAfter(100, 99, 2) = %d\n", learn.BalanceAfter(100, 99, 2))

	// Example 5: preconditions and bounds carried by types.
	fmt.Printf("checkedDivide 10 2 = %d\n", learn.CheckedDivide(10, 2))
	fmt.Printf("checkedDivide 10 0 = %d\n", learn.CheckedDivide(10, 0))
	fmt.Printf("percent 42 = %d\n", learn.Percent(42))
	fmt.Printf("percent 250 = %d\n", learn.Percent(250))
	fmt.Printf("slot 10 20 30 1 = %d\n", learn.Slot(10, 20, 30, 1))
	fmt.Printf("slot 10 20 30 99 = %d\n", learn.Slot(10, 20, 30, 99))
}
