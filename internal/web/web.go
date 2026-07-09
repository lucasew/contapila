package web

import (
	"embed"
	"fmt"
	"html/template"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/lucasew/contapila-go/internal/diag"
	"github.com/lucasew/contapila-go/internal/engine"
	"github.com/lucasew/contapila-go/internal/period"
	"github.com/lucasew/contapila-go/internal/prices"
	"github.com/lucasew/contapila-go/pkg/project"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Server struct {
	Project *project.Project
	Prices  *prices.DB
	Tmpl    *template.Template
}

func Listen(p *project.Project, pdb *prices.DB, defaultLedger string, addr string) error {
	s, err := New(p, pdb)
	if err != nil {
		return err
	}
	fmt.Printf("contapila web on http://%s/\n", addr)
	if defaultLedger != "" {
		fmt.Printf("  ledger: http://%s/l/%s/check\n", addr, defaultLedger)
	}
	return http.ListenAndServe(addr, s.Handler())
}

func New(p *project.Project, pdb *prices.DB) (*Server, error) {
	funcMap := template.FuncMap{
		"eq": func(a, b string) bool { return a == b },
		"queryEscape": url.QueryEscape,
		"ledgerURL": func(ledger, page, timeFilter string) string {
			u := "/l/" + url.PathEscape(ledger) + "/" + url.PathEscape(page)
			if timeFilter != "" {
				u += "?time=" + url.QueryEscape(timeFilter)
			}
			return u
		},
		"accountURL": func(ledger, account, timeFilter string) string {
			// Keep ":" readable in URLs; escape only reserved bits.
			u := "/l/" + url.PathEscape(ledger) + "/account/" + url.PathEscape(account)
			if timeFilter != "" {
				u += "?time=" + url.QueryEscape(timeFilter)
			}
			return u
		},
	}
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{Project: p, Prices: pdb, Tmpl: tmpl}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	// Built CSS (daisyUI themes via @plugin) — correct text/css MIME.
	mux.Handle("GET /static/", http.FileServer(http.FS(staticFS)))
	mux.HandleFunc("GET /{$}", s.handleIndex)
	// Account before generic page so "account" is not a page name.
	mux.HandleFunc("GET /l/{ledger}/account/{account...}", s.handleAccount)
	mux.HandleFunc("GET /l/{ledger}/{page}", s.handleLedgerPage)
	mux.HandleFunc("GET /l/{ledger}/{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/l/"+r.PathValue("ledger")+"/check", http.StatusFound)
	})
	return mux
}

type pageData struct {
	Title        string
	Page         string
	LedgerName   string
	Ledgers      []string
	ProjectRoot  string
	OpCurrency   string
	Diags        diag.List
	HasErrors    bool
	HasWarnings  bool
	OK           bool
	BalanceRows  []balanceRow
	Journal      []engine.JournalEntry
	IncomeRows   []kvRow
	ExpenseRows  []kvRow
	NetWorthRows []nwRow
	NetWorthTot  string
	AsOf         string
	Time        string
	PeriodLabel string
	Error       string
	// Account page
	AccountName     string
	AccountBalances []balanceRow
	AccountActivity []balanceRow
}

type balanceRow struct {
	Account   string
	Commodity string
	Amount    string
}

type kvRow struct {
	Key   string
	Value string
}

type nwRow struct {
	Account   string
	Commodity string
	Units     string
	Value     string
	UsedCost  bool
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := pageData{
		Title:       "Ledgers",
		Page:        "home",
		Ledgers:     engine.LedgerNames(s.Project),
		ProjectRoot: s.Project.Root,
	}
	s.render(w, "index.html", data)
}

