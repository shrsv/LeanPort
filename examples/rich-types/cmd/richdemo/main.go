// Command richdemo shows Lean string, byte-array and scalar values crossing
// the boundary. The archive is runtime-linked; the consumer still needs no Lean
// installation.
package main

import (
	"fmt"

	"example.com/richtypes/internal/verified/rich"
)

func main() {
	fmt.Println(rich.Greet("LeanPort"))
	fmt.Printf("sum([1 2 3 250]) = %d\n", rich.Sum([]byte{1, 2, 3, 250}))
	fmt.Printf("reverse(\"leanport\") = %q\n", string(rich.Reverse([]byte("leanport"))))
}
