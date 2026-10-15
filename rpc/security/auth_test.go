package security

import (
	"testing"
)

func TestRand(t *testing.T) {
	s := RandString(32)
	if len(s) == 0 {
		t.Fatal("RandString failed to generate random string")
	}
	t.Log("Generated RandString:", s)
}
