package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

func TestPDFDump(t *testing.T) {
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "test.pdf")

	// Generate a 1-page PDF using gofpdf
	pdfWriter := gofpdf.New("P", "mm", "A4", "")
	pdfWriter.AddPage()
	pdfWriter.SetFont("Arial", "B", 16)
	pdfWriter.Cell(40, 10, "Hello World from PDF!")
	if err := pdfWriter.OutputFileAndClose(pdfPath); err != nil {
		t.Fatalf("failed to write pdf: %v", err)
	}

	// Run pdf-dump
	cmd := pdfDumpCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{pdfPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to execute pdfDumpCmd: %v", err)
	}

	var dump PDFDump
	if err := json.Unmarshal(buf.Bytes(), &dump); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v, output: %s", err, buf.String())
	}

	if len(dump.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(dump.Pages))
	}

	page := dump.Pages[0]
	if page.PageNumber != 1 {
		t.Fatalf("expected page number 1, got %d", page.PageNumber)
	}

	// Verify that text is present
	var sb strings.Builder
	for _, text := range page.Text {
		sb.WriteString(text.S)
	}
	combinedText := sb.String()
	if !strings.Contains(combinedText, "Hello") || !strings.Contains(combinedText, "World") {
		t.Fatalf("expected text to contain 'Hello' and 'World', got %q", combinedText)
	}
}

func TestXLSXDump(t *testing.T) {
	tmpDir := t.TempDir()
	xlsxPath := filepath.Join(tmpDir, "test.xlsx")

	// Create a new Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Set cell values
	sheetName := "Sheet1"
	f.SetCellValue(sheetName, "A1", "Hello")
	f.SetCellValue(sheetName, "B1", "World")
	f.SetCellValue(sheetName, "A2", 123)

	if err := f.SaveAs(xlsxPath); err != nil {
		t.Fatalf("failed to save excel: %v", err)
	}

	// Run xlsx-dump
	cmd := xlsxDumpCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{xlsxPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to execute xlsxDumpCmd: %v", err)
	}

	var dump XLSXDump
	if err := json.Unmarshal(buf.Bytes(), &dump); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v, output: %s", err, buf.String())
	}

	if len(dump.Sheets) == 0 {
		t.Fatalf("expected sheets, got 0")
	}

	var found bool
	for _, s := range dump.Sheets {
		if s.Name == sheetName {
			found = true
			if len(s.Rows) < 2 {
				t.Fatalf("expected at least 2 rows, got %d", len(s.Rows))
			}
			if len(s.Rows[0]) < 2 || s.Rows[0][0] != "Hello" || s.Rows[0][1] != "World" {
				t.Fatalf("unexpected content in row 0: %v", s.Rows[0])
			}
			if len(s.Rows[1]) < 1 || s.Rows[1][0] != "123" {
				t.Fatalf("unexpected content in row 1: %v", s.Rows[1])
			}
		}
	}
	if !found {
		t.Fatalf("sheet %q not found in dump", sheetName)
	}
}
