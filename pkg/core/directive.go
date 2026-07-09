package core

import "time"

// Directive is the interface for all Beancount directives in the stream.
type Directive interface {
	GetDate() time.Time
}

// Note represents a 'note' directive.
type Note struct {
	Date    time.Time
	Account string
	Comment string
}

func (n Note) GetDate() time.Time { return n.Date }

// Event represents an 'event' directive.
type Event struct {
	Date        time.Time
	Type        string
	Description string
}

func (e Event) GetDate() time.Time { return e.Date }

// Transaction represents a Beancount transaction.
type Transaction struct {
	Date      time.Time
	Flag      string
	Payee     string
	Narration string
	// Postings will be added in a separate PR
}

func (t Transaction) GetDate() time.Time { return t.Date }
