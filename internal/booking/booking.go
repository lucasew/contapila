package booking

import (
	"fmt"
	"math/big"
)

// Decimal is a wrapper around big.Rat for monetary values.
type Decimal struct {
	val *big.Rat
}

func NewDecimal(val *big.Rat) Decimal {
	d := Decimal{new(big.Rat)}
	if val != nil {
		d.val.Set(val)
	}
	return d
}

func NewDecimalFromInt(i int64) Decimal {
	return NewDecimal(new(big.Rat).SetInt64(i))
}

func (d Decimal) ensure() *big.Rat {
	if d.val == nil {
		return new(big.Rat)
	}
	return d.val
}

func (d Decimal) Add(other Decimal) Decimal {
	res := new(big.Rat).Add(d.ensure(), other.ensure())
	return NewDecimal(res)
}

func (d Decimal) Sub(other Decimal) Decimal {
	res := new(big.Rat).Sub(d.ensure(), other.ensure())
	return NewDecimal(res)
}

func (d Decimal) Mul(other Decimal) Decimal {
	res := new(big.Rat).Mul(d.ensure(), other.ensure())
	return NewDecimal(res)
}

func (d Decimal) Quo(other Decimal) Decimal {
	res := new(big.Rat).Quo(d.ensure(), other.ensure())
	return NewDecimal(res)
}

func (d Decimal) Neg() Decimal {
	res := new(big.Rat).Neg(d.ensure())
	return NewDecimal(res)
}

func (d Decimal) Abs() Decimal {
	res := new(big.Rat).Abs(d.ensure())
	return NewDecimal(res)
}

func (d Decimal) Sign() int {
	return d.ensure().Sign()
}

func (d Decimal) IsZero() bool {
	return d.ensure().Sign() == 0
}

func (d Decimal) Cmp(other Decimal) int {
	return d.ensure().Cmp(other.ensure())
}

func (d Decimal) String() string {
	if d.val == nil {
		return "0.00000"
	}
	return d.val.FloatString(5) // Default precision 5 as per SPEC §6.5
}

func (d Decimal) Rat() *big.Rat {
	return new(big.Rat).Set(d.ensure())
}

func ParseDecimal(s string) (Decimal, error) {
	r := new(big.Rat)
	_, ok := r.SetString(s)
	if !ok {
		return Decimal{}, fmt.Errorf("invalid decimal: %s", s)
	}
	return NewDecimal(r), nil
}

// Amount is a value and a commodity.
type Amount struct {
	Number    Decimal
	Commodity string
}

func (a Amount) String() string {
	return a.Number.String() + " " + a.Commodity
}

// Position tracks a single merged average-cost position for one commodity in an account.
type Position struct {
	Units         Decimal
	TotalCost     Decimal
	CostCommodity string
}

func (p *Position) AverageCost() Decimal {
	if p.Units.IsZero() {
		return NewDecimal(nil)
	}
	return p.TotalCost.Quo(p.Units.Abs())
}

// Inventory maps commodity to its Position in an account.
type Inventory map[string]*Position

func NewInventory() Inventory {
	return make(Inventory)
}

// Apply handles an increase or reduction in the inventory.
func (inv Inventory) Apply(units Amount, cost *Amount) (gain *Amount, err error) {
	pos, ok := inv[units.Commodity]
	if !ok {
		pos = &Position{
			Units:     NewDecimal(nil),
			TotalCost: NewDecimal(nil),
		}
		inv[units.Commodity] = pos
	}

	// Increase (Buy)
	if units.Number.Sign() > 0 {
		if cost == nil {
			return nil, fmt.Errorf("inventory increase requires explicit cost")
		}
		if pos.Units.Sign() < 0 {
			return nil, fmt.Errorf("mixing short and long positions not supported in MVP")
		}
		pos.Units = pos.Units.Add(units.Number)
		pos.TotalCost = pos.TotalCost.Add(cost.Number)
		pos.CostCommodity = cost.Commodity
		return nil, nil
	}

	// Reduction (Sell)
	if units.Number.Sign() < 0 {
		if pos.Units.Sign() > 0 && pos.Units.Abs().Cmp(units.Number.Abs()) < 0 {
			return nil, fmt.Errorf("over-sell: selling %s but only have %s", units.Number.Abs(), pos.Units)
		}

		if cost == nil {
			pos.Units = pos.Units.Add(units.Number)
			if pos.Units.IsZero() {
				delete(inv, units.Commodity)
			}
			return nil, nil
		}

		// Reduction with cost
		avg := pos.AverageCost()
		reductionUnits := units.Number.Abs()
		reductionCost := reductionUnits.Mul(avg)

		pos.Units = pos.Units.Add(units.Number)
		pos.TotalCost = pos.TotalCost.Sub(reductionCost)

		realizedGain := cost.Number.Sub(reductionCost)
		gain = &Amount{
			Number:    realizedGain,
			Commodity: cost.Commodity,
		}

		if pos.Units.IsZero() {
			delete(inv, units.Commodity)
		}

		return gain, nil
	}

	return nil, nil
}
