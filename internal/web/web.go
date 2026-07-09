package web

import (
	"embed"
	"fmt"
	"html/template"
	"math/big"
	"net/http"
	"sort"
	"time"

	"github.com/lucasew/contapila-go/internal/diag"
	"github.com/lucasew/contapila-go/internal/engine"
	"github.com/lucasew/contapila-go/internal/prices"
	"github.com/lucasew/contapila-go/pkg/project"
)

//go:embed templates/*
var templateFS embed.FS

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
		"severity": func(s string) string {
			if s == "" {
				return "—"
			}
			return s
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
	mux.HandleFunc("GET /{$}", s.handleIndex)
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
	From         string
	To           string
	Error        string
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
	asOfStr := q.Get("as-of")
	asOf, _ := engine.ParseDate(asOfStr)
	if asOf.IsZero() {
		asOf = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
		asOfStr = ""
	} else {
		asOfStr = asOf.Format("2006-01-02")
	}
	fromStr := q.Get("from")
	toStr := q.Get("to")
	from, _ := engine.ParseDate(fromStr)
	to, _ := engine.ParseDate(toStr)

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
		From:        fromStr,
		To:          toStr,
	}

	switch page {
	case "check":
		// fields already set
	case "balances":
		data.BalanceRows = buildBalances(l.BalancesAsOf(asOf))
	case "journal":
		data.Journal = l.Journal(from, to)
	case "pnl":
		p := l.PnL(from, to)
		data.IncomeRows = ratMapRows(p.Income)
		data.ExpenseRows = ratMapRows(p.Expenses)
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
	var accts []string
	for a := range bals {
		accts = append(accts, a)
	}
	sort.Strings(accts)
	var rows []balanceRow
	for _, a := range accts {
		var cs []string
		for c := range bals[a] {
			cs = append(cs, c)
		}
		sort.Strings(cs)
		for _, c := range cs {
			rows = append(rows, balanceRow{
				Account: a, Commodity: c, Amount: bals[a][c].FloatString(4),
			})
		}
	}
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
