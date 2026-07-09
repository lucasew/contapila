package ledger

import (
	"github.com/lucasew/contapila-go/internal/model"
	"github.com/lucasew/contapila-go/internal/price"
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

	db1 := price.NewPriceDB([]model.Directive{
		&model.PriceDirective{Price: model.Price{Date: asOf, Commodity: "AAPL", Value: big.NewRat(160, 1), Target: "USD"}},
	})
	res1 := CalculateNetWorth(positions, db1, "USD", asOf)
	if res1.Total.FloatString(2) != "1600.00" {
		t.Errorf("expected 1600.00, got %s", res1.Total.FloatString(2))
	}

	db2 := price.NewPriceDB(nil)
	res2 := CalculateNetWorth(positions, db2, "USD", asOf)
	if res2.Total.FloatString(2) != "1500.00" {
		t.Errorf("expected 1500.00, got %s", res2.Total.FloatString(2))
	}
	if len(res2.Warnings) == 0 {
		t.Error("expected warning for cost fallback")
	}
}

func TestResolveOperatingCurrency(t *testing.T) {
	l := &Ledger{
		Directives: []model.Directive{
			&model.Transaction{
				Postings: []model.Posting{
					{Units: model.Amount{Value: big.NewRat(10, 1), Currency: "USD"}},
				},
			},
		},
	}
	curr, explicit := l.ResolveOperatingCurrency(nil)
	if curr != "USD" || explicit {
		t.Errorf("expected inferred USD, got %s, %v", curr, explicit)
	}

	curr, explicit = l.ResolveOperatingCurrency([]string{"BRL"})
	if curr != "BRL" || !explicit {
		t.Errorf("expected explicit BRL, got %s, %v", curr, explicit)
	}
}
