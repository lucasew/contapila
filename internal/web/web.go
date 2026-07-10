package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"math/big"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"cuelang.org/go/cue"

	"github.com/lucasew/contapila-go/internal/ast"
	"github.com/lucasew/contapila-go/internal/diag"
	docsutil "github.com/lucasew/contapila-go/internal/docs"
	"github.com/lucasew/contapila-go/internal/engine"
	"github.com/lucasew/contapila-go/internal/period"
	"github.com/lucasew/contapila-go/internal/prices"
	"github.com/lucasew/contapila-go/pkg/project"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed all:static
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
		"eq":          func(a, b string) bool { return a == b },
		"queryEscape": url.QueryEscape,
		"jsonStr": func(s string) template.JS {
			b, err := json.Marshal(s)
			if err != nil {
				return template.JS(`""`)
			}
			return template.JS(b)
		},
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
		"commodityURL": func(ledger, commodity, timeFilter string) string {
			u := "/l/" + url.PathEscape(ledger) + "/commodity/" + url.PathEscape(commodity)
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
	// Ledger docs: URL /docfile/<ledger>/docs/by-account/... → <root>/<ledger>/docs/...
	mux.HandleFunc("GET /docfile/{path...}", s.handleDocFile)
	mux.HandleFunc("GET /{$}", s.handleIndex)
	// Named entity routes before generic page so "account"/"commodity" are not page names.
	mux.HandleFunc("GET /l/{ledger}/account/{account...}", s.handleAccount)
	mux.HandleFunc("GET /l/{ledger}/commodity/{commodity...}", s.handleCommodity)
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
	IncomeRows   []balanceRow
	ExpenseRows  []balanceRow
	NetWorthRows []nwRow
	NetWorthTot  string
	AsOf         string
	Time        string
	PeriodLabel string
	Error       string
	// Account page
	AccountName       string
	AccountBalances   []balanceRow
	AccountActivity   []balanceRow
	AccountDocs       []docRow
	AccountMeta       []metaKV
	AccountCurrencies []string
	// Documents is the ledger documents report (/documents) list.
	Documents []docRow
	// Prices is the shared price-pairs report (/prices).
	PriceSeries []priceSeriesRow
	// Commodity page
	CommodityName     string
	CommodityBalances []balanceRow // Account + Amount for this commodity
	CommodityActivity []balanceRow
	CommodityMeta     []metaKV
	CommodityPrices   []pricePointRow // price history for this base commodity
	// Charts (uPlot): ChartJSON is safe JSON embedded in the page.
	ChartID    string
	ChartTitle string
	ChartJSON  template.JS
	NeedCharts bool
}

// priceSeriesRow is one base/quote pair summary on the prices report.
type priceSeriesRow struct {
	Base, Quote       string
	Count             int
	FirstDate         string
	LastDate          string
	LastRate          string
	LastMeta          []metaKV
}

// pricePointRow is one observation on a commodity's price history.
type pricePointRow struct {
	Date     string
	Quote    string
	Rate     string
	Metadata []metaKV
}

type metaKV struct {
	Key, Value string
}

type docRow struct {
	Date      string
	Account   string // owning account (for sidebar)
	Path      string // project-relative
	Href      string // /docs/... URL
	Name      string // base filename
	Synthetic bool
}

type balanceRow struct {
	Account   string
	Name      string // short label (last segment) for tree display
	Commodity string
	Amount    string
	Depth     int    // hierarchical indent (P&L tree)
	IsRollup  bool   // parent total row
	PadLeft   string // e.g. "1.5rem" for template indent
}

