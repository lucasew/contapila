package adapter

import (
	"errors"
	"fmt"

	"github.com/contapila/contapila/internal/directive"
)

// ErrParserUnavailable is returned when the modernc tree-sitter Beancount grammar is not available.
var ErrParserUnavailable = errors.New("modernc tree-sitter Beancount grammar is not yet available")

// Severity represents the importance of a Diagnostic.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarn
	SeverityError
)

// Diagnostic represents a message from the parser (warning or error).
type Diagnostic struct {
	Message  string
	Severity Severity
	Position directive.Position
}

func (d Diagnostic) String() string {
	var sev string
	switch d.Severity {
	case SeverityInfo:
		sev = "INFO"
	case SeverityWarn:
		sev = "WARN"
	case SeverityError:
		sev = "ERROR"
	}
	return fmt.Sprintf("%s [%d:%d]: %s", sev, d.Position.Line, d.Position.Column, d.Message)
}

// Parse turns Beancount source text into an internal ordered list of directives.
func Parse(filename string, src []byte) ([]directive.Directive, []Diagnostic, error) {
	// HARD EXTERNAL DEPENDENCY: This work cannot complete until a usable Beancount
	// grammar is available from the modernc/ccgo-tree-sitter ecosystem.
	return nil, nil, ErrParserUnavailable
}
