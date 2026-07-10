package config

import (
	"embed"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

//go:embed prelude.cue
var fs embed.FS

const PreludeFilename = "prelude.cue"

// Ledger is one filesystem-discovered ledger injected into CUE as ledgers.<name>.
type Ledger struct {
	Name string // directory name
	Main string // absolute path to main.beancount
}

type Config struct {
	Value cue.Value
}

var ledgerIDRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

// Load unifies embedded prelude ⊔ generated ledgers map ⊔ user contapila.cue.
// discovered comes from scanning <root>/*/main.beancount (host lookup).
func Load(userCue []byte, userFilename string, discovered []Ledger) (*Config, error) {
	ctx := cuecontext.New()

	preludeBytes, err := fs.ReadFile(PreludeFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded prelude: %w", err)
	}

	prelude := ctx.CompileBytes(preludeBytes, cue.Filename(PreludeFilename))
	if err := prelude.Err(); err != nil {
		return nil, fmt.Errorf("failed to compile prelude: %w", err)
	}

	gen := ctx.CompileString(encodeLedgersCUE(discovered), cue.Filename("ledgers.gen.cue"))
	if err := gen.Err(); err != nil {
		return nil, fmt.Errorf("failed to compile discovered ledgers: %w", err)
	}

	user := ctx.CompileBytes(userCue, cue.Filename(userFilename))
	if err := user.Err(); err != nil {
		return nil, fmt.Errorf("failed to compile user config: %w", err)
	}

	unified := prelude.Unify(gen).Unify(user)
	if err := unified.Validate(); err != nil {
		return nil, fmt.Errorf("config unification failed: %w", err)
	}

	return &Config{Value: unified}, nil
}

// encodeLedgersCUE builds a closed ledgers struct from filesystem discovery.
// Example:
//
//	ledgers: close({
//		personal: {name: "personal", main: "/abs/personal/main.beancount"}
//	})
func encodeLedgersCUE(discovered []Ledger) string {
	// stable key order
	names := make([]string, 0, len(discovered))
	byName := make(map[string]Ledger, len(discovered))
	for _, l := range discovered {
		if l.Name == "" {
			continue
		}
		if _, ok := byName[l.Name]; ok {
			continue
		}
		byName[l.Name] = l
		names = append(names, l.Name)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString("// Generated from filesystem lookup of */main.beancount — do not edit.\n")
	b.WriteString("ledgers: close({\n")
	for _, name := range names {
		l := byName[name]
		b.WriteString("\t")
		b.WriteString(cueLabel(name))
		b.WriteString(": {\n")
		b.WriteString("\t\tname: ")
		b.WriteString(strconv.Quote(l.Name))
		b.WriteString("\n\t\tmain: ")
		b.WriteString(strconv.Quote(l.Main))
		b.WriteString("\n\t}\n")
	}
	b.WriteString("})\n")
	return b.String()
}

func cueLabel(name string) string {
	if ledgerIDRe.MatchString(name) {
		return name
	}
	return strconv.Quote(name)
}
