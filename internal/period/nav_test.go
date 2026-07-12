package period

import (
	"testing"
	"time"
)

func TestDisplayLabel(t *testing.T) {
	now := d("2024-07-15")
	if g := DisplayLabel("2024-03", now); g != "March 2024" {
		t.Fatalf("got %q", g)
	}
	if g := DisplayLabel("", now); g != "All time" {
		t.Fatalf("got %q", g)
	}
}

func TestKind(t *testing.T) {
	now := time.Now()
	if Kind("2024-03", now) != KindMonth {
		t.Fatal()
	}
	if Kind("month-1", now) != KindMonth {
		t.Fatal()
	}
}
