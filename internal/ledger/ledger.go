package ledger

import (
	"github.com/lucasew/contapila-go/internal/model"
	"math/big"
	"time"
)

type Ledger struct {
	Name       string
	Directives []model.Directive
}

func (l *Ledger) ResolveOperatingCurrency(cueCurrencies []string) (string, bool) {
	if len(cueCurrencies) > 0 {
		return cueCurrencies[0], true
	}

	for _, d := range l.Directives {
		if txn, ok := d.(*model.Transaction); ok {
			for _, p := range txn.Postings {
				if p.Units.Currency != "" && p.Units.Value != nil {
					return p.Units.Currency, false
				}
			}
		}
	}

	return "", false
}

func (l *Ledger) GetPositions(asOf time.Time) []model.Position {
	type key struct {
		account   string
		commodity string
	}
	positions := make(map[key]*model.Position)

	for _, d := range l.Directives {
		txn, ok := d.(*model.Transaction)
		if !ok || txn.Date.After(asOf) {
			continue
		}

		missingIdx := -1
		totalCost := make(map[string]*big.Rat)

		for i, p := range txn.Postings {
			if p.Units.Value == nil {
				if missingIdx == -1 {
					missingIdx = i
				}
				continue
			}
			cost := p.Units.Value
			if p.Cost != nil {
				cost = new(big.Rat).Mul(p.Units.Value, p.Cost.Value)
				curr := p.Cost.Currency
				if _, ok := totalCost[curr]; !ok {
					totalCost[curr] = new(big.Rat)
				}
				totalCost[curr].Add(totalCost[curr], cost)
			} else {
				curr := p.Units.Currency
				if _, ok := totalCost[curr]; !ok {
					totalCost[curr] = new(big.Rat)
				}
				totalCost[curr].Add(totalCost[curr], cost)
			}
		}

		for i, p := range txn.Postings {
			units := p.Units.Value
			currency := p.Units.Currency
			costVal := p.Cost

			if i == missingIdx {
				if len(totalCost) == 1 {
					for curr, total := range totalCost {
						units = new(big.Rat).Neg(total)
						currency = curr
					}
				} else {
					continue
				}
			}

			if units == nil {
				continue
			}

			k := key{p.Account, currency}
			pos, exists := positions[k]
			if !exists {
				pos = &model.Position{
					Account:   p.Account,
					Units:     new(big.Rat),
					Commodity: currency,
				}
				positions[k] = pos
			}

			if units.Sign() > 0 {
				if costVal != nil {
					newUnits := new(big.Rat).Add(pos.Units, units)
					if newUnits.Sign() != 0 {
						currentTotalCost := new(big.Rat)
						if pos.AverageCost != nil {
							currentTotalCost.Mul(pos.Units, pos.AverageCost)
						}
						addedTotalCost := new(big.Rat).Mul(units, costVal.Value)
						total := new(big.Rat).Add(currentTotalCost, addedTotalCost)
						pos.AverageCost = new(big.Rat).Quo(total, newUnits)
						pos.CostCurrency = costVal.Currency
					}
					pos.Units = newUnits
				} else {
					pos.Units.Add(pos.Units, units)
				}
			} else if units.Sign() < 0 {
				pos.Units.Add(pos.Units, units)
			}
		}
	}

	var result []model.Position
	for _, pos := range positions {
		if pos.Units.Sign() != 0 {
			result = append(result, *pos)
		}
	}
	return result
}
