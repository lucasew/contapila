package main

import (
	"encoding/json"
	"fmt"

	"github.com/dslipak/pdf"
	"github.com/spf13/cobra"
	"github.com/xuri/excelize/v2"
)

type PDFDump struct {
	Pages []PDFPage `json:"pages"`
}

type PDFPage struct {
	PageNumber int         `json:"page_number"`
	Text       []PDFText   `json:"text"`
	Rects      []PDFRect   `json:"rects"`
	Rows       []PDFRow    `json:"rows"`
	Columns    []PDFColumn `json:"columns"`
}

type PDFText struct {
	Font     string  `json:"font"`
	FontSize float64 `json:"font_size"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	W        float64 `json:"w"`
	S        string  `json:"s"`
}

type PDFPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type PDFRect struct {
	Min PDFPoint `json:"min"`
	Max PDFPoint `json:"max"`
}

type PDFRow struct {
	Position int64     `json:"position"`
	Content  []PDFText `json:"content"`
}

type PDFColumn struct {
	Position int64     `json:"position"`
	Content  []PDFText `json:"content"`
}

func mapPDFText(t pdf.Text) PDFText {
	return PDFText{
		Font:     t.Font,
		FontSize: t.FontSize,
		X:        t.X,
		Y:        t.Y,
		W:        t.W,
		S:        t.S,
	}
}

func mapPDFTexts(ts []pdf.Text) []PDFText {
	var out []PDFText
	for _, t := range ts {
		out = append(out, mapPDFText(t))
	}
	return out
}

func pdfDumpCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "pdf-dump <file>",
		Aliases: []string{"pdf"},
		Short:   "Dump a PDF file's layout/text hierarchy as standard JSON",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			r, err := pdf.Open(filePath)
			if err != nil {
				return fmt.Errorf("open pdf: %w", err)
			}

			var dump PDFDump
			numPages := r.NumPage()
			for i := 1; i <= numPages; i++ {
				p := r.Page(i)
				content := p.Content()

				var rects []PDFRect
				for _, rc := range content.Rect {
					rects = append(rects, PDFRect{
						Min: PDFPoint{X: rc.Min.X, Y: rc.Min.Y},
						Max: PDFPoint{X: rc.Max.X, Y: rc.Max.Y},
					})
				}

				var pdfRows []PDFRow
				rows, _ := p.GetTextByRow()
				for _, r := range rows {
					pdfRows = append(pdfRows, PDFRow{
						Position: r.Position,
						Content:  mapPDFTexts(r.Content),
					})
				}

				var pdfCols []PDFColumn
				cols, _ := p.GetTextByColumn()
				for _, c := range cols {
					pdfCols = append(pdfCols, PDFColumn{
						Position: c.Position,
						Content:  mapPDFTexts(c.Content),
					})
				}

				dump.Pages = append(dump.Pages, PDFPage{
					PageNumber: i,
					Text:       mapPDFTexts(content.Text),
					Rects:      rects,
					Rows:       pdfRows,
					Columns:    pdfCols,
				})
			}

			out, err := json.MarshalIndent(dump, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal json: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return nil
		},
	}
}

type XLSXDump struct {
	Sheets []XLSXSheet `json:"sheets"`
}

type XLSXSheet struct {
	Name string     `json:"name"`
	Rows [][]string `json:"rows"`
}

func xlsxDumpCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "xlsx-dump <file>",
		Aliases: []string{"xlsx"},
		Short:   "Dump an Excel (.xlsx) file's content hierarchy as standard JSON",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			f, err := excelize.OpenFile(filePath)
			if err != nil {
				return fmt.Errorf("open excel: %w", err)
			}
			defer f.Close()

			var dump XLSXDump
			sheets := f.GetSheetList()
			for _, sheet := range sheets {
				rows, err := f.GetRows(sheet)
				if err != nil {
					return fmt.Errorf("get rows for sheet %q: %w", sheet, err)
				}
				dump.Sheets = append(dump.Sheets, XLSXSheet{
					Name: sheet,
					Rows: rows,
				})
			}

			out, err := json.MarshalIndent(dump, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal json: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return nil
		},
	}
}