type nwRow struct {
	Account   string
	Path      string // collapse key
	Name      string // leaf segment
	Commodity string
	Units     string
	Value     string
	UsedCost  bool
	Depth     int
	IsRollup  bool
	PadLeft   string
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
			inc, exp := l.PnLTree(pr.Start, pr.End)
			data.IncomeRows = buildPnLRows(inc)
			data.ExpenseRows = buildPnLRows(exp)
			from, to := pr.Start, pr.End
			kind := period.ChartBin(timeStr, pr)
			bars := l.PnLBars(from, to, kind)
			if js, err := chartBarsJSON(bars, l.OpCurrency); err == nil && js != "" {
				data.NeedCharts = true
				data.ChartID = "chart-pnl"
				data.ChartTitle = "Income vs expenses"
				data.ChartJSON = js
			}
		}
	case "networth":
		tree, total, err := l.NetWorthTree(asOf)
		if err != nil {
			data.Error = err.Error()
		} else {
			data.NetWorthTot = total.FloatString(2)
			data.NetWorthRows = buildNetWorthRows(tree)
			// Event series over filter range (or full history if open).
			pts, serr := l.NetWorthSeries(pr.Start, pr.End)
			if serr == nil {
				if js, jerr := chartLineJSON(pts, l.OpCurrency, "Net worth"); jerr == nil && js != "" {
					data.NeedCharts = true
					data.ChartID = "chart-networth"
					data.ChartTitle = "Net worth over time"
					data.ChartJSON = js
				}
			}
		}
	case "documents":
		data.Documents = documentRows(l.Documents)
	case "prices":
		data.PriceSeries = priceSeriesRows(s.Prices)
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
	data.AccountDocs = documentRows(l.DocumentsForAccount(account))
	if info, ok := l.Accounts[account]; ok {
		data.AccountMeta = metaRows(info.Metadata)
		data.AccountCurrencies = info.Currencies
	}
	if pts, err := l.AccountSeries(account, pr.Start, pr.End); err == nil {
		if js, jerr := chartLineJSON(pts, l.OpCurrency, "Balance"); jerr == nil && js != "" {
			data.NeedCharts = true
			data.ChartID = "chart-account"
			data.ChartTitle = "Balance over time"
			data.ChartJSON = js
		}
	}
	s.render(w, "account.html", data)
}

// chartLineJSON builds uPlot line payload (event series, op currency).
func chartLineJSON(pts []engine.SeriesPoint, currency, label string) (template.JS, error) {
	if len(pts) == 0 {
		return "", nil
	}
	type payload struct {
		Kind     string    `json:"kind"`
		Currency string    `json:"currency"`
		Label    string    `json:"label"`
		X        []int64   `json:"x"`
		Y        []float64 `json:"y"`
	}
	p := payload{Kind: "line", Currency: currency, Label: label, X: make([]int64, len(pts)), Y: make([]float64, len(pts))}
	for i, pt := range pts {
		p.X[i] = pt.Date.UTC().Unix()
		p.Y[i] = ratFloat(pt.Value)
	}
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return template.JS(b), nil
}

