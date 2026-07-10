package engine

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/lucasew/contapila-go/internal/period"
)

func TestExampleNetWorthSeries(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "example")
	p, pdb, _, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}
	l, err := OpenLedger(p, pdb, "personal")
	if err != nil {
		t.Fatal(err)
	}
	if l.OpCurrency == "" {
		t.Fatal("expected op currency")
	}
	pts, err := l.NetWorthSeries(time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) < 2 {
		t.Fatalf("expected multiple NW points, got %d", len(pts))
	}
	// monotonic dates
	for i := 1; i < len(pts); i++ {
		if !pts[i].Date.After(pts[i-1].Date) && !pts[i].Date.Equal(pts[i-1].Date) {
			t.Fatalf("dates not ordered")
		}
	}
}

// Net worth series must revalue on PriceDB days even without balance-changing txns.
func TestNetWorthSeriesIncludesPriceDays(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "example")
	p, pdb, _, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}
	l, err := OpenLedger(p, pdb, "personal")
	if err != nil {
		t.Fatal(err)
	}
	// 2023-07-04 has prices (USD, SPDW, BTC, …) in example prices.beancount.
	priceDay := time.Date(2023, 7, 4, 0, 0, 0, 0, time.UTC)
	// Window that may include fewer txns than price observations.
	from := time.Date(2023, 7, 3, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 7, 5, 0, 0, 0, 0, time.UTC)
	pts, err := l.NetWorthSeries(from, to)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, pt := range pts {
		if pt.Date.Equal(priceDay) {
			found = true
			if pt.Value == nil {
				t.Fatal("nil value on price day")
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected NW point on price day %s; got %d points", priceDay.Format("2006-01-02"), len(pts))
	}
}

func TestExamplePnLBarsYear(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "example")
	p, pdb, _, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}
	l, err := OpenLedger(p, pdb, "personal")
	if err != nil {
		t.Fatal(err)
	}
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	bars := l.PnLBars(from, to, period.BinMonth)
	if len(bars) != 12 {
		t.Fatalf("months=%d", len(bars))
	}
}
