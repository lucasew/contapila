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
	// Balances tracks the simple "amount" balance for reporting.
	// For inventory accounts, it tracks units.
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
	var costBalances = make(map[string]amount.Amount) // commodity -> amount (cost basis)

	addCostBalance := func(a amount.Amount) {
		if a.Commodity == "" {
			return
		}
		if b, ok := costBalances[a.Commodity]; ok {
			var err error
			costBalances[a.Commodity], err = b.Add(a)
			if err != nil {
				panic(err)
			}
		} else {
			costBalances[a.Commodity] = a
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
				// Balancing value is units * unit_cost (Cost Basis)
				costVal := p.Cost.Mul(units.Num)
				addCostBalance(costVal)
			} else {
				// Regular posting
				addCostBalance(units)
			}
			pending = append(pending, pendingCommit{p.Account, units.Commodity, units.Num})
		} else if units.Num.Sign() < 0 {
			// Decrease (Sell)
			// Check if it's an inventory account (has units of this commodity)
			posUnits := l.Inventory.GetUnits(p.Account, p.Units.Commodity)
			if posUnits.Sign() > 0 {
				// Inventory Sell
				costOfReduction, err := l.Inventory.Sell(p.Account, units, p.Cost, l.Tolerance)
				if err != nil {
					return err
				}
				// Balancing value for Cost Basis is the cost of reduction (negative)
				costVal := costOfReduction.Neg()
				addCostBalance(costVal)

				// Support @ and @@ proceeds
				if p.Price != nil || p.Total != nil {
					var proceeds amount.Amount
					if p.Total != nil {
						proceeds = *p.Total
					} else {
						proceeds = p.Price.Mul(new(big.Rat).Abs(units.Num))
					}
					// Gain = Proceeds - costOfReduction
					// Gain is a CREDIT (negative).
					gainAmount, err := proceeds.Sub(costOfReduction)
					if err != nil {
						return err
					}
					addCostBalance(gainAmount.Neg())
				}
			} else {
				// Regular posting
				addCostBalance(units)
			}
			pending = append(pending, pendingCommit{p.Account, units.Commodity, units.Num})
		} else {
			// Zero units
			addCostBalance(units)
			pending = append(pending, pendingCommit{p.Account, units.Commodity, units.Num})
		}
	}

	if residualPosting != nil {
		for commodity, b := range costBalances {
			if !b.Zero() {
				// Residual absorbs the negative of the balance
				val := b.Neg()
				l.addBalance(residualPosting.Account, commodity, val.Num)
			}
		}
	} else {
		// Check that all cost balances are zero
		for _, b := range costBalances {
			if !b.Zero() {
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
