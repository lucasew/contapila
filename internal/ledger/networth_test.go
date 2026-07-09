package ledger

import (
	"contapila/internal/model"
	"contapila/internal/price"
	"math/big"
	"testing"
	"time"
)

func TestCalculateNetWorth(t *testing.T) {
	asOf, _ := time.Parse("2006-01-02", "2024-01-01")

	positions := []model.Position{
		{
			Commodity: "AAPL",
			Units:     big.NewRat(10, 1),
			AverageCost: big.NewRat(150, 1),
			CostCurrency: "USD",
		},
	}

	// Case 1: Price exists
	db1 := price.NewPriceDB([]model.Directive{
		&model.PriceDirective{Price: model.Price{Date: asOf, Commodity: "AAPL", Value: big.NewRat(160, 1), Target: "USD"}},
	})
	res1 := CalculateNetWorth(positions, db1, "USD", asOf)
	if res1.Total.FloatString(2) != "1600.00" {
		t.Errorf("expected 1600.00, got %s", res1.Total.FloatString(2))
	}

	// Case 2: Price missing, cost fallback
	db2 := price.NewPriceDB(nil)
	res2 := CalculateNetWorth(positions, db2, "USD", asOf)
	if res2.Total.FloatString(2) != "1500.00" {
		t.Errorf("expected 1500.00, got %s", res2.Total.FloatString(2))
	}
	if len(res2.Warnings) == 0 {
		t.Error("expected warning for cost fallback")
	}
}
