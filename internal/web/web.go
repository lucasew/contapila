package web

import (
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"time"

	"github.com/lucasew/contapila-go/internal/engine"
	"github.com/lucasew/contapila-go/internal/prices"
	"github.com/lucasew/contapila-go/pkg/project"
)

func Listen(p *project.Project, pdb *prices.DB, defaultLedger string, port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		names := engine.LedgerNames(p)
		_ = indexTmpl.Execute(w, names)
	})
	mux.HandleFunc("/ledger/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[len("/ledger/"):]
		if name == "" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		// strip subpath
		page := "check"
		if i := len(name); i > 0 {
			for j, c := range name {
				if c == '/' {
					page = name[j+1:]
					name = name[:j]
					break
				}
			}
		}
		l, err := engine.OpenLedger(p, pdb, name)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		switch page {
		case "", "check":
			_ = checkTmpl.Execute(w, l)
		case "balances":
			asOf := r.URL.Query().Get("as-of")
			t, _ := engine.ParseDate(asOf)
			if t.IsZero() {
				t = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
			}
			bals := l.BalancesAsOf(t)
			type row struct{ Account, Commodity, Amount string }
			var rows []row
			var accts []string
			for a := range bals {
				accts = append(accts, a)
			}
			sort.Strings(accts)
			for _, a := range accts {
				var cs []string
				for c := range bals[a] {
					cs = append(cs, c)
				}
				sort.Strings(cs)
				for _, c := range cs {
					rows = append(rows, row{a, c, bals[a][c].FloatString(4)})
				}
			}
			_ = balancesTmpl.Execute(w, map[string]any{"Ledger": l, "Rows": rows})
		case "journal":
			_ = journalTmpl.Execute(w, map[string]any{"Ledger": l, "Entries": l.Journal(time.Time{}, time.Time{})})
		case "pnl":
			_ = pnlTmpl.Execute(w, map[string]any{"Ledger": l, "PnL": l.PnL(time.Time{}, time.Time{})})
		case "networth":
			lines, total, err := l.NetWorth(time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC))
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			_ = nwTmpl.Execute(w, map[string]any{"Ledger": l, "Lines": lines, "Total": total.FloatString(2)})
		default:
			http.NotFound(w, r)
		}
	})

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	fmt.Printf("contapila web on http://%s/\n", addr)
	if defaultLedger != "" {
		fmt.Printf("  ledger: http://%s/ledger/%s/check\n", addr, defaultLedger)
	}
	return http.ListenAndServe(addr, mux)
}

var indexTmpl = template.Must(template.New("i").Parse(`<!doctype html><title>contapila</title>
<h1>Ledgers</h1><ul>{{range .}}<li><a href="/ledger/{{.}}/check">{{.}}</a>
 — <a href="/ledger/{{.}}/balances">balances</a>
 — <a href="/ledger/{{.}}/journal">journal</a>
 — <a href="/ledger/{{.}}/pnl">pnl</a>
 — <a href="/ledger/{{.}}/networth">networth</a></li>{{end}}</ul>`))

var checkTmpl = template.Must(template.New("c").Parse(`<!doctype html><title>{{.Name}} check</title>
<p><a href="/">home</a></p><h1>{{.Name}} check</h1>
{{if .Diags}}<ul>{{range .Diags}}<li>{{.}}</li>{{end}}</ul>{{else}}<p>OK</p>{{end}}`))

var balancesTmpl = template.Must(template.New("b").Parse(`<!doctype html><title>{{.Ledger.Name}} balances</title>
<p><a href="/">home</a></p><h1>{{.Ledger.Name}} balances</h1>
<table border=1 cellpadding=4><tr><th>Account</th><th>Amount</th><th>Commodity</th></tr>
{{range .Rows}}<tr><td>{{.Account}}</td><td align=right>{{.Amount}}</td><td>{{.Commodity}}</td></tr>{{end}}
</table>`))

var journalTmpl = template.Must(template.New("j").Parse(`<!doctype html><title>{{.Ledger.Name}} journal</title>
<p><a href="/">home</a></p><h1>{{.Ledger.Name}} journal</h1>
{{range .Entries}}<div><strong>{{.Date.Format "2006-01-02"}}</strong> {{.Kind}} {{.Narration}} {{.Comment}}
{{if .Postings}}<ul>{{range .Postings}}<li>{{.Account}} {{if .Units}}{{.Units.Number.FloatString 4}} {{.Units.Commodity}}{{end}}</li>{{end}}</ul>{{end}}
</div>{{end}}`))

var pnlTmpl = template.Must(template.New("p").Parse(`<!doctype html><title>{{.Ledger.Name}} pnl</title>
<p><a href="/">home</a></p><h1>{{.Ledger.Name}} P&amp;L</h1>
<h2>Income</h2><ul>{{range $k,$v := .PnL.Income}}<li>{{$k}}: {{$v.FloatString 4}}</li>{{end}}</ul>
<h2>Expenses</h2><ul>{{range $k,$v := .PnL.Expenses}}<li>{{$k}}: {{$v.FloatString 4}}</li>{{end}}</ul>`))

var nwTmpl = template.Must(template.New("n").Parse(`<!doctype html><title>{{.Ledger.Name}} networth</title>
<p><a href="/">home</a></p><h1>{{.Ledger.Name}} net worth ({{.Ledger.OpCurrency}})</h1>
<ul>{{range .Lines}}<li>{{.Account}} {{.Units.FloatString 4}} {{.Commodity}} => {{.Value.FloatString 2}}</li>{{end}}</ul>
<p><strong>TOTAL {{.Total}} {{.Ledger.OpCurrency}}</strong></p>`))