// chartBarsJSON builds uPlot diverging bar payload.
// X is ordinal (0..n-1), not unix time — avoids time-scale bar width/overlap artifacts.
func chartBarsJSON(bars []engine.BarPoint, currency string) (template.JS, error) {
	if len(bars) == 0 {
		return "", nil
	}
	type payload struct {
		Kind     string    `json:"kind"`
		Currency string    `json:"currency"`
		X        []float64 `json:"x"`
		Labels   []string  `json:"labels"`
		Income   []float64 `json:"income"`
		Expense  []float64 `json:"expense"`
	}
	p := payload{
		Kind: "bars", Currency: currency,
		X: make([]float64, len(bars)), Labels: make([]string, len(bars)),
		Income: make([]float64, len(bars)), Expense: make([]float64, len(bars)),
	}
	for i, b := range bars {
		p.X[i] = float64(i)
		p.Labels[i] = b.Label
		// Per-bin flow magnitudes (not cumulative across bins).
		p.Income[i] = ratFloat(b.Income)
		p.Expense[i] = ratFloat(b.Expense)
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return template.JS(raw), nil
}

func ratFloat(r *big.Rat) float64 {
	if r == nil {
		return 0
	}
	f, _ := r.Float64()
	return f
}

func documentRows(docs []ast.Document) []docRow {
	var rows []docRow
	for _, d := range docs {
		p := filepath.ToSlash(d.Path)
		p = strings.TrimPrefix(p, "/")
		href := ""
		// serve only <ledger>/docs/... via /docfile/
		if docsutil.IsLedgerDocPath(p) {
			href = "/docfile/" + p
		}
		rows = append(rows, docRow{
			Date:      d.Date.Format("2006-01-02"),
			Account:   d.Account,
			Path:      p,
			Href:      href,
			Name:      path.Base(p),
			Synthetic: d.Synthetic,
		})
	}
	return rows
}

// handleDocFile serves project-relative paths under <ledger>/docs/ only.
func (s *Server) handleDocFile(w http.ResponseWriter, r *http.Request) {
	rel := filepath.ToSlash(r.PathValue("path"))
	rel = strings.TrimPrefix(path.Clean("/"+rel), "/")
	if !docsutil.IsLedgerDocPath(rel) {
		http.NotFound(w, r)
		return
	}
	full := filepath.Join(s.Project.Root, filepath.FromSlash(rel))
	// Contain under project root
	root := s.Project.Root
	absFull, err1 := filepath.Abs(full)
	absRoot, err2 := filepath.Abs(root)
	if err1 != nil || err2 != nil {
		http.NotFound(w, r)
		return
	}
	sep := string(filepath.Separator)
	if absFull != absRoot && !strings.HasPrefix(absFull, absRoot+sep) {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, full)
}

func (s *Server) handleCommodity(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("ledger")
	commodity := r.PathValue("commodity")
	commodity, _ = url.PathUnescape(commodity)
	if commodity == "" {
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
		Title:         commodity,
		Page:          "commodity",
		LedgerName:    name,
		Ledgers:       engine.LedgerNames(s.Project),
		ProjectRoot:   s.Project.Root,
		OpCurrency:    l.OpCurrency,
		Time:          timeStr,
		PeriodLabel:   periodLabel,
		AsOf:          asOfStr,
		CommodityName: commodity,
	}
	if perr != nil {
		data.Error = perr.Error()
		s.render(w, "commodity.html", data)
		return
	}

	bals := l.CommodityBalances(commodity, asOf)
	var accts []string
	for a := range bals {
		accts = append(accts, a)
	}
	sort.Strings(accts)
	for _, a := range accts {
		data.CommodityBalances = append(data.CommodityBalances, balanceRow{
			Account: a, Commodity: commodity, Amount: bals[a].FloatString(4),
		})
	}

	act := l.CommodityActivity(commodity, pr.Start, pr.End)
	accts = nil
	for a := range act {
		accts = append(accts, a)
	}
	sort.Strings(accts)
	for _, a := range accts {
		data.CommodityActivity = append(data.CommodityActivity, balanceRow{
			Account: a, Commodity: commodity, Amount: act[a].FloatString(4),
		})
	}

	data.Journal = l.JournalForCommodity(commodity, pr.Start, pr.End)
	if info, ok := l.Commodities[commodity]; ok {
		data.CommodityMeta = metaRows(info.Metadata)
	}
	// Overlay CUE commodity fields (name, asset-class, …) when present.
	if s.Project.Config != nil {
		data.CommodityMeta = mergeCUECommodityMeta(s.Project.Config.Value, commodity, data.CommodityMeta)
	}
	data.CommodityPrices = commodityPriceRows(s.Prices, commodity)
	s.render(w, "commodity.html", data)
}

func priceSeriesRows(db *prices.DB) []priceSeriesRow {
	if db == nil {
		return nil
	}
	series := db.AllSeries()
	out := make([]priceSeriesRow, 0, len(series))
	for _, s := range series {
		if len(s.Points) == 0 {
			continue
		}
		first, last := s.Points[0], s.Points[len(s.Points)-1]
		row := priceSeriesRow{
			Base:      s.Base,
			Quote:     s.Quote,
			Count:     len(s.Points),
			FirstDate: first.Date.Format("2006-01-02"),
			LastDate:  last.Date.Format("2006-01-02"),
			LastRate:  last.Rate.FloatString(6),
			LastMeta:  metaRows(last.Metadata),
		}
		out = append(out, row)
	}
	return out
}

func commodityPriceRows(db *prices.DB, base string) []pricePointRow {
	if db == nil {
		return nil
	}
	var out []pricePointRow
	for _, s := range db.SeriesForBase(base) {
		// newest first for history reading
		for i := len(s.Points) - 1; i >= 0; i-- {
			p := s.Points[i]
			out = append(out, pricePointRow{
				Date:     p.Date.Format("2006-01-02"),
				Quote:    s.Quote,
				Rate:     p.Rate.FloatString(6),
				Metadata: metaRows(p.Metadata),
			})
		}
	}
	return out
}

func metaRows(m ast.Metadata) []metaKV {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]metaKV, 0, len(keys))
	for _, k := range keys {
		out = append(out, metaKV{Key: k, Value: m[k]})
	}
	return out
}

