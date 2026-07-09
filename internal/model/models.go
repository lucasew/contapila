package model

import (
	"math/big"
	"time"
)

// Amount represents a quantity of a commodity.
type Amount struct {
	Value    *big.Rat
	Currency string
}

// Price represents a market price at a specific date.
type Price struct {
	Date      time.Time
	Commodity string
	Value     *big.Rat
	Target    string
}

// Directive is a marker interface for Beancount directives.
type Directive interface{}

// Option represents an 'option "name" "value"' directive.
type Option struct {
	Name  string
	Value string
}

// Include represents an 'include "path"' directive.
type Include struct {
	Path string
}

// PriceDirective represents a 'YYYY-MM-DD price ...' directive.
type PriceDirective struct {
	Price Price
}

// Transaction represents a Beancount transaction.
type Transaction struct {
	Date      time.Time
	Flag      string
	Payee     string
	Narration string
	Postings  []Posting
}

// Posting represents a single entry in a transaction.
type Posting struct {
	Account string
	Units   Amount
	Cost    *Amount // Cost basis e.g., {100.00 USD}
	Price   *Amount // Price e.g., @ 100.00 USD
}

// Position represents a held amount in an account with its average cost basis.
type Position struct {
	Account      string
	Units        *big.Rat
	Commodity    string
	AverageCost  *big.Rat // Average cost per unit
	CostCurrency string
}
