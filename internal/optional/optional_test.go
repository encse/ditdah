package optional

import "testing"

func TestValueDistinguishesAbsentFromZero(t *testing.T) {
	if !None[int]().IsNone() {
		t.Fatal("None().IsNone() = false")
	}

	value, present := Some(0).Get()
	if !present || value != 0 {
		t.Fatalf("Some(0).Get() = %d, %v; want 0, true", value, present)
	}
}
