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
	Date      time.Time // optional cost date → synthetic price (zero if omitted)
}

type PriceSpec struct {
	Number    *big.Rat
	Commodity string
	Total     bool // @@ when true, @ when false
}

type Posting struct {
	Account  string
	Units    *Amount // nil => residual
	Cost     *CostSpec
	Price    *PriceSpec
	Metadata Metadata // key_value under the posting (not CUE — journal stream only)
}

type Directive interface {
	GetDate() time.Time
	GetFile() string
}

type Meta struct {
	Date time.Time
	File string
	Line int // 1-based source line; 0 if unknown
}

func (m Meta) GetDate() time.Time { return m.Date }
func (m Meta) GetFile() string    { return m.File }
func (m Meta) GetLine() int       { return m.Line }

type Option struct {
	Meta
	Key, Value string
}

type Include struct {
	Meta
	Path string
}

// Metadata is Beancount key: value attributes on a directive (strings normalized).
// Keys are stored as written (e.g. "asset-class", "institution").
type Metadata map[string]string

type Commodity struct {
	Meta
	Currency string
	Metadata Metadata // from key_value under the commodity directive
}

type Open struct {
	Meta
	Account    string
	Currencies []string // optional commodities declared on open (e.g. open Assets:Cash BRL)
	Metadata   Metadata // from key_value under the open directive
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
	Metadata  Metadata // key_value under the txn header (not CUE — journal stream only)
}

type Price struct {
	Meta
	Currency string   // base commodity being priced
	Amount   Amount   // quote amount (Number + quote Commodity)
	Metadata Metadata // key_value under the price directive
}

type Balance struct {
	Meta
	Account  string
	Amount   Amount
	Metadata Metadata // key_value under the balance directive (journal stream only)
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
	Type     string
	Desc     string
	Metadata Metadata // key_value under the event directive (journal stream only)
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
