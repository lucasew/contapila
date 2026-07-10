package parser

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/lucasew/contapila-go/internal/ast"
	"github.com/lucasew/contapila-go/internal/diag"
	"github.com/modernc-tree-sitter/ccgo-tree-sitter/grammar"
	bc "github.com/modernc-tree-sitter/ccgo-tree-sitter/grammar/beancount"
)

// Parse converts Beancount source into directives using modernc tree-sitter.
func Parse(filename string, src []byte) ([]ast.Directive, diag.List, error) {
	p := grammar.NewParser()
	defer p.Delete()
	if !p.SetLanguage(bc.Language()) {
		return nil, nil, fmt.Errorf("failed to set beancount language")
	}
	tree := p.ParseString(string(src))
	defer tree.Delete()
	root := tree.RootNode()
	if root.IsNull() {
		return nil, nil, fmt.Errorf("null parse tree for %s", filename)
	}

	// One O(n) newline scan; each line lookup is O(log lines) binary search.
	// Avoids rescanning the file for every directive/metadata key.
	lines := newLineTable(src)

	var diags diag.List
	var out []ast.Directive
	for i := uint32(0); i < root.NamedChildCount(); i++ {
		ch := root.NamedChild(i)
		if ch.IsError() {
			diags.Error(filename, lines.At(ch.StartByte()), fmt.Sprintf("syntax error near %q", clip(slice(src, ch), 40)))
			continue
		}
		if d, ok := convert(filename, src, ch, &diags, lines); ok {
			out = append(out, d)
		}
	}
	return out, diags, nil
}

func convert(file string, src []byte, n *grammar.Node, diags *diag.List, lines lineTable) (ast.Directive, bool) {
	switch n.Type() {
	case "option":
		return ast.Option{Meta: meta(file, src, n, lines), Key: unquote(textField(src, n, "key")), Value: unquote(textField(src, n, "value"))}, true
	case "include":
		// include has a string child, not always a field
		path := ""
		for i := uint32(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(i)
			if c.Type() == "string" {
				path = unquote(slice(src, c))
				break
			}
		}
		return ast.Include{Meta: meta(file, src, n, lines), Path: path}, true
	case "commodity":
		o := ast.Commodity{
			Meta:     meta(file, src, n, lines),
			Currency: textField(src, n, "currency"),
			Metadata: parseKeyValues(src, n),
		}
		return o, true
	case "open":
		o := ast.Open{
			Meta:     meta(file, src, n, lines),
			Account:  textField(src, n, "account"),
			Metadata: parseKeyValues(src, n),
		}
		for i := uint32(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(i)
			if c.Type() == "currency" {
				o.Currencies = append(o.Currencies, strings.TrimSpace(slice(src, c)))
			}
		}
		return o, true
	case "close":
		warnIgnoredKeyValues(file, src, n, diags, lines)
		return ast.Close{Meta: meta(file, src, n, lines), Account: textField(src, n, "account")}, true
	case "transaction":
		return convertTxn(file, src, n, diags, lines), true
	case "price":
		amt, _ := parseAmountNode(src, field(n, "amount"))
		return ast.Price{
			Meta:     meta(file, src, n, lines),
			Currency: textField(src, n, "currency"),
			Amount:   amt,
			Metadata: parseKeyValues(src, n),
		}, true
	case "balance":
		// amount may be field "amount" or amount_tolerance child
		an := field(n, "amount")
		if an == nil {
			an = field(n, "amount_tolerance")
		}
		if an == nil {
			for i := uint32(0); i < n.NamedChildCount(); i++ {
				c := n.NamedChild(i)
				if c.Type() == "amount" || c.Type() == "amount_tolerance" {
					an = c
					break
				}
			}
		}
		amt, _ := parseAmountNode(src, an)
		warnIgnoredKeyValues(file, src, n, diags, lines)
		return ast.Balance{Meta: meta(file, src, n, lines), Account: textField(src, n, "account"), Amount: amt}, true
	case "pad":
		warnIgnoredKeyValues(file, src, n, diags, lines)
		return ast.Pad{
			Meta:        meta(file, src, n, lines),
			Account:     textField(src, n, "account"),
			FromAccount: textField(src, n, "from_account"),
		}, true
	case "note":
		warnIgnoredKeyValues(file, src, n, diags, lines)
		return ast.Note{Meta: meta(file, src, n, lines), Account: textField(src, n, "account"), Comment: unquote(textField(src, n, "note"))}, true
	case "event":
		warnIgnoredKeyValues(file, src, n, diags, lines)
		return ast.Event{Meta: meta(file, src, n, lines), Type: unquote(textField(src, n, "type")), Desc: unquote(textField(src, n, "desc"))}, true
	case "document":
		// 2020-01-01 document Assets:Cash "docs/..."
		path := ""
		if fn := field(n, "filename"); fn != nil {
			path = unquote(strings.TrimSpace(slice(src, fn)))
			if path == "" {
				// filename node may wrap a string child
				for i := uint32(0); i < fn.NamedChildCount(); i++ {
					c := fn.NamedChild(i)
					if c.Type() == "string" {
						path = unquote(slice(src, c))
						break
					}
				}
			}
		}
		if path == "" {
			for i := uint32(0); i < n.NamedChildCount(); i++ {
				c := n.NamedChild(i)
				if c.Type() == "string" {
					path = unquote(slice(src, c))
					break
				}
			}
		}
		warnIgnoredKeyValues(file, src, n, diags, lines)
		return ast.Document{
			Meta:    meta(file, src, n, lines),
			Account: textField(src, n, "account"),
			Path:    path,
		}, true
	case "query", "custom":
		diags.Warn(file, lines.At(n.StartByte()), fmt.Sprintf("%s not supported; skipped", n.Type()))
		return nil, false
	case "comment":
		return nil, false
	case "pushtag", "poptag", "pushmeta", "popmeta":
		diags.Warn(file, lines.At(n.StartByte()), fmt.Sprintf("%s not supported; skipped", n.Type()))
		return nil, false
	default:
		diags.Warn(file, lines.At(n.StartByte()), fmt.Sprintf("unsupported directive %q skipped", n.Type()))
		return nil, false
	}
}

