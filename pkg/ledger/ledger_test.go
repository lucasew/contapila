package ledger

import (
	"math/big"
	"testing"

	"github.com/lucasew/contapila/pkg/amount"
)

func TestLedgerGolden(t *testing.T) {
	l := NewLedger()
	brl := "BRL"
	petr4 := "PETR4"
	vale3 := "VALE3"

	// Setup: Buy some PETR4 and VALE3
	// Assets:Broker:PETR4  100 PETR4 {30.00 BRL}
	txn1 := &Transaction{
		Description: "Buy PETR4",
		Postings: []*Posting{
			{
				Account: "Assets:Broker:PETR4",
				Units:   ptr(amount.New(big.NewRat(100, 1), petr4)),
				Cost:    ptr(amount.New(big.NewRat(30, 1), brl)),
			},
			{
				Account: "Assets:Cash",
				Units:   ptr(amount.New(big.NewRat(-3000, 1), brl)),
			},
		},
	}
	if err := l.Process(txn1); err != nil {
		t.Fatalf("Buy PETR4 failed: %v", err)
	}

	// Assets:Broker:VALE3   50 VALE3 {60.00 BRL}
	txn2 := &Transaction{
		Description: "Buy VALE3",
		Postings: []*Posting{
			{
				Account: "Assets:Broker:VALE3",
				Units:   ptr(amount.New(big.NewRat(50, 1), vale3)),
				Cost:    ptr(amount.New(big.NewRat(60, 1), brl)),
			},
			{
				Account: "Assets:Cash",
				Units:   ptr(amount.New(big.NewRat(-3000, 1), brl)),
			},
		},
	}
	if err := l.Process(txn2); err != nil {
		t.Fatalf("Buy VALE3 failed: %v", err)
	}

	// Sell example from SPEC §7.3
	// 2024-03-10 * "Sell PETR4 + VALE3"
	//   Assets:Broker:PETR4  -40 PETR4 @@ 1400.00 BRL
	//   Assets:Broker:VALE3  -20 VALE3 @@ 1400.00 BRL
	//   Assets:Cash           2800.00 BRL
	//   Income:Gains
	txnSell := &Transaction{
		Description: "Sell PETR4 + VALE3",
		Postings: []*Posting{
			{
				Account: "Assets:Broker:PETR4",
				Units:   ptr(amount.New(big.NewRat(-40, 1), petr4)),
				Total:   ptr(amount.New(big.NewRat(1400, 1), brl)), // @@ proceeds
			},
			{
				Account: "Assets:Broker:VALE3",
				Units:   ptr(amount.New(big.NewRat(-20, 1), vale3)),
				Total:   ptr(amount.New(big.NewRat(1400, 1), brl)), // @@ proceeds
			},
			{
				Account: "Assets:Cash",
				Units:   ptr(amount.New(big.NewRat(2800, 1), brl)),
			},
			{
				Account: "Income:Gains",
				Units:   nil, // Residual leg
			},
		},
	}

	if err := l.Process(txnSell); err != nil {
		t.Fatalf("Sell txn failed: %v", err)
	}

	// Verify Gains:
	// PETR4 cost = 40 * 30 = 1200. Proceeds = 1400. Gain = 200.
	// VALE3 cost = 20 * 60 = 1200. Proceeds = 1400. Gain = 200.
	// Total gain = 400. Income:Gains should be -400 BRL.
	gains := l.Balances["Income:Gains"][brl]
	if gains == nil {
		t.Logf("Balances for Income:Gains: %v", l.Balances["Income:Gains"])
		t.Fatalf("Expected -400 BRL in Income:Gains, got nil")
	}
	if gains.Cmp(big.NewRat(-400, 1)) != 0 {
		t.Errorf("Expected -400 BRL in Income:Gains, got %v", gains)
	}

	// Verify remaining inventory (Units)
	petr4Units := l.Balances["Assets:Broker:PETR4"][petr4]
	if petr4Units.Cmp(big.NewRat(60, 1)) != 0 {
		t.Errorf("Expected 60 PETR4 left, got %s", petr4Units.String())
	}
	vale3Units := l.Balances["Assets:Broker:VALE3"][vale3]
	if vale3Units.Cmp(big.NewRat(30, 1)) != 0 {
		t.Errorf("Expected 30 VALE3 left, got %s", vale3Units.String())
	}
}
