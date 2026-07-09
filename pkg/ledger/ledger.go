package ledger

import (
	"fmt"
	"math/big"

	"github.com/lucasew/contapila/pkg/amount"
	"github.com/lucasew/contapila/pkg/inventory"
)

type Posting struct {
	Account   string
	Units     *amount.Amount
	Cost      *amount.Amount // unit cost {}
	Price     *amount.Amount // unit proceeds @
	Total     *amount.Amount // total proceeds @@
}

type Transaction struct {
	Description string
	Postings    []*Posting
}

type Ledger struct {
	Inventory *inventory.Inventory
	Tolerance *big.Rat
	// Balances tracks the simple "units" balance for reporting.
	Balances map[string]map[string]*big.Rat // Account -> Commodity -> Total
}

func NewLedger() *Ledger {
	return &Ledger{
		Inventory: inventory.NewInventory(),
		Tolerance: big.NewRat(1, 100000), // Default tolerance
		Balances:  make(map[string]map[string]*big.Rat),
	}
}

func (l *Ledger) addBalance(account, commodity string, units *big.Rat) {
	if l.Balances[account] == nil {
		l.Balances[account] = make(map[string]*big.Rat)
	}
	if b, ok := l.Balances[account][commodity]; ok {
		b.Add(b, units)
	} else {
		l.Balances[account][commodity] = new(big.Rat).Set(units)
	}
}

func (l *Ledger) Process(txn *Transaction) error {
	var residualPosting *Posting
	var txnBalances = make(map[string]amount.Amount) // commodity -> amount (sum to be balanced)

	addTxnBalance := func(a amount.Amount) {
		if a.Commodity == "" {
			return
		}
		if b, ok := txnBalances[a.Commodity]; ok {
			var err error
			txnBalances[a.Commodity], err = b.Add(a)
			if err != nil {
				panic(err)
			}
		} else {
			txnBalances[a.Commodity] = a
		}
	}

	type pendingCommit struct {
		account   string
		commodity string
		units     *big.Rat
	}
	var pending []pendingCommit

	for _, p := range txn.Postings {
		if p.Units == nil {
			if residualPosting != nil {
				return fmt.Errorf("more than one residual posting in transaction")
			}
			residualPosting = p
			continue
		}

		units := *p.Units
		if units.Num.Sign() > 0 {
			// Increase (Buy)
			if p.Cost != nil {
				// Inventory Buy
				err := l.Inventory.Buy(p.Account, units, *p.Cost)
				if err != nil {
					return err
				}
				// Balancing value for a Buy is units * cost
				costVal := p.Cost.Mul(units.Num)
				addTxnBalance(costVal)
			} else {
				// Regular posting
				addTxnBalance(units)
			}
			pending = append(pending, pendingCommit{p.Account, units.Commodity, units.Num})
		} else if units.Num.Sign() < 0 {
			// Decrease (Sell)
			// Check if we have inventory of this commodity in this account
			posUnits := l.Inventory.GetUnits(p.Account, p.Units.Commodity)
			if posUnits.Sign() > 0 {
				// Inventory Sell
				costOfReduction, err := l.Inventory.Sell(p.Account, units, p.Cost, l.Tolerance)
				if err != nil {
					return err
				}

				// Balancing value for an inventory Sell:
				// If @/@@ provided, use that as the balancing amount.
				// Else use the cost of reduction.
				var proceeds *amount.Amount
				if p.Total != nil {
					proceeds = p.Total
				} else if p.Price != nil {
					pVal := p.Price.Mul(new(big.Rat).Abs(units.Num))
					proceeds = &pVal
				}

				if proceeds != nil {
					// Use proceeds for balancing the posting
					addTxnBalance(proceeds.Neg())
					// Record the realized gain: Proceeds - Cost
					// (A gain is a credit/negative in the balance)
					gain, err := proceeds.Sub(costOfReduction)
					if err != nil {
						return err
					}
					addTxnBalance(gain.Neg())
				} else {
					// No proceeds specified, balance at cost
					addTxnBalance(costOfReduction.Neg())
				}
			} else {
				// Regular posting
				addTxnBalance(units)
			}
			pending = append(pending, pendingCommit{p.Account, units.Commodity, units.Num})
		} else {
			// Zero units
			addTxnBalance(units)
			pending = append(pending, pendingCommit{p.Account, units.Commodity, units.Num})
		}
	}

	if residualPosting != nil {
		for commodity, b := range txnBalances {
			if !b.Zero() {
				// Residual absorbs the negative of the balance
				val := b.Neg()
				l.addBalance(residualPosting.Account, commodity, val.Num)
			}
		}
	} else {
		// Check that all balances are zero
		for _, b := range txnBalances {
			if !b.Zero() {
				// TODO: handle tolerance
				return fmt.Errorf("transaction is unbalanced: %s", b.String())
			}
		}
	}

	// Commit pending balances
	for _, p := range pending {
		l.addBalance(p.account, p.commodity, p.units)
	}

	return nil
}

func ptr[T any](v T) *T {
	return &v
}
