package engine

import (
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/lucasew/contapila-go/internal/period"
)

func TestPnLBarsArePerBinFlows(t *testing.T) {
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
		t.Fatalf("bars=%d", len(bars))
	}
	// Not a running total: Jan != Jan+Feb
	if bars[1].Income.Cmp(new(big.Rat).Add(bars[0].Income, bars[1].Income)) == 0 && bars[0].Income.Sign() != 0 {
		t.Fatal("looks cumulative")
	}
	// Feb should not equal full year
	sum := big.NewRat(0, 1)
	for _, b := range bars {
		sum.Add(sum, b.Income)
	}
	if bars[0].Income.Cmp(sum) == 0 && sum.Sign() != 0 {
		t.Fatal("first bar equals year sum — cumulative bug")
	}
}
