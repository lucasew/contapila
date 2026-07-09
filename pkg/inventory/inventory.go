package inventory

import (
	"fmt"
	"math/big"

	"github.com/lucasew/contapila/pkg/amount"
)

type Position struct {
	Units     *big.Rat
	TotalCost amount.Amount // In cost commodity
}

func (p *Position) AverageCost() amount.Amount {
	if p.Units.Sign() == 0 {
		return amount.Amount{Num: big.NewRat(0, 1), Commodity: p.TotalCost.Commodity}
	}
	return p.TotalCost.Div(p.Units)
}

type Inventory struct {
	// Account name -> Commodity -> Position
	Accounts map[string]map[string]*Position
}

func NewInventory() *Inventory {
	return &Inventory{
		Accounts: make(map[string]map[string]*Position),
	}
}

func (inv *Inventory) getPosition(account, commodity string) *Position {
	if inv.Accounts[account] == nil {
		inv.Accounts[account] = make(map[string]*Position)
	}
	pos := inv.Accounts[account][commodity]
	if pos == nil {
		pos = &Position{
			Units: big.NewRat(0, 1),
		}
		inv.Accounts[account][commodity] = pos
	}
	return pos
}

func (inv *Inventory) GetUnits(account, commodity string) *big.Rat {
	acc, ok := inv.Accounts[account]
	if !ok {
		return big.NewRat(0, 1)
	}
	pos, ok := acc[commodity]
	if !ok {
		return big.NewRat(0, 1)
	}
	return new(big.Rat).Set(pos.Units)
}

func (inv *Inventory) GetTotalCost(account, commodity string) amount.Amount {
	acc, ok := inv.Accounts[account]
	if !ok {
		return amount.Amount{Num: big.NewRat(0, 1)}
	}
	pos, ok := acc[commodity]
	if !ok {
		return amount.Amount{Num: big.NewRat(0, 1)}
	}
	return pos.TotalCost.Clone()
}

func (inv *Inventory) Buy(account string, units amount.Amount, unitCost amount.Amount) error {
	if units.Num.Sign() <= 0 {
		return fmt.Errorf("buy units must be positive, got %s", units.Num.String())
	}
	pos := inv.getPosition(account, units.Commodity)

	if pos.Units.Sign() != 0 && pos.TotalCost.Commodity != unitCost.Commodity {
		return fmt.Errorf("cost commodity mismatch: existing %s, new %s", pos.TotalCost.Commodity, unitCost.Commodity)
	}

	totalBuyCost := unitCost.Mul(units.Num)

	if pos.Units.Sign() == 0 {
		pos.TotalCost = totalBuyCost
	} else {
		var err error
		pos.TotalCost, err = pos.TotalCost.Add(totalBuyCost)
		if err != nil {
			return err
		}
	}
	pos.Units.Add(pos.Units, units.Num)

	return nil
}

func (inv *Inventory) Sell(account string, units amount.Amount, explicitUnitCost *amount.Amount, tolerance *big.Rat) (amount.Amount, error) {
	if units.Num.Sign() >= 0 {
		return amount.Amount{}, fmt.Errorf("sell units must be negative, got %s", units.Num.String())
	}
	absUnits := new(big.Rat).Abs(units.Num)

	pos := inv.getPosition(account, units.Commodity)

	if pos.Units.Cmp(absUnits) < 0 {
		return amount.Amount{}, fmt.Errorf("oversell in %s: have %s, want to sell %s", account, pos.Units.String(), absUnits.String())
	}

	avgCost := pos.AverageCost()

	if explicitUnitCost != nil {
		if !avgCost.EqualWithTolerance(*explicitUnitCost, tolerance) {
			return amount.Amount{}, fmt.Errorf("explicit cost %s does not match average cost %s", explicitUnitCost.String(), avgCost.String())
		}
	}

	reductionCost := avgCost.Mul(absUnits)

	// Update inventory
	pos.Units.Sub(pos.Units, absUnits)
	if pos.Units.Sign() == 0 {
		pos.TotalCost = amount.Amount{Num: big.NewRat(0, 1), Commodity: avgCost.Commodity}
	} else {
		var err error
		pos.TotalCost, err = pos.TotalCost.Sub(reductionCost)
		if err != nil {
			return amount.Amount{}, err
		}
	}

	return reductionCost, nil
}
