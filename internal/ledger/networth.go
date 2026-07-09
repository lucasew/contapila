package ledger

import (
	"contapila/internal/model"
	"contapila/internal/price"
	"fmt"
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
		return new(big.Rat), false, "" // Skip empty commodity (usually unbalanced residual legs we don't want in NW)
	}
	p, ok := db.GetPrice(pos.Commodity, targetCurrency, asOf)
	if ok {
		return new(big.Rat).Mul(pos.Units, p), true, ""
	}
	// Fallback to cost basis
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
		val, ok, warning := ConvertPosition(pos, db, targetCurrency, asOf)
		result.Total.Add(result.Total, val)
		if warning != "" {
			result.Warnings = append(result.Warnings, warning)
		}
		_ = ok
	}

	return result
}
