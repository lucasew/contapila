package ledger

import (
	"contapila/internal/model"
	"math/big"
	"testing"
	"time"
)

func TestResolveOperatingCurrency(t *testing.T) {
	l1 := &Ledger{
		Directives: []model.Directive{
			&model.Option{Name: "operating_currency", Value: "BRL"},
		},
	}
	curr, explicit := l1.ResolveOperatingCurrency()
	if curr != "BRL" || !explicit {
		t.Errorf("expected explicit BRL, got %s, %v", curr, explicit)
	}

	l2 := &Ledger{
		Directives: []model.Directive{
			&model.Transaction{
				Postings: []model.Posting{
					{Units: model.Amount{Value: big.NewRat(10, 1), Currency: "USD"}},
				},
			},
		},
	}
	curr, explicit = l2.ResolveOperatingCurrency()
	if curr != "USD" || explicit {
		t.Errorf("expected inferred USD, got %s, %v", curr, explicit)
	}
}

func TestGetPositions(t *testing.T) {
	d1, _ := time.Parse("2006-01-02", "2024-01-01")
	l := &Ledger{
		Directives: []model.Directive{
			&model.Transaction{
				Date: d1,
				Postings: []model.Posting{
					{
						Account: "Assets:Broker",
						Units:   model.Amount{Value: big.NewRat(10, 1), Currency: "AAPL"},
						Cost:    &model.Amount{Value: big.NewRat(150, 1), Currency: "USD"},
					},
				},
			},
		},
	}

	positions := l.GetPositions(d1)
	if len(positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(positions))
	}
	pos := positions[0]
	if pos.Units.FloatString(0) != "10" || pos.AverageCost.FloatString(0) != "150" {
		t.Errorf("unexpected position: %+v", pos)
	}
}
