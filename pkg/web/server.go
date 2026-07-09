package web

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/lucasew/contapila/pkg/ledger"
)

var layoutTmpl = `
<!DOCTYPE html>
<html>
<head>
    <title>{{block "title" .}}Contapila{{end}}</title>
    <style>
        body { font-family: sans-serif; line-height: 1.5; max-width: 800px; margin: 2rem auto; padding: 0 1rem; }
        nav { margin-bottom: 2rem; border-bottom: 1px solid #ccc; padding-bottom: 0.5rem; }
        nav a { margin-right: 1rem; text-decoration: none; color: #007bff; }
        nav a:hover { text-decoration: underline; }
        table { width: 100%; border-collapse: collapse; margin-top: 1rem; }
        th, td { text-align: left; padding: 0.5rem; border-bottom: 1px solid #eee; }
        .error { color: #dc3545; }
        .warn { color: #ffc107; }
        .amount { text-align: right; font-family: monospace; }
    </style>
</head>
<body>
    <nav>
        <a href="/">Home</a>
        {{if .LedgerName}}
        <a href="/ledger/{{.LedgerName}}/check">Check</a>
        <a href="/ledger/{{.LedgerName}}/balances">Balances</a>
        {{end}}
    </nav>
    <h1>{{block "header" .}}Contapila{{end}}</h1>
    {{block "content" .}}{{end}}
</body>
</html>
`

var indexTmpl = `
{{define "title"}}Contapila{{end}}
{{define "header"}}Ledgers{{end}}
{{define "content"}}
<ul>
    {{range .LedgerNames}}
    <li><a href="/ledger/{{.}}/check">{{.}}</a></li>
    {{end}}
</ul>
{{end}}
`

var checkTmpl = `
{{define "title"}}Check - {{.LedgerName}}{{end}}
{{define "header"}}Check: {{.LedgerName}}{{end}}
{{define "content"}}
{{if .Diagnostics}}
    <table>
        <thead>
            <tr>
                <th>File:Line</th>
                <th>Severity</th>
                <th>Message</th>
            </tr>
        </thead>
        <tbody>
            {{range .Diagnostics}}
            <tr class="{{.Severity}}">
                <td>{{.File}}:{{.Line}}</td>
                <td>{{.Severity}}</td>
                <td>{{.Message}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
{{else}}
    <p>Clean</p>
{{end}}
{{end}}
`

var balancesTmpl = `
{{define "title"}}Balances - {{.LedgerName}}{{end}}
{{define "header"}}Balances: {{.LedgerName}} (as of {{.AsOf}}){{end}}
{{define "content"}}
<form method="GET">
    <label for="as-of">As of:</label>
    <input type="date" id="as-of" name="as-of" value="{{.AsOf}}">
    <button type="submit">Update</button>
</form>
<table>
    <thead>
        <tr>
            <th>Account</th>
            <th class="amount">Amount</th>
            <th>Commodity</th>
        </tr>
    </thead>
    <tbody>
        {{range .Balances}}
        <tr>
            <td>{{.Account}}</td>
            <td class="amount">{{.Amount.FloatString 2}}</td>
            <td>{{.Commodity}}</td>
        </tr>
        {{end}}
    </tbody>
</table>
{{end}}
`

func StartServer(p *ledger.Project, defaultLedger string, port int) error {
	layout := template.Must(template.New("layout").Parse(layoutTmpl))
	index := template.Must(template.Must(layout.Clone()).Parse(indexTmpl))
	check := template.Must(template.Must(layout.Clone()).Parse(checkTmpl))
	balances := template.Must(template.Must(layout.Clone()).Parse(balancesTmpl))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := struct {
			LedgerNames []string
			LedgerName  string
		}{
			LedgerNames: p.LedgerNames,
		}
		index.Execute(w, data)
	})

	http.HandleFunc("/ledger/", func(w http.ResponseWriter, r *http.Request) {
		// Basic routing: /ledger/<name>/<action>
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 3 {
			http.NotFound(w, r)
			return
		}
		name := parts[1]
		action := parts[2]

		l, err := p.LoadLedger(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		switch action {
		case "check":
			diagnostics, _ := l.Check()
			data := struct {
				LedgerName  string
				Diagnostics []ledger.Diagnostic
			}{
				LedgerName:  name,
				Diagnostics: diagnostics,
			}
			check.Execute(w, data)
		case "balances":
			asOfStr := r.URL.Query().Get("as-of")
			asOf := time.Now()
			if asOfStr != "" {
				if t, err := time.Parse("2006-01-02", asOfStr); err == nil {
					asOf = t
				}
			}
			bal, _ := l.GetBalances(asOf)
			data := struct {
				LedgerName string
				Balances   []ledger.Balance
				AsOf       string
			}{
				LedgerName: name,
				Balances:   bal,
				AsOf:       asOf.Format("2006-01-02"),
			}
			balances.Execute(w, data)
		default:
			http.NotFound(w, r)
		}
	})

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	fmt.Printf("Starting read-only server on http://%s\n", addr)
	return http.ListenAndServe(addr, nil)
}
