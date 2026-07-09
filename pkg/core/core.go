package core

import "time"

// Directive represents a Beancount directive.
type Directive interface {
	GetDate() time.Time
}

// Transaction represents a Beancount transaction.
type Transaction struct {
	Date      time.Time
	Flag      string
	Payee     string
	Narration string
	Postings  []Posting
}

func (t Transaction) GetDate() time.Time { return t.Date }

// Posting represents a single posting in a transaction.
type Posting struct {
	Account  string
	Amount   string
	Currency string
}

// Note represents a Beancount note directive.
type Note struct {
	Date    time.Time
	Account string
	Comment string
}

func (n Note) GetDate() time.Time { return n.Date }

// Event represents a Beancount event directive.
type Event struct {
	Date        time.Time
	Type        string
	Description string
}

func (e Event) GetDate() time.Time { return e.Date }
