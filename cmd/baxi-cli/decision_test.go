package main

import (
	"testing"
)

func TestStrPtr_NonEmpty(t *testing.T) {
	got := strPtr("hello")
	if got == nil {
		t.Fatal("strPtr(\"hello\") returned nil")
	}
	if *got != "hello" {
		t.Errorf("strPtr(\"hello\") = %q, want %q", *got, "hello")
	}
}

func TestStrPtr_Empty(t *testing.T) {
	got := strPtr("")
	if got != nil {
		t.Errorf("strPtr(\"\") = %v, want nil", *got)
	}
}
