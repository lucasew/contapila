package main

import (
	"github.com/lucasew/contapila-go/internal/booking"
	"github.com/lucasew/contapila-go/internal/parser"
	"os"
	"testing"
)

func TestGolden(t *testing.T) {
	tests := []struct {
		filename    string
		wantError   bool
		wantWarning bool
	}{
		{"testdata/pass.beancount", false, false},
		{"testdata/fail_unbalanced.beancount", true, false},
		{"testdata/warn_unopened.beancount", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			f, err := os.Open(tt.filename)
			if err != nil {
				t.Fatalf("failed to open %s: %v", tt.filename, err)
			}
			defer f.Close()

			directives, err := parser.Parse(f)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", tt.filename, err)
			}

			booker := booking.NewBooker()
			booker.Book(directives)

			hasError := false
			hasWarning := false
			for _, diag := range booker.Diagnostics {
				if diag.Severity == booking.Error {
					hasError = true
				} else if diag.Severity == booking.Warning {
					hasWarning = true
				}
			}

			if hasError != tt.wantError {
				t.Errorf("got error=%v, want %v. Diags: %v", hasError, tt.wantError, booker.Diagnostics)
			}
			if hasWarning != tt.wantWarning {
				t.Errorf("got warning=%v, want %v. Diags: %v", hasWarning, tt.wantWarning, booker.Diagnostics)
			}
		})
	}
}
