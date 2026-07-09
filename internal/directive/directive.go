package directive

import (
	"math/big"
	"time"
)

// Directive is the interface for all Beancount directives.
type Directive interface {
	directive()
}

// Position represents a location in a source file.
type Position struct {
	Line   int
	Column int
}

// Common fields for directives.
type meta struct {
	Position Position
}

func (m meta) directive() {}

// Option represents an 'option' directive.
type Option struct {
	meta
	Name  string
	Value string
}

// Include represents an 'include' directive.
type Include struct {
	meta
	Path string
}

// Commodity represents a 'commodity' directive.
type Commodity struct {
	meta
	Date     time.Time
	Currency string
}

// Open represents an 'open' directive.
type Open struct {
	meta
	Date       time.Time
	Account    string
	Currencies []string
}

// Close represents a 'close' directive.
type Close struct {
	meta
	Date    time.Time
	Account string
}

// Transaction represents a transaction directive.
type Transaction struct {
	meta
	Date      time.Time
	Flag      string // "*" or "!"
	Payee     string
	Narration string
	Postings  []Posting
}

// Posting represents a single entry in a transaction.
type Posting struct {
	Account string
	Amount  *Amount // Can be nil for residual leg
	Cost    *Amount // Optional
	Price   *Price  // Optional
}

// Amount represents a value and its commodity.
type Amount struct {
	Number    *big.Rat
	Commodity string
}

// PriceType indicates whether a price is per-unit or total.
type PriceType int

const (
	PriceTypeUnit  PriceType = iota // @
	PriceTypeTotal                  // @@
)

// Price represents an optional price on a posting.
type Price struct {
	Amount Amount
	Type   PriceType
}

// MarketPrice represents a 'price' directive.
type MarketPrice struct {
	meta
	Date      time.Time
	Commodity string
	Amount    Amount
}

// Balance represents a 'balance' directive.
type Balance struct {
	meta
	Date    time.Time
	Account string
	Amount  Amount
}

// Pad represents a 'pad' directive.
type Pad struct {
	meta
	Date      time.Time
	Account   string
	SourceAcc string
}

// Note represents a 'note' directive.
type Note struct {
	meta
	Date    time.Time
	Account string
	Comment string
}

// Event represents an 'event' directive.
type Event struct {
	meta
	Date  time.Time
	Type  string
	Value string
}

// Unknown represents an unsupported or unrecognized directive.
type Unknown struct {
	meta
	Type string
	Text string
}
