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
		"intervalURL": func(ledger, page, kind, current string) string {
			now := time.Now()
			next := period.SetInterval(current, now, kind)
			return "/l/" + url.PathEscape(ledger) + "/" + url.PathEscape(page) + "?time=" + url.QueryEscape(next)
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
	Time         string
	PeriodLabel  string
	TimePrev     string
	TimeNext     string
	Interval     string
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

	// Optional: change interval via ?interval=month
	if iv := q.Get("interval"); iv != "" {
		timeStr = period.SetInterval(timeStr, now, iv)
		// redirect to clean URL with only time=
		http.Redirect(w, r, "/l/"+url.PathEscape(name)+"/"+url.PathEscape(page)+"?time="+url.QueryEscape(timeStr), http.StatusFound)
		return
	}

	pr, perr := period.Parse(timeStr, now)
	interval := period.Kind(timeStr, now)
	periodLabel := period.DisplayLabel(timeStr, now)
	timePrev, _ := period.Shift(timeStr, now, -1)
	timeNext, _ := period.Shift(timeStr, now, 1)
	if timeStr == "" {
		// Prev/next from empty jump into current month then step
		timePrev, _ = period.Shift(period.At(period.KindMonth, now), now, -1)
		timeNext = period.At(period.KindMonth, now)
	}

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
		TimePrev:    timePrev,
		TimeNext:    timeNext,
		Interval:    interval,
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
