package ledger

import (
	"math/big"
	"time"
)

// Amount represents a quantity of a commodity.
type Amount struct {
	Number    *big.Rat
	Commodity string
}

// NewAmount creates a new Amount with a copy of the given number.
func NewAmount(number *big.Rat, commodity string) Amount {
	return Amount{
		Number:    new(big.Rat).Set(number),
		Commodity: commodity,
	}
}

// Posting represents a single line in a transaction.
type Posting struct {
	Account string
	Amount  *Amount // nil if residual (to be filled by booking)
}

// Directive is the interface for all ledger entries.
type Directive interface {
	GetDate() time.Time
}

// Open represents an account opening.
type Open struct {
	Date        time.Time
	Account     string
	Commodities []string
}

func (o Open) GetDate() time.Time { return o.Date }

// Close represents an account closing.
type Close struct {
	Date    time.Time
	Account string
}

func (c Close) GetDate() time.Time { return c.Date }

// Transaction represents a financial transaction.
type Transaction struct {
	Date      time.Time
	Flag      string
	Payee     string
	Narration string
	Postings  []Posting
}

func (t Transaction) GetDate() time.Time { return t.Date }
