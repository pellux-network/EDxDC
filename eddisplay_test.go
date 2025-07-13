package main

import (
	"testing"
)

// Sample test function
func TestHelloWorld(t *testing.T) {
	got := "Hello, world!"
	want := "Hello, world!"
	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}