// parseKeyValues collects key_value children into a metadata map (string values).
func parseKeyValues(src []byte, n *grammar.Node) ast.Metadata {
	if n == nil {
		return nil
	}
	var md ast.Metadata
	for i := uint32(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if c.Type() != "key_value" {
			continue
		}
		key, val := keyValuePair(src, c)
		if key == "" {
			continue
		}
		if md == nil {
			md = ast.Metadata{}
		}
		md[key] = val
	}
	return md
}

func keyValuePair(src []byte, n *grammar.Node) (key, val string) {
	key = textField(src, n, "key")
	if key == "" {
		for j := uint32(0); j < n.NamedChildCount(); j++ {
			ch := n.NamedChild(j)
			if ch.Type() == "key" {
				key = strings.TrimSpace(slice(src, ch))
				break
			}
		}
	}
	// value is often a field, or a string / number child under "value"
	if vf := field(n, "value"); vf != nil {
		val = metadataValue(src, vf)
	}
	if val == "" {
		for j := uint32(0); j < n.NamedChildCount(); j++ {
			ch := n.NamedChild(j)
			if ch.Type() == "value" || ch.Type() == "string" || ch.Type() == "number" {
				val = metadataValue(src, ch)
				if val != "" {
					break
				}
			}
		}
	}
	return key, val
}

func metadataValue(src []byte, n *grammar.Node) string {
	if n == nil {
		return ""
	}
	switch n.Type() {
	case "string":
		return unquote(slice(src, n))
	case "number", "bool", "currency", "account", "tag", "link", "date":
		return strings.TrimSpace(slice(src, n))
	case "value":
		// unwrap single named child if present
		for i := uint32(0); i < n.NamedChildCount(); i++ {
			if s := metadataValue(src, n.NamedChild(i)); s != "" {
				return s
			}
		}
		return unquote(strings.TrimSpace(slice(src, n)))
	default:
		// try nested string
		for i := uint32(0); i < n.NamedChildCount(); i++ {
			if s := metadataValue(src, n.NamedChild(i)); s != "" {
				return s
			}
		}
		t := strings.TrimSpace(slice(src, n))
		return unquote(t)
	}
}

// warnIgnoredKeyValues emits a warn for each key_value child (metadata not stored yet).
func warnIgnoredKeyValues(file string, src []byte, n *grammar.Node, diags *diag.List, lines lineTable) {
	if n == nil || diags == nil {
		return
	}
	for i := uint32(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if c.Type() != "key_value" {
			continue
		}
		key, _ := keyValuePair(src, c)
		if key == "" {
			key = strings.TrimSpace(slice(src, c))
		}
		diags.Warn(file, lines.At(c.StartByte()), fmt.Sprintf("metadata %q ignored (not stored yet)", key))
	}
}

