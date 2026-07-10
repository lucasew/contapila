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
	Account    string
	Currencies []string // optional commodities declared on open (e.g. open Assets:Cash BRL)
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

// Document is a Beancount document directive, or one synthesized from <ledger>/docs/by-account.
type Document struct {
	Meta
	Account   string
	Path      string // project-relative (e.g. personal/docs/by-account/Assets/Cash/20240101_x.txt)
	Synthetic bool   // true when expanded from the docs/ tree at runtime
}

type Unknown struct {
	Meta
	Kind string
	Text string
}
