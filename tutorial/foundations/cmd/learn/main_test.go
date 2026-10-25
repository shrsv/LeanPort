package main

import (
	"testing"

	"example.com/foundations/internal/verified/learn"
)

func TestExample1Increment(t *testing.T) {
	if got := learn.Increment(41); got != 42 {
		t.Fatalf("Increment(41) = %d, want 42", got)
	}
}

func TestExample2Functions(t *testing.T) {
	if got := learn.SumTo(4); got != 10 {
		t.Fatalf("SumTo(4) = %d, want 10", got)
	}
	if got := learn.IsEven(7); got {
		t.Fatal("IsEven(7) = true, want false")
	}
	if got := learn.Quadruple(3); got != 12 {
		t.Fatalf("Quadruple(3) = %d, want 12", got)
	}
}

func TestExample3SafeDivide(t *testing.T) {
	if got := learn.SafeDivide(12, 3); got != 4 {
		t.Fatalf("SafeDivide(12, 3) = %d, want 4", got)
	}
	if got := learn.SafeDivide(12, 0); got != 0 {
		t.Fatalf("SafeDivide(12, 0) = %d, want 0", got)
	}
}

func TestExample4Wallet(t *testing.T) {
	if !learn.Allowed(100, 30, 2) {
		t.Fatal("Allowed(100, 30, 2) = false, want true")
	}
	if got := learn.BalanceAfter(100, 30, 2); got != 68 {
		t.Fatalf("BalanceAfter(100, 30, 2) = %d, want 68", got)
	}
	if learn.Allowed(100, 99, 2) {
		t.Fatal("Allowed(100, 99, 2) = true, want false (fee does not fit)")
	}
	if got := learn.BalanceAfter(100, 99, 2); got != 100 {
		t.Fatalf("rejected withdrawal changed the balance to %d, want 100", got)
	}
}

func TestExample5Refined(t *testing.T) {
	if got := learn.CheckedDivide(10, 2); got != 5 {
		t.Fatalf("CheckedDivide(10, 2) = %d, want 5", got)
	}
	if got := learn.CheckedDivide(10, 0); got != 0 {
		t.Fatalf("CheckedDivide(10, 0) = %d, want 0", got)
	}
	if got := learn.Percent(42); got != 42 {
		t.Fatalf("Percent(42) = %d, want 42", got)
	}
	if got := learn.Percent(250); got != 100 {
		t.Fatalf("Percent(250) = %d, want 100", got)
	}
	if got := learn.Slot(10, 20, 30, 1); got != 20 {
		t.Fatalf("Slot(..., 1) = %d, want 20", got)
	}
	if got := learn.Slot(10, 20, 30, 99); got != 10 {
		t.Fatalf("Slot(..., 99) = %d, want the fail-closed first slot (10)", got)
	}
}
