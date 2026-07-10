// Package docs expands per-ledger <ledger>/docs/by-account trees into document directives.
package docs

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/lucasew/contapila-go/internal/ast"
)

// ByAccountDir is the account document folder under each ledger's docs/.
const ByAccountDir = "by-account"

// yyyymmdd at start of filename (SPEC §4.4).
var datePrefix = regexp.MustCompile(`^(\d{8})`)

// LedgerDocsRel returns <ledger>/docs relative to the project root.
func LedgerDocsRel(ledger string) string {
	return filepath.ToSlash(filepath.Join(ledger, "docs"))
}

// ScanByAccount walks <root>/<ledger>/docs/by-account and synthesizes document directives.
// Account path is directory segments joined with ':' under by-account;
// file names must start with yyyymmdd.
func ScanByAccount(projectRoot, ledger string) ([]ast.Document, error) {
	if ledger == "" {
		return nil, nil
	}
	root := filepath.Join(projectRoot, ledger, "docs", ByAccountDir)
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}

	var out []ast.Document
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		base := d.Name()
		m := datePrefix.FindStringSubmatch(base)
		if m == nil {
			return nil
		}
		dt, err := time.ParseInLocation("20060102", m[1], time.UTC)
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		dir := filepath.ToSlash(filepath.Dir(rel))
		if dir == "." {
			return nil
		}
		account := strings.ReplaceAll(dir, "/", ":")
		projRel := filepath.ToSlash(filepath.Join(ledger, "docs", ByAccountDir, rel))
		out = append(out, ast.Document{
			Meta:      ast.Meta{Date: dt, File: "docs.gen"},
			Account:   account,
			Path:      projRel,
			Synthetic: true,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.Before(out[j].Date)
		}
		if out[i].Account != out[j].Account {
			return out[i].Account < out[j].Account
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

// Merge combines ledger document directives with synthetic docs.
// Prefer explicit (non-synthetic) when the same Path appears twice.
func Merge(fromLedger, synthetic []ast.Document) []ast.Document {
	byPath := map[string]ast.Document{}
	order := make([]string, 0)
	add := func(d ast.Document) {
		if d.Path == "" {
			return
		}
		key := filepath.ToSlash(d.Path)
		if prev, ok := byPath[key]; ok {
			if prev.Synthetic && !d.Synthetic {
				byPath[key] = d
			}
			return
		}
		byPath[key] = d
		order = append(order, key)
	}
	for _, d := range fromLedger {
		add(d)
	}
	for _, d := range synthetic {
		add(d)
	}
	out := make([]ast.Document, 0, len(order))
	for _, k := range order {
		out = append(out, byPath[k])
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.Before(out[j].Date)
		}
		if out[i].Account != out[j].Account {
			return out[i].Account < out[j].Account
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// ForAccount filters documents linked to account (exact match).
func ForAccount(docs []ast.Document, account string) []ast.Document {
	var out []ast.Document
	for _, d := range docs {
		if d.Account == account {
			out = append(out, d)
		}
	}
	return out
}

// IsLedgerDocPath reports whether rel is under <ledger>/docs/ (safe to serve).
func IsLedgerDocPath(rel string) bool {
	rel = filepath.ToSlash(rel)
	rel = strings.TrimPrefix(rel, "/")
	parts := strings.Split(rel, "/")
	// <ledger>/docs/...
	return len(parts) >= 3 && parts[1] == "docs"
}
