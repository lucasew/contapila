package ledger

import (
	"fmt"
	"github.com/lucasew/contapila-go/internal/model"
	"github.com/lucasew/contapila-go/internal/price"
	"math/big"
	"time"
)

type NetWorthResult struct {
	Total    *big.Rat
	Currency string
	Warnings []string
}

func ConvertPosition(pos model.Position, db *price.PriceDB, targetCurrency string, asOf time.Time) (*big.Rat, bool, string) {
	if pos.Commodity == "" {
		return new(big.Rat), false, ""
	}
	p, ok := db.GetPrice(pos.Commodity, targetCurrency, asOf)
	if ok {
		return new(big.Rat).Mul(pos.Units, p), true, ""
	}
	if pos.AverageCost != nil && pos.CostCurrency == targetCurrency {
		return new(big.Rat).Mul(pos.Units, pos.AverageCost), false, fmt.Sprintf("No price for %s on or before %s; using cost fallback", pos.Commodity, asOf.Format("2006-01-02"))
	}
	return new(big.Rat), false, fmt.Sprintf("No price for %s on or before %s and no suitable cost fallback", pos.Commodity, asOf.Format("2006-01-02"))
}

func CalculateNetWorth(positions []model.Position, db *price.PriceDB, targetCurrency string, asOf time.Time) NetWorthResult {
	result := NetWorthResult{
		Total:    new(big.Rat),
		Currency: targetCurrency,
	}

	for _, pos := range positions {
		val, _, warning := ConvertPosition(pos, db, targetCurrency, asOf)
		result.Total.Add(result.Total, val)
		if warning != "" {
			result.Warnings = append(result.Warnings, warning)
		}
	}

	return result
}
