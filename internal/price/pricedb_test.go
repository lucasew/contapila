package price

import (
	"contapila/internal/model"
	"math/big"
	"testing"
	"time"
)

func TestPriceDB(t *testing.T) {
	d1, _ := time.Parse("2006-01-02", "2024-01-01")
	d2, _ := time.Parse("2006-01-02", "2024-01-03")

	directives := []model.Directive{
		&model.PriceDirective{Price: model.Price{Date: d1, Commodity: "AAPL", Value: big.NewRat(150, 1), Target: "USD"}},
		&model.PriceDirective{Price: model.Price{Date: d2, Commodity: "AAPL", Value: big.NewRat(160, 1), Target: "USD"}},
	}
	db := NewPriceDB(directives)

	tests := []struct {
		date string
		want string
		ok   bool
	}{
		{"2024-01-01", "150.00", true},
		{"2024-01-02", "150.00", true}, // walk back
		{"2024-01-03", "160.00", true},
		{"2024-01-04", "160.00", true}, // walk back
		{"2023-12-31", "", false},      // no future price
	}

	for _, tt := range tests {
		asOf, _ := time.Parse("2006-01-02", tt.date)
		got, ok := db.GetPrice("AAPL", "USD", asOf)
		if ok != tt.ok {
			t.Errorf("GetPrice at %s ok = %v, want %v", tt.date, ok, tt.ok)
		}
		if ok && got.FloatString(2) != tt.want {
			t.Errorf("GetPrice at %s = %s, want %s", tt.date, got.FloatString(2), tt.want)
		}
	}
}
