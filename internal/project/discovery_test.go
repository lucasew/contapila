package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRoot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "contapila-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	root := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(filepath.Join(root, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	cuePath := filepath.Join(root, "contapila.cue")
	if err := os.WriteFile(cuePath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		start string
		want  string
	}{
		{root, root},
		{filepath.Join(root, "subdir"), root},
	}

	for _, tt := range tests {
		got, err := FindRoot(tt.start)
		if err != nil {
			t.Errorf("FindRoot(%q) error: %v", tt.start, err)
			continue
		}
		if got != tt.want {
			t.Errorf("FindRoot(%q) = %q, want %q", tt.start, got, tt.want)
		}
	}
}

func TestLoadProject(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "contapila-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	root := tmpDir
	os.Mkdir(filepath.Join(root, "personal"), 0755)
	os.WriteFile(filepath.Join(root, "personal", "main.beancount"), []byte(""), 0644)
	os.Mkdir(filepath.Join(root, "empresa"), 0755)
	os.WriteFile(filepath.Join(root, "empresa", "main.beancount"), []byte(""), 0644)
	os.Mkdir(filepath.Join(root, "ignored"), 0755)

	p, err := LoadProject(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Ledgers) != 2 {
		t.Errorf("expected 2 ledgers, got %d", len(p.Ledgers))
	}
	if _, ok := p.Ledgers["personal"]; !ok {
		t.Error("missing 'personal' ledger")
	}
	if _, ok := p.Ledgers["empresa"]; !ok {
		t.Error("missing 'empresa' ledger")
	}
}