func (s *Server) handleLedgerPage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("ledger")
	page := r.PathValue("page")
	l, err := engine.OpenLedger(s.Project, s.Prices, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	now := time.Now()
	timeStr := q.Get("time")
	if timeStr == "" {
		fromStr := q.Get("from")
		toStr := q.Get("to")
		if fromStr != "" || toStr != "" {
			if fromStr != "" && toStr != "" {
				timeStr = fromStr + " - " + toStr
			} else if fromStr != "" {
				timeStr = fromStr
			} else {
				timeStr = toStr
			}
		}
	}

	pr, perr := period.Parse(timeStr, now)
	periodLabel := period.DisplayLabel(timeStr, now)

	// as-of: explicit flag wins; else end of time filter; else far future
	asOfStr := q.Get("as-of")
	var asOf time.Time
	if asOfStr != "" {
		asOf, _ = engine.ParseDate(asOfStr)
	}
	if asOf.IsZero() && !pr.End.IsZero() {
		asOf = pr.End
		asOfStr = asOf.Format("2006-01-02")
	}
	if asOf.IsZero() {
		asOf = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
		asOfStr = ""
	}

	data := pageData{
		Title:       name + " · " + page,
		Page:        page,
		LedgerName:  name,
		Ledgers:     engine.LedgerNames(s.Project),
		ProjectRoot: s.Project.Root,
		OpCurrency:  l.OpCurrency,
		Diags:       l.Diags,
		HasErrors:   l.Diags.HasErrors(),
		HasWarnings: hasWarn(l.Diags),
		OK:          !l.Diags.HasErrors(),
		AsOf:        asOfStr,
		Time:        timeStr,
		PeriodLabel: periodLabel,
	}
	if perr != nil {
		data.Error = perr.Error()
	}

	switch page {
	case "check":
		// ok
	case "balances":
		data.BalanceRows = buildBalances(l.BalancesAsOf(asOf))
	case "journal":
		if perr == nil {
			data.Journal = l.Journal(pr.Start, pr.End)
		}
	case "pnl":
		if perr == nil {
			p := l.PnL(pr.Start, pr.End)
			data.IncomeRows = ratMapRows(p.Income)
			data.ExpenseRows = ratMapRows(p.Expenses)
		}
	case "networth":
		lines, total, err := l.NetWorth(asOf)
		if err != nil {
			data.Error = err.Error()
		} else {
			data.NetWorthTot = total.FloatString(2)
			for _, ln := range lines {
				data.NetWorthRows = append(data.NetWorthRows, nwRow{
					Account: ln.Account, Commodity: ln.Commodity,
					Units: ln.Units.FloatString(4), Value: ln.Value.FloatString(2),
					UsedCost: ln.UsedCost,
				})
			}
		}
	default:
		http.NotFound(w, r)
		return
	}
	s.render(w, "ledger.html", data)
}


func (s *Server) handleAccount(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("ledger")
	account := r.PathValue("account")
	account, _ = url.PathUnescape(account)
	// Wildcard may use slashes if any; join
	if account == "" {
		http.NotFound(w, r)
		return
	}
	l, err := engine.OpenLedger(s.Project, s.Prices, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	now := time.Now()
	timeStr := q.Get("time")
	pr, perr := period.Parse(timeStr, now)
	periodLabel := period.DisplayLabel(timeStr, now)

	asOf := pr.End
	asOfStr := ""
	if asOf.IsZero() {
		asOf = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	} else {
		asOfStr = asOf.Format("2006-01-02")
	}

	data := pageData{
		Title:       account,
		Page:        "account",
		LedgerName:  name,
		Ledgers:     engine.LedgerNames(s.Project),
		ProjectRoot: s.Project.Root,
		OpCurrency:  l.OpCurrency,
		Time:        timeStr,
		PeriodLabel: periodLabel,
		AsOf:        asOfStr,
		AccountName: account,
	}
	if perr != nil {
		data.Error = perr.Error()
		s.render(w, "account.html", data)
		return
	}

	// balances for this account
	bals := l.AccountBalances(account, asOf)
	var brow []balanceRow
	var comms []string
	for c := range bals {
		comms = append(comms, c)
	}
	sort.Strings(comms)
	for _, c := range comms {
		brow = append(brow, balanceRow{Account: account, Commodity: c, Amount: bals[c].FloatString(4)})
	}
	data.AccountBalances = brow

	// activity in period
	act := l.AccountActivity(account, pr.Start, pr.End)
	comms = nil
	for c := range act {
		comms = append(comms, c)
	}
	sort.Strings(comms)
	var arow []balanceRow
	for _, c := range comms {
		arow = append(arow, balanceRow{Account: account, Commodity: c, Amount: act[c].FloatString(4)})
	}
	data.AccountActivity = arow

	data.Journal = l.JournalForAccount(account, pr.Start, pr.End)
	s.render(w, "account.html", data)
}

func (s *Server) render(w http.ResponseWriter, name string, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.Tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func hasWarn(ds diag.List) bool {
	for _, d := range ds {
		if d.Severity == diag.Warn {
			return true
		}
	}
	return false
}

func buildBalances(bals map[string]map[string]*big.Rat) []balanceRow {
	var rows []balanceRow
	for a, byComm := range bals {
		for c, n := range byComm {
			rows = append(rows, balanceRow{
				Account: a, Commodity: c, Amount: n.FloatString(4),
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Account != rows[j].Account {
			return rows[i].Account < rows[j].Account
		}
		return rows[i].Commodity < rows[j].Commodity
	})
	return rows
}

func ratMapRows(m map[string]*big.Rat) []kvRow {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var rows []kvRow
	for _, k := range keys {
		rows = append(rows, kvRow{Key: k, Value: m[k].FloatString(4)})
	}
	return rows
}
