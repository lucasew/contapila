package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/lucasew/contapila-go/internal/ast"
	"github.com/lucasew/contapila-go/internal/diag"
	"github.com/lucasew/contapila-go/internal/parser"
)

// LoadFile parses a file and expands includes depth-first.
func LoadFile(path string) ([]ast.Directive, diag.List, error) {
	var diags diag.List
	seen := map[string]bool{}
	stack := map[string]bool{}
	var out []ast.Directive
	err := loadOne(path, &out, &diags, seen, stack)
	return out, diags, err
}

func loadOne(path string, out *[]ast.Directive, diags *diag.List, seen, stack map[string]bool) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// file may not exist yet for eval; use abs
		real = abs
	}
	if stack[real] {
		diags.Error(path, 0, "include cycle detected")
		return fmt.Errorf("include cycle at %s", path)
	}
	if seen[real] {
		return nil // dedupe
	}
	stack[real] = true
	defer delete(stack, real)

	src, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	seen[real] = true

	dirs, pdiags, err := parser.Parse(abs, src)
	diags.Merge(pdiags)
	if err != nil {
		return err
	}

	dir := filepath.Dir(abs)
	for _, d := range dirs {
		inc, ok := d.(ast.Include)
		if !ok {
			*out = append(*out, d)
			continue
		}
		if err := expandInclude(dir, inc.Path, out, diags, seen, stack); err != nil {
			return err
		}
	}
	return nil
}

func expandInclude(baseDir, pattern string, out *[]ast.Directive, diags *diag.List, seen, stack map[string]bool) error {
	// absolute or relative to baseDir
	target := pattern
	if !filepath.IsAbs(pattern) {
		target = filepath.Join(baseDir, pattern)
	}

	// literal file?
	if !hasGlob(pattern) {
		if _, err := os.Stat(target); err != nil {
			if os.IsNotExist(err) {
				diags.Error(baseDir, 0, fmt.Sprintf("include missing: %s", pattern))
				return fmt.Errorf("include missing: %s", pattern)
			}
			return err
		}
		return loadOne(target, out, diags, seen, stack)
	}

	matches, err := filepath.Glob(target)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		// also try walking with path.Match style via Glob only
		diags.Warn(baseDir, 0, fmt.Sprintf("include glob matched zero files: %s", pattern))
		return nil
	}
	sort.Strings(matches)
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil {
			return err
		}
		if info.IsDir() {
			continue
		}
		if err := loadOne(m, out, diags, seen, stack); err != nil {
			return err
		}
	}
	return nil
}

func hasGlob(s string) bool {
	for _, c := range s {
		if c == '*' || c == '?' || c == '[' {
			return true
		}
	}
	return false
}