// mergeCUECommodityMeta adds missing keys from contapila.cue commodities.<name>.
func mergeCUECommodityMeta(cfg cue.Value, commodity string, rows []metaKV) []metaKV {
	if !cfg.Exists() {
		return rows
	}
	// commodities."B3_PETR4" path for odd names
	v := cfg.LookupPath(cue.ParsePath("commodities." + strconv.Quote(commodity)))
	if !v.Exists() {
		v = cfg.LookupPath(cue.ParsePath("commodities." + commodity))
	}
	if !v.Exists() {
		return rows
	}
	have := map[string]bool{}
	for _, r := range rows {
		have[r.Key] = true
	}
	it, err := v.Fields()
	if err != nil {
		return rows
	}
	var extra []metaKV
	for it.Next() {
		sel := it.Selector()
		k := sel.String()
		if k == "precision" || have[k] {
			continue
		}
		s, err := it.Value().String()
		if err != nil {
			// allow int precision already skipped; try other concrete forms
			continue
		}
		extra = append(extra, metaKV{Key: k, Value: s})
		have[k] = true
	}
	if len(extra) == 0 {
		return rows
	}
	sort.Slice(extra, func(i, j int) bool { return extra[i].Key < extra[j].Key })
	return append(rows, extra...)
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

func buildPnLRows(lines []engine.PnLLine) []balanceRow {
	rows := make([]balanceRow, 0, len(lines))
	for _, ln := range lines {
		pad := ""
		if ln.Depth > 0 {
			// 0.75rem per level
			pad = strconv.FormatFloat(float64(ln.Depth)*0.75, 'f', 2, 64) + "rem"
		}
		name := ln.Name
		if name == "" {
			name = ln.Account
		}
		rows = append(rows, balanceRow{
			Account:   ln.Account,
			Name:      name,
			Commodity: ln.Commodity,
			Amount:    ln.Amount.FloatString(2),
			Depth:     ln.Depth,
			IsRollup:  ln.IsRollup,
			PadLeft:   pad,
		})
	}
	return rows
}

func buildNetWorthRows(lines []engine.NetWorthTreeLine) []nwRow {
	rows := make([]nwRow, 0, len(lines))
	for _, ln := range lines {
		pad := ""
		if ln.Depth > 0 {
			pad = strconv.FormatFloat(float64(ln.Depth)*0.75, 'f', 2, 64) + "rem"
		}
		name := ln.Name
		if name == "" {
			name = ln.Account
		}
		path := ln.Path
		if path == "" {
			path = ln.Account
		}
		units := ""
		if ln.Units != nil {
			units = ln.Units.FloatString(4)
		}
		rows = append(rows, nwRow{
			Account:   ln.Account,
			Path:      path,
			Name:      name,
			Commodity: ln.Commodity,
			Units:     units,
			Value:     ln.Value.FloatString(2),
			UsedCost:  ln.UsedCost,
			Depth:     ln.Depth,
			IsRollup:  ln.IsRollup,
			PadLeft:   pad,
		})
	}
	return rows
}


