package price

import (
	"github.com/lucasew/contapila-go/internal/model"
	"math/big"
	"sort"
	"time"
)

type PriceDB struct {
	prices map[string][]model.Price
}

func NewPriceDB(directives []model.Directive) *PriceDB {
	db := &PriceDB{
		prices: make(map[string][]model.Price),
	}
	for _, d := range directives {
		if pd, ok := d.(*model.PriceDirective); ok {
			key := pd.Price.Commodity + "|" + pd.Price.Target
			db.prices[key] = append(db.prices[key], pd.Price)
		}
	}
	for key := range db.prices {
		sort.Slice(db.prices[key], func(i, j int) bool {
			return db.prices[key][i].Date.Before(db.prices[key][j].Date)
		})
	}
	return db
}

func (db *PriceDB) GetPrice(commodity, target string, asOf time.Time) (*big.Rat, bool) {
	if commodity == target {
		return big.NewRat(1, 1), true
	}
	key := commodity + "|" + target
	prices, ok := db.prices[key]
	if !ok {
		return nil, false
	}

	var bestPrice *model.Price
	for i := len(prices) - 1; i >= 0; i-- {
		if !prices[i].Date.After(asOf) {
			bestPrice = &prices[i]
			break
		}
	}

	if bestPrice != nil {
		return bestPrice.Value, true
	}

	return nil, false
}
