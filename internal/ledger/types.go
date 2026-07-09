package ledger

import (
	"math/big"
	"time"
)

type Amount struct {
	Number    *big.Rat
	Commodity string
}

func NewAmount(number *big.Rat, commodity string) Amount {
	return Amount{
		Number:    new(big.Rat).Set(number),
		Commodity: commodity,
	}
}

type Posting struct {
	Account string
	Amount  *Amount // nil if residual
}

type Directive interface {
	GetDate() time.Time
}

type Open struct {
	Date        time.Time
	Account     string
	Commodities []string
}

func (o Open) GetDate() time.Time { return o.Date }

type Close struct {
	Date    time.Time
	Account string
}

func (c Close) GetDate() time.Time { return c.Date }

type Transaction struct {
	Date      time.Time
	Flag      string
	Payee     string
	Narration string
	Postings  []Posting
}

func (t Transaction) GetDate() time.Time { return t.Date }
