package assertutil

import "testing"

func TestAssertf_PanicsWhenFalse(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got none")
		}
		if r != "bad value: 42" {
			t.Fatalf("panic message = %v, want %q", r, "bad value: 42")
		}
	}()
	Assertf(false, "bad value: %d", 42)
}

func TestAssertf_NoPanicWhenTrue(t *testing.T) {
	Assertf(true, "unused: %d", 42)
}
