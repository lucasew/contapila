package ast

import (
	"math/big"
	"time"
)

type Amount struct {
	Number    *big.Rat
	Commodity string
}

func (a Amount) Clone() Amount {
	if a.Number == nil {
		return Amount{Commodity: a.Commodity}
	}
	return Amount{Number: new(big.Rat).Set(a.Number), Commodity: a.Commodity}
}

type CostSpec struct {
	Number    *big.Rat // nil with Empty=true means {}
	Commodity string
	Empty     bool
}

type PriceSpec struct {
	Number    *big.Rat
	Commodity string
	Total     bool // @@ when true, @ when false
}

type Posting struct {
	Account string
	Units   *Amount // nil => residual
	Cost    *CostSpec
	Price   *PriceSpec
}

type Directive interface {
	GetDate() time.Time
	GetFile() string
}

type Meta struct {
	Date time.Time
	File string
}

func (m Meta) GetDate() time.Time { return m.Date }
func (m Meta) GetFile() string    { return m.File }

type Option struct {
	Meta
	Key, Value string
}

type Include struct {
	Meta
	Path string
}

type Commodity struct {
	Meta
	Currency string
}

type Open struct {
	Meta
	Account string
}

type Close struct {
	Meta
	Account string
}

type Transaction struct {
	Meta
	Flag      string
	Narration string
	Payee     string
	Postings  []Posting
	Tags      []string
	Links     []string
}

type Price struct {
	Meta
	Currency string
	Amount   Amount
}

type Balance struct {
	Meta
	Account string
	Amount  Amount
}

type Pad struct {
	Meta
	Account     string
	FromAccount string
}

type Note struct {
	Meta
	Account string
	Comment string
}

type Event struct {
	Meta
	Type string
	Desc string
}

type Unknown struct {
	Meta
	Kind string
	Text string
}
