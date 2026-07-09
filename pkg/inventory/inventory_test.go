package inventory

import (
	"math/big"
	"testing"

	"github.com/lucasew/contapila/pkg/amount"
)

func TestModelA(t *testing.T) {
	inv := NewInventory()
	acc := "Assets:Broker"
	comm := "PETR4"
	brl := "BRL"

	// 1. Buy 10 X {10.00 BRL}
	err := inv.Buy(acc, amount.New(big.NewRat(10, 1), comm), amount.New(big.NewRat(10, 1), brl))
	if err != nil {
		t.Fatalf("Buy 1 failed: %v", err)
	}

	pos := inv.Accounts[acc][comm]
	if pos.Units.Cmp(big.NewRat(10, 1)) != 0 {
		t.Errorf("Expected 10 units, got %s", pos.Units.String())
	}
	if avg := pos.AverageCost(); avg.Num.Cmp(big.NewRat(10, 1)) != 0 {
		t.Errorf("Expected average cost 10, got %s", avg.Num.String())
	}

	// 2. Buy 10 X {20.00 BRL}
	err = inv.Buy(acc, amount.New(big.NewRat(10, 1), comm), amount.New(big.NewRat(20, 1), brl))
	if err != nil {
		t.Fatalf("Buy 2 failed: %v", err)
	}

	if pos.Units.Cmp(big.NewRat(20, 1)) != 0 {
		t.Errorf("Expected 20 units, got %s", pos.Units.String())
	}
	// Average should be (10*10 + 10*20) / 20 = 15
	if avg := pos.AverageCost(); avg.Num.Cmp(big.NewRat(15, 1)) != 0 {
		t.Errorf("Expected average cost 15, got %s", avg.Num.String())
	}

	// 3. Sell 5 X (should use average 15)
	reductionCost, err := inv.Sell(acc, amount.New(big.NewRat(-5, 1), comm), nil, big.NewRat(0, 1))
	if err != nil {
		t.Fatalf("Sell failed: %v", err)
	}
	if reductionCost.Num.Cmp(big.NewRat(75, 1)) != 0 {
		t.Errorf("Expected reduction cost 75, got %s", reductionCost.Num.String())
	}
	if pos.Units.Cmp(big.NewRat(15, 1)) != 0 {
		t.Errorf("Expected 15 units left, got %s", pos.Units.String())
	}
	if avg := pos.AverageCost(); avg.Num.Cmp(big.NewRat(15, 1)) != 0 {
		t.Errorf("Expected average cost 15 to remain, got %s", avg.Num.String())
	}

	// 4. Oversell
	_, err = inv.Sell(acc, amount.New(big.NewRat(-20, 1), comm), nil, big.NewRat(0, 1))
	if err == nil {
		t.Errorf("Expected error on oversell, got nil")
	}

	// 5. Mismatched explicit cost
	wrongCost := amount.New(big.NewRat(16, 1), brl)
	_, err = inv.Sell(acc, amount.New(big.NewRat(-1, 1), comm), &wrongCost, big.NewRat(1, 100))
	if err == nil {
		t.Errorf("Expected error on mismatched cost, got nil")
	}

	// 6. Correct explicit cost
	rightCost := amount.New(big.NewRat(15, 1), brl)
	_, err = inv.Sell(acc, amount.New(big.NewRat(-1, 1), comm), &rightCost, big.NewRat(1, 100))
	if err != nil {
		t.Errorf("Expected no error on correct explicit cost, got %v", err)
	}
}
