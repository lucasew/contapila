package parser

import (
	"math/big"
	"strings"

	"github.com/modernc-tree-sitter/ccgo-tree-sitter/grammar"
)

// evalNumberExpr evaluates a Beancount number / unary / binary expression tree.
// Supports +, -, *, / and unary minus as exposed by the tree-sitter grammar.
func evalNumberExpr(src []byte, n *grammar.Node) *big.Rat {
	if n == nil || n.IsNull() {
		return nil
	}
	switch n.Type() {
	case "number":
		return rat(strings.TrimSpace(slice(src, n)))
	case "unary_number_expr":
		return evalUnaryExpr(src, n)
	case "binary_number_expr":
		return evalBinaryExpr(src, n)
	}
	// Unwrap single named child expr, or find first number-ish descendant.
	var found *big.Rat
	for i := uint32(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		switch c.Type() {
		case "number", "unary_number_expr", "binary_number_expr":
			if r := evalNumberExpr(src, c); r != nil {
				return r
			}
		default:
			if found == nil {
				found = evalNumberExpr(src, c)
			}
		}
	}
	if found != nil {
		return found
	}
	// Last resort: parse whole text (handles simple "-1.5").
	t := strings.TrimSpace(slice(src, n))
	t = strings.ReplaceAll(t, " ", "")
	return rat(t)
}

func evalUnaryExpr(src []byte, n *grammar.Node) *big.Rat {
	neg := false
	var inner *big.Rat
	// Prefer full child walk (ops may be unnamed in some builds).
	for i := uint32(0); i < n.ChildCount(); i++ {
		c := n.Child(i)
		switch c.Type() {
		case "minus", "-":
			neg = true
		case "number", "unary_number_expr", "binary_number_expr":
			inner = evalNumberExpr(src, c)
		}
	}
	for i := uint32(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		switch c.Type() {
		case "minus":
			neg = true
		case "number", "unary_number_expr", "binary_number_expr":
			if inner == nil {
				inner = evalNumberExpr(src, c)
			}
		}
	}
	if inner == nil {
		t := strings.TrimSpace(slice(src, n))
		t = strings.ReplaceAll(t, " ", "")
		inner = rat(t)
	}
	if inner == nil {
		return nil
	}
	out := new(big.Rat).Set(inner)
	if neg {
		// If rat already parsed a leading minus from text, don't double-negate.
		// Only negate when we saw a distinct minus token and value is positive,
		// or always negate when token present and text doesn't start with '-'.
		out.Neg(out)
	}
	return out
}

func evalBinaryExpr(src []byte, n *grammar.Node) *big.Rat {
	var left, right *big.Rat
	op := ""
	// Named children order: left, op, right (ops are named: plus/minus/asterisk/slash).
	for i := uint32(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		switch c.Type() {
		case "number", "unary_number_expr", "binary_number_expr":
			if left == nil {
				left = evalNumberExpr(src, c)
			} else {
				right = evalNumberExpr(src, c)
			}
		case "plus":
			op = "+"
		case "minus":
			op = "-"
		case "asterisk":
			op = "*"
		case "slash":
			op = "/"
		}
	}
	// Unnamed operator tokens fallback.
	if op == "" {
		for i := uint32(0); i < n.ChildCount(); i++ {
			c := n.Child(i)
			t := c.Type()
			if t == "+" || t == "-" || t == "*" || t == "/" {
				op = t
			}
			if t == "plus" {
				op = "+"
			}
			if t == "minus" && op == "" {
				op = "-"
			}
			if t == "asterisk" {
				op = "*"
			}
			if t == "slash" {
				op = "/"
			}
		}
	}
	if left == nil || right == nil || op == "" {
		// Fallback: whole-node text (may fail for complex trees).
		return rat(strings.ReplaceAll(strings.TrimSpace(slice(src, n)), " ", ""))
	}
	out := new(big.Rat)
	switch op {
	case "+":
		return out.Add(left, right)
	case "-":
		return out.Sub(left, right)
	case "*":
		return out.Mul(left, right)
	case "/":
		if right.Sign() == 0 {
			return nil
		}
		return out.Quo(left, right)
	default:
		return nil
	}
}
