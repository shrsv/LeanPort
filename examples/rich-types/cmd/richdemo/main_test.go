package main

import (
	"bytes"
	"testing"

	"example.com/richtypes/internal/verified/rich"
)

func TestGreet(t *testing.T) {
	if got, want := rich.Greet("world"), "Hello, world!"; got != want {
		t.Fatalf("Greet = %q, want %q", got, want)
	}
}

func TestSum(t *testing.T) {
	if got, want := rich.Sum([]byte{1, 2, 3, 250}), uint64(256); got != want {
		t.Fatalf("Sum = %d, want %d", got, want)
	}
	if got := rich.Sum(nil); got != 0 {
		t.Fatalf("Sum(nil) = %d, want 0", got)
	}
}

func TestReverse(t *testing.T) {
	if got, want := rich.Reverse([]byte("leanport")), []byte("tropnael"); !bytes.Equal(got, want) {
		t.Fatalf("Reverse = %q, want %q", got, want)
	}
	if got := rich.Reverse(nil); len(got) != 0 {
		t.Fatalf("Reverse(nil) = %q, want empty", got)
	}
}
