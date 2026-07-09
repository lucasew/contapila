package diag

import "fmt"

type Severity int

const (
	Warn Severity = iota
	Error
)

func (s Severity) String() string {
	if s == Error {
		return "error"
	}
	return "warn"
}

type Diagnostic struct {
	Severity Severity
	Message  string
	File     string
	Line     int
}

func (d Diagnostic) IsError() bool { return d.Severity == Error }
func (d Diagnostic) IsWarn() bool  { return d.Severity == Warn }

func (d Diagnostic) String() string {
	loc := d.File
	if d.Line > 0 {
		loc = fmt.Sprintf("%s:%d", d.File, d.Line)
	}
	if loc == "" {
		return fmt.Sprintf("%s: %s", d.Severity, d.Message)
	}
	return fmt.Sprintf("%s: %s: %s", loc, d.Severity, d.Message)
}

type List []Diagnostic

func (l List) HasErrors() bool {
	for _, d := range l {
		if d.Severity == Error {
			return true
		}
	}
	return false
}

func (l *List) Warn(file, msg string) {
	*l = append(*l, Diagnostic{Severity: Warn, Message: msg, File: file})
}

func (l *List) Error(file, msg string) {
	*l = append(*l, Diagnostic{Severity: Error, Message: msg, File: file})
}

func (l *List) Merge(other List) {
	*l = append(*l, other...)
}
