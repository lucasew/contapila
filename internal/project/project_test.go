package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRoot(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "contapila-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	root := filepath.Join(tempDir, "my-project")
	sub := filepath.Join(root, "personal", "2024")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}

	markerPath := filepath.Join(root, MarkerFile)
	if err := os.WriteFile(markerPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	found, err := FindRoot(sub)
	if err != nil {
		t.Errorf("FindRoot failed: %v", err)
	}
	if found != root {
		t.Errorf("got root %s, want %s", found, root)
	}

	_, err = FindRoot(tempDir)
	if err == nil {
		t.Error("FindRoot should have failed outside project")
	}
}

func TestDiscoverLedgers(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "contapila-ledgers-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// personal ledger
	if err := os.MkdirAll(filepath.Join(tempDir, "personal"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "personal", "main.beancount"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// empresa ledger
	if err := os.MkdirAll(filepath.Join(tempDir, "empresa"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "empresa", "main.beancount"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// ignore-me (no main.beancount)
	if err := os.MkdirAll(filepath.Join(tempDir, "ignore-me"), 0755); err != nil {
		t.Fatal(err)
	}

	ledgers, err := DiscoverLedgers(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(ledgers) != 2 {
		t.Errorf("got %d ledgers, want 2", len(ledgers))
	}

	foundPersonal := false
	foundEmpresa := false
	for _, l := range ledgers {
		if l.Name == "personal" {
			foundPersonal = true
		}
		if l.Name == "empresa" {
			foundEmpresa = true
		}
	}

	if !foundPersonal || !foundEmpresa {
		t.Errorf("did not find both personal and empresa ledgers: %+v", ledgers)
	}
}