func convertTxn(file string, src []byte, n *grammar.Node, diags *diag.List, lines lineTable) ast.Transaction {
	txn := ast.Transaction{
		Meta:      meta(file, src, n, lines),
		Flag:      strings.TrimSpace(textField(src, n, "txn")),
		Narration: unquote(textField(src, n, "narration")),
		Payee:     unquote(textField(src, n, "payee")),
	}
	// tags/links
	if tl := field(n, "tags_links"); tl != nil {
		for i := uint32(0); i < tl.NamedChildCount(); i++ {
			c := tl.NamedChild(i)
			t := slice(src, c)
			switch c.Type() {
			case "tag":
				txn.Tags = append(txn.Tags, strings.TrimPrefix(t, "#"))
			case "link":
				txn.Links = append(txn.Links, strings.TrimPrefix(t, "^"))
			}
		}
	}
	warnIgnoredKeyValues(file, src, n, diags, lines)
	for i := uint32(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if c.Type() != "posting" {
			continue
		}
		txn.Postings = append(txn.Postings, convertPosting(file, src, c, diags, lines))
	}
	return txn
}

func convertPosting(file string, src []byte, n *grammar.Node, diags *diag.List, lines lineTable) ast.Posting {
	p := ast.Posting{Account: textField(src, n, "account")}
	warnIgnoredKeyValues(file, src, n, diags, lines)
	// amount field or incomplete_amount child
	an := field(n, "amount")
	if an == nil {
		for i := uint32(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(i)
			if c.Type() == "amount" || c.Type() == "incomplete_amount" {
				an = c
				break
			}
		}
	}
	if an != nil {
		if amt, ok := parseAmountNode(src, an); ok {
			p.Units = &amt
		}
	}
	if cs := field(n, "cost_spec"); cs != nil {
		p.Cost = parseCost(src, cs)
	}
	// price: child "atat" or "at" + field price_annotation
	hasAtAt := false
	hasAt := false
	for i := uint32(0); i < n.ChildCount(); i++ {
		c := n.Child(i)
		switch c.Type() {
		case "atat":
			hasAtAt = true
		case "at":
			hasAt = true
		}
	}
	if pa := field(n, "price_annotation"); pa != nil {
		if amt, ok := parseAmountNode(src, firstAmountish(pa)); ok {
			p.Price = &ast.PriceSpec{Number: amt.Number, Commodity: amt.Commodity, Total: hasAtAt || !hasAt && hasAtAt}
			if hasAtAt {
				p.Price.Total = true
			} else if hasAt {
				p.Price.Total = false
			} else {
				// default total if @@ node missing but only one form
				p.Price.Total = hasAtAt
			}
		}
	}
	// Fix Total detection: if we saw atat token
	if p.Price != nil {
		p.Price.Total = hasAtAt
	}
	return p
}

func parseCost(src []byte, n *grammar.Node) *ast.CostSpec {
	// cost_spec → cost_comp_list → cost_comp → compound_amount
	empty := true
	var num *big.Rat
	var cur string
	walkNamed(n, func(c *grammar.Node) {
		if c.Type() == "compound_amount" {
			empty = false
			if per := field(c, "per"); per != nil {
				num = parseNumber(src, per)
			}
			if cu := field(c, "currency"); cu != nil {
				cur = strings.TrimSpace(slice(src, cu))
			}
			// also total form
			if num == nil {
				if t := field(c, "total"); t != nil {
					num = parseNumber(src, t)
				}
			}
		}
		if c.Type() == "number" && num == nil {
			// bare
		}
	})
	// empty braces {}
	text := strings.TrimSpace(slice(src, n))
	if text == "{}" || text == "{ }" {
		return &ast.CostSpec{Empty: true}
	}
	if empty && (strings.Contains(text, "{}") || text == "{}") {
		return &ast.CostSpec{Empty: true}
	}
	if num == nil && cur == "" {
		// treat as empty cost spec if no numbers found
		if strings.Contains(text, "{") {
			return &ast.CostSpec{Empty: true}
		}
		return nil
	}
	return &ast.CostSpec{Number: num, Commodity: cur, Empty: num == nil}
}

func firstAmountish(n *grammar.Node) *grammar.Node {
	if n == nil {
		return nil
	}
	switch n.Type() {
	case "amount", "incomplete_amount", "amount_tolerance":
		return n
	}
	for i := uint32(0); i < n.NamedChildCount(); i++ {
		if c := firstAmountish(n.NamedChild(i)); c != nil {
			return c
		}
	}
	return n
}

