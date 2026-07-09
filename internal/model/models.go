package model

import (
	"math/big"
	"time"
)

type Amount struct {
	Value    *big.Rat
	Currency string
}

type Price struct {
	Date      time.Time
	Commodity string
	Value     *big.Rat
	Target    string
}

type Directive interface{}

type PriceDirective struct {
	Price Price
}

type Transaction struct {
	Date      time.Time
	Flag      string
	Payee     string
	Narration string
	Postings  []Posting
}

type Posting struct {
	Account string
	Units   Amount
	Cost    *Amount
}

type Position struct {
	Account      string
	Units        *big.Rat
	Commodity    string
	AverageCost  *big.Rat
	CostCurrency string
}
