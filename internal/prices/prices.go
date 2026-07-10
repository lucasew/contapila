package prices

import (
	"math/big"
	"sort"
	"time"

	"github.com/lucasew/contapila-go/internal/ast"
	"github.com/lucasew/contapila-go/internal/diag"
	"github.com/lucasew/contapila-go/internal/loader"
)

// DB maps base commodity -> quote commodity -> sorted price points.
type DB struct {
	// key base|quote
	series map[string][]point
}

type point struct {
	Date  time.Time
	Rate  *big.Rat // quote per 1 base
}

func NewDB() *DB {
	return &DB{series: map[string][]point{}}
}

func key(base, quote string) string { return base + "|" + quote }

// LoadFile loads prices.beancount (includes expanded).
func LoadFile(path string) (*DB, diag.List, error) {
	db := NewDB()
	dirs, diags, err := loader.LoadFile(path)
	if err != nil {
		// missing file handled by caller
		return db, diags, err
	}
	for _, d := range dirs {
		if p, ok := d.(ast.Price); ok {
			db.Add(p.Currency, p.Amount.Commodity, p.Date, p.Amount.Number)
		}
	}
	return db, diags, nil
}

func (db *DB) Add(base, quote string, date time.Time, rate *big.Rat) {
	k := key(base, quote)
	// Last write wins for the same (base, quote, date).
	for i := range db.series[k] {
		if db.series[k][i].Date.Equal(date) {
			db.series[k][i].Rate = new(big.Rat).Set(rate)
			return
		}
	}
	db.series[k] = append(db.series[k], point{Date: date, Rate: new(big.Rat).Set(rate)})
	sort.Slice(db.series[k], func(i, j int) bool {
		return db.series[k][i].Date.Before(db.series[k][j].Date)
	})
}

// Rate returns quote per base on or before asOf. ok=false if none.
func (db *DB) Rate(base, quote string, asOf time.Time) (*big.Rat, time.Time, bool) {
	if base == quote {
		return big.NewRat(1, 1), asOf, true
	}
	pts := db.series[key(base, quote)]
	var best *point
	for i := range pts {
		if !pts[i].Date.After(asOf) {
			best = &pts[i]
		}
	}
	if best == nil {
		return nil, time.Time{}, false
	}
	return new(big.Rat).Set(best.Rate), best.Date, true
}