func parseAmountNode(src []byte, n *grammar.Node) (ast.Amount, bool) {
	if n == nil {
		return ast.Amount{}, false
	}
	n = firstAmountish(n)
	var num *big.Rat
	var cur string
	walkNamed(n, func(c *grammar.Node) {
		switch c.Type() {
		case "number":
			if num == nil {
				num = rat(strings.TrimSpace(slice(src, c)))
			}
		case "unary_number_expr":
			num = parseUnary(src, c)
		case "currency":
			cur = strings.TrimSpace(slice(src, c))
		}
	})
	// also direct children order for incomplete_amount
	if num == nil {
		for i := uint32(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(i)
			if c.Type() == "unary_number_expr" {
				num = parseUnary(src, c)
			}
			if c.Type() == "number" && num == nil {
				num = rat(strings.TrimSpace(slice(src, c)))
			}
			if c.Type() == "currency" {
				cur = strings.TrimSpace(slice(src, c))
			}
		}
	}
	if num == nil {
		return ast.Amount{}, false
	}
	return ast.Amount{Number: num, Commodity: cur}, true
}

func parseUnary(src []byte, n *grammar.Node) *big.Rat {
	neg := false
	var num *big.Rat
	for i := uint32(0); i < n.ChildCount(); i++ {
		c := n.Child(i)
		switch c.Type() {
		case "minus", "-":
			neg = true
		case "number":
			num = rat(strings.TrimSpace(slice(src, c)))
		}
	}
	for i := uint32(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if c.Type() == "number" {
			num = rat(strings.TrimSpace(slice(src, c)))
		}
	}
	// fallback whole text
	if num == nil {
		t := strings.TrimSpace(slice(src, n))
		t = strings.ReplaceAll(t, " ", "")
		num = rat(t)
	}
	if num == nil {
		return big.NewRat(0, 1)
	}
	if neg && num.Sign() > 0 {
		num.Neg(num)
	}
	// if text already had minus, rat() handled it
	return num
}

func parseNumber(src []byte, n *grammar.Node) *big.Rat {
	if n == nil {
		return nil
	}
	if n.Type() == "unary_number_expr" {
		return parseUnary(src, n)
	}
	if n.Type() == "number" {
		return rat(strings.TrimSpace(slice(src, n)))
	}
	// search
	var r *big.Rat
	walkNamed(n, func(c *grammar.Node) {
		if r != nil {
			return
		}
		if c.Type() == "unary_number_expr" {
			r = parseUnary(src, c)
		}
		if c.Type() == "number" {
			r = rat(strings.TrimSpace(slice(src, c)))
		}
	})
	return r
}

func meta(file string, src []byte, n *grammar.Node, lines lineTable) ast.Meta {
	d := time.Time{}
	if dn := field(n, "date"); dn != nil {
		d, _ = time.ParseInLocation("2006-01-02", strings.TrimSpace(slice(src, dn)), time.UTC)
	}
	return ast.Meta{Date: d, File: file, Line: lines.At(n.StartByte())}
}

// lineTable maps byte offsets → 1-based line numbers.
// starts[i] is the byte offset where line i+1 begins.
type lineTable struct {
	starts []int
}

func newLineTable(src []byte) lineTable {
	// Pre-size ~1 entry per 40 bytes (typical ledger line length); grows if denser.
	starts := make([]int, 1, len(src)/40+2)
	starts[0] = 0
	for i, b := range src {
		if b == '\n' && i+1 < len(src) {
			starts = append(starts, i+1)
		}
	}
	return lineTable{starts: starts}
}

// At returns the 1-based line containing byte offset off.
func (lt lineTable) At(off uint32) int {
	if len(lt.starts) == 0 {
		return 1
	}
	o := int(off)
	// largest i with starts[i] <= o
	i := sort.Search(len(lt.starts), func(i int) bool {
		return lt.starts[i] > o
	}) - 1
	if i < 0 {
		return 1
	}
	return i + 1
}

func field(n *grammar.Node, name string) *grammar.Node {
	for i := uint32(0); i < n.ChildCount(); i++ {
		if n.FieldNameForChild(i) == name {
			c := n.Child(i)
			if !c.IsNull() {
				return c
			}
		}
	}
	return nil
}

func textField(src []byte, n *grammar.Node, name string) string {
	c := field(n, name)
	if c == nil {
		return ""
	}
	return strings.TrimSpace(slice(src, c))
}

func slice(src []byte, n *grammar.Node) string {
	if n == nil || n.IsNull() {
		return ""
	}
	s, e := int(n.StartByte()), int(n.EndByte())
	if s < 0 || e > len(src) || s > e {
		return ""
	}
	return string(src[s:e])
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func rat(s string) *big.Rat {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	r := new(big.Rat)
	if _, ok := r.SetString(s); !ok {
		return nil
	}
	return r
}

func walkNamed(n *grammar.Node, fn func(*grammar.Node)) {
	if n == nil || n.IsNull() {
		return
	}
	fn(n)
	for i := uint32(0); i < n.NamedChildCount(); i++ {
		walkNamed(n.NamedChild(i), fn)
	}
}

func clip(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
