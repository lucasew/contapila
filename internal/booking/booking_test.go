package booking

import (
	"testing"
)

func TestAverageCost(t *testing.T) {
	inv := NewInventory()

	// 1. Buy 10 AAPL at 150 USD
	u1, _ := ParseDecimal("10")
	c1, _ := ParseDecimal("1500")
	_, err := inv.Apply(Amount{u1, "AAPL"}, &Amount{c1, "USD"})
	if err != nil {
		t.Fatalf("Buy 1 failed: %v", err)
	}

	pos := inv["AAPL"]
	if avg := pos.AverageCost().String(); avg != "150.00000" {
		t.Errorf("Expected average cost 150.00000, got %s", avg)
	}

	// 2. Buy 10 more AAPL at 170 USD
	u2, _ := ParseDecimal("10")
	c2, _ := ParseDecimal("1700")
	_, err = inv.Apply(Amount{u2, "AAPL"}, &Amount{c2, "USD"})
	if err != nil {
		t.Fatalf("Buy 2 failed: %v", err)
	}

	if avg := pos.AverageCost().String(); avg != "160.00000" {
		t.Errorf("Expected average cost 160.00000, got %s", avg)
	}

	// 3. Sell 5 AAPL at 200 USD
	u3, _ := ParseDecimal("-5")
	c3, _ := ParseDecimal("1000") // 5 * 200
	gain, err := inv.Apply(Amount{u3, "AAPL"}, &Amount{c3, "USD"})
	if err != nil {
		t.Fatalf("Sell failed: %v", err)
	}

	// Expected gain: proceeds (1000) - cost basis (5 * 160 = 800) = 200
	if gain == nil || gain.Number.String() != "200.00000" {
		t.Errorf("Expected gain 200.00000, got %v", gain)
	}

	if pos.Units.String() != "15.00000" {
		t.Errorf("Expected 15 AAPL remaining, got %s", pos.Units)
	}

	if avg := pos.AverageCost().String(); avg != "160.00000" {
		t.Errorf("Expected average cost 160.00000 after partial sell, got %s", avg)
	}
}

func TestOversell(t *testing.T) {
	inv := NewInventory()
	u1, _ := ParseDecimal("10")
	c1, _ := ParseDecimal("1000")
	inv.Apply(Amount{u1, "AAPL"}, &Amount{c1, "USD"})

	u2, _ := ParseDecimal("-11")
	_, err := inv.Apply(Amount{u2, "AAPL"}, nil)
	if err == nil {
		t.Error("Expected error when selling more than available")
	}
}
