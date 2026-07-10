# Contapila — Specification

Status: MVP implementation in progress (tree-sitter grammar wired via modernc-tree-sitter/ccgo-tree-sitter).

Contapila is a self-contained **Go** reimplementation of a Beancount-class ledger engine plus a Fava-class read-only web UI: **one binary** (Cobra CLI + HTTP server with Go templates). Philosophy is **Helix, not Neovim**: good defaults, batteries included, no plugin system, poetic license on tooling.

---

## 1. Goals

| Goal | Detail |
|------|--------|
| Self-contained | Single Go binary; no Python Beancount at runtime; embedded CUE |
| Drop-in language | Parse and interpret real `.beancount` journals with high fidelity |
| Semantics bar **B** | Same balances/lots on **plugin-free** ledgers for supported features; document intentional divergences |
| Ready to use | Enough reports for a normal person: month-end balances, activity, P&L, net worth, `check` |
| Project-oriented | Git-like project root + conventional multi-ledger layout |

### Non-goals (MVP)

- Python plugin compatibility
- Full BQL / `bean-query` parity
- Fava editor / write-back
- Multi-user auth / remote multi-tenant hosting
- Tooling flag-compatibility with upstream Beancount CLIs
- Second/temporary parser before modernc grammar lands

---

## 2. Compatibility

### 2.1 Contract

- **In scope:** syntax + loader + booking + validation for the MVP directive set, without plugins.
- **Tooling:** poetic license (Cobra command names/flags need not match `bean-*`).
- **Plugins:** none. Unknown/unsupported constructs: **warn + skip** via `log/slog` where safe; **error** when continuing would corrupt inventory or lie about balances.

### 2.2 Intentional divergence: booking default

Upstream Beancount is lot-centric; average-cost is not its comfortable default.

Contapila defaults to **merged average-cost inventory** (model A below), aimed at real use (e.g. Receita Federal / preço médio for equities). Files that never set a booking policy may **disagree** with Beancount on inventories and gains. This is documented product policy, not an accident.

When / if `option "booking_method"` (or CUE equivalent) appears, honor it only for methods actually implemented; unsupported method → clear load/`check` error.

---

## 3. Product surface (MVP)

### 3.1 CLI (Cobra)

Illustrative commands (names may be refined at implement time):

| Command | Behavior |
|---------|----------|
| `contapila check [ledger]` | Validate; all ledgers if name omitted |
| `contapila balances [ledger]` | Balances as-of date |
| `contapila journal [ledger]` | Period journal / activity |
| `contapila pnl [ledger]` | Income vs expenses for a period |
| `contapila networth [ledger]` | Net worth as-of (shared prices) |
| `contapila web [ledger]` | Read-only HTTP UI |

Ledger argument is the **directory name** under the project root (see §4).

### 3.2 Web server

- **Read-only** viewer over the same `*Ledger` APIs as the CLI.
- **Go templates** (server-rendered).
- Default bind: **localhost** (no auth story in MVP).
- **Live reload** (watch ledger includes + prices + config): nice-to-have, not blocking first server slice.
- Out of MVP: in-browser edit, multi-user, write-back.

### 3.3 Reports

| Report | Question |
|--------|----------|
| Balances as-of | What is in each account on date D? |
| Journal / activity | What moved in period [from, to]? |
| P&L | Income vs expenses for the period (by account type prefix) |
| Net worth | Assets − liabilities in operating currency as-of D |
| Check | Opens/closes, balance assertions, booking errors, unbalanced txns |

---

## 4. Project layout

### 4.1 Root discovery

- Walk **upward from the process CWD** looking for `contapila.cue` (same idea as git finding `.git`).
- Nearest file wins; its directory is the **project root**.
- If none found → error (`not a contapila project`).
- **No `--config` flag.**

### 4.2 Convention

```text
<root>/
  contapila.cue           # required project marker; may be empty
  prices.beancount        # shared prices (empty/missing → warn)
  personal/
    main.beancount        # ledger name = "personal"
  empresa/
    main.beancount        # ledger name = "empresa"
  scratch/                # no main.beancount → ignore
```

| Rule | Behavior |
|------|----------|
| Config marker | `contapila.cue` at project root; **empty file is valid** (prelude supplies defaults) |
| Ledgers | Exactly one level: `<root>/*/main.beancount` |
| Ledger name | **Directory name** |
| Dir without `main.beancount` | **Ignore** |
| Recursive `**/main.beancount` | **No** |
| Root-level `main.beancount` | Not an entrypoint |
| Zero ledgers found | Error when running check/web/reports |
| Shared prices | `<root>/prices.beancount` |
| Prices empty or missing | **Warn**; empty price DB; conversion falls back per §8 |
| Includes | Paths relative to the **including file's directory**; globs allowed |
| Optional root commodities | `<root>/commodities.beancount` — often `include`d from ledgers |

**Price DB:** for a given (base, quote, date), **last write wins** when loading `prices.beancount` (and if the same day appears twice).

Ledgers may also `include "../prices.beancount"` / `include "../commodities.beancount"` so those directives appear in the ledger stream (real-repo style). The project-level PriceDB still comes from the root prices file path.

### 4.3 Isolation and sharing

| Concern | Scope |
|---------|--------|
| Inventory, transactions, pads, balance assertions, accounts (`open`/`close`) | **Per ledger** (isolated) |
| Commodity policy (precision, tolerance, class) | **Shared** (project CUE) |
| Market `price` directives | **Shared** (`prices.beancount` → one PriceDB for all ledgers) |

Multiple entrypoints are **named parallel ledgers**, never merged into one inventory.

### 4.4 Account documents (`<ledger>/docs/by-account`)

Documents are **per ledger** (same isolation as inventory). Layout:

```text
<root>/<ledger>/docs/by-account/<seg>/<seg>/…/<filename>
```

Account components become **subdirectories** (`:` → `/`):

Example: ledger `personal`, account `Assets:BR:Alfa:ContaCorrente` →  
`personal/docs/by-account/Assets/BR/Alfa/ContaCorrente/`.

**Filenames** start with a calendar date in **`yyyymmdd`**, then an optional
separator and rest of the name:

```text
20240301_statement.txt
20230810-INV-001.pdf
```

On ledger open the host walks **`<that-ledger>/docs/by-account/**`** and synthesizes
`document` directives (date from filename prefix, account from path). Explicit
`document` lines in that ledger’s journal merge in; same path prefers the explicit
directive. Account web UI lists documents and serves files under `/docfile/<ledger>/docs/…`.

Metadata `document: "…"` on transactions/postings is **stored** on the journal AST
and expanded into the ledger’s document list at open (same merge rules as filesystem
synth; path prefers explicit `document` directive). Not injected into CUE.

### 4.5 Ledgers in CUE (discovered) and inter-ledger links

On project open the host **looks up** `<root>/*/main.beancount` and injects a
generated CUE fragment (workspaced-style host data):

```cue
ledgers: close({
  personal: {name: "personal", main: "<abs>/personal/main.beancount"}
  acme:     {name: "acme",     main: "<abs>/acme/main.beancount"}
  // …
})
```

Types live in the embedded **prelude**:

| Type | Meaning |
|------|---------|
| `#Ledger` | `{name: #LedgerID, main: string}` — one discovered ledger |
| `#LedgerID` | Directory-name shape: `^[A-Za-z][A-Za-z0-9_-]*$` |
| `#LedgerName` | `or([for n, _ in ledgers {n}])` — **keys of the injected map only** |
| `#LedgerRef` / `#LedgerLink` | Cross-ledger endpoints using `#LedgerName` |

User `contapila.cue` does **not** list ledgers; inventing keys under `ledgers` fails (struct is `close`d). Links:

```cue
links: [{
  name: "acme-profit-distribution"
  from: {ledger: "acme", account: "Equity:DistribuicaoLucros"}
  to:   {ledger: "personal", account: "Income:Ativo:BR:DistribuicaoLucros:Acme"}
}]
```

**MVP:** CUE validates ledger **names** against discovery; `check` does **not** reconcile balances yet.

---

## 5. Architecture

### 5.1 Public API shape

- Open project from CWD → project handle (root, config, PriceDB, ledger names).
- Open/load each named ledger → `*Ledger`.
- Surfaces (CLI, HTTP) call only project/`Ledger` methods — no parsing in handlers.

Suggested capabilities on `*Ledger`:

- `Check() error` (hard errors fail; warnings via slog)
- `Balances(asOf)`
- `Journal(from, to)`
- `PnL(from, to)`
- `NetWorth(asOf)` — uses shared PriceDB + operating currency rules

### 5.2 Pipeline (per ledger)

```text
resolve project root (contapila.cue)
load prices.beancount → PriceDB          # once per project
for each ledger name:
  parse main.beancount + include graph   # tree-sitter (deferred)
  split config-ish directives vs stream
  encode config facts → CUE
  unify: prelude & contapila.cue & ledgerFacts
  decode RuntimeConfig
  apply stream: tags/meta (none in MVP), booking, pads, assertions
  reports
```

Internal stages are separate packages; the public surface stays a deep module (single entry, hidden stages).

### 5.3 Parser bootstrap

- **Wait** for Beancount grammar via [modernc / ccgo-tree-sitter](https://github.com/modernc-tree-sitter/ccgo-tree-sitter).
- No temporary hand parser, no Python subprocess, no alternate long-term cgo binding as the product path.
- Design freezes the AST/config/booking contracts so the grammar drops into one adapter.

### 5.4 Numeric types

- Engine amounts and costs: **`math/big.Rat`** (never `float64` for money).
- Display/tolerance from commodity policy (§7).

---

## 6. CUE config plane

### 6.1 Runtime

- **Embed** CUE (`cuelang.org/go`), workspaced-style — no `cue` CLI required for normal use.
- Shipped **prelude** (schema, defaults, asset-class short-circuits) unified with user `contapila.cue` and **ledger-derived config facts**.
- **CUE decides** conflicts on the config plane (unification failure → load error). Do not implement ad hoc “who wins” tables in Go for config.

### 6.2 What goes into CUE

| In CUE (config plane) | Not in CUE |
|-----------------------|------------|
| Options (e.g. operating currency) | Transactions / postings |
| Commodities + precision/tolerance/class | Full `price` time series (volume) |
| Price **pair inventory** (`price_pairs` inject) | Individual price points / rates |
| Per-ledger account open/close facts | `balance`, `pad` |
| Project overlays in `contapila.cue` | `note`, `event`, journal stream |
| Prelude defaults | Include graph resolution (Go first) |
| (nothing for txn meta) | Txn/posting `key_value` metadata (Go journal only) |

### 6.3 Dual definition

- Commodities (and policy) may be declared in CUE and/or `commodity` directives; facts are encoded and **unified in CUE**.
- Accounts are **per ledger**: from that ledger’s `open`/`close` (and only that ledger’s facts in the per-ledger unify).
- Transactions are **never** executed or stored as CUE.

### 6.4 Minimal user file

```cue
// contapila.cue — valid empty project marker
// Optional overlays, e.g.:
// commodities: { BRL: {class: "fiat"}, BTC: {class: "crypto"} }
```

### 6.5 Defaults (prelude)

- Default **precision: 5** decimal places.
- Asset classes (illustrative; exact prelude schema at implement time) override precision (e.g. fiat → 2, crypto → 8).
- Default **tolerance**: half unit in the last place of precision, unless overridden per commodity/class.
- Undeclared commodities in journals: usable with prelude defaults (precision 5) unless stricter policy is added later.

---

## 7. Booking and inventory (model A)

### 7.1 Inventory model

- Per **account + commodity**: a **single merged average-cost** position (not multi-lot history).
- Lot theatre (FIFO/LIFO/STRICT multi-lot) is out of MVP unless explicitly reintroduced later.

### 7.2 Buys (increases)

- Require **explicit cost** `{...}` on inventory increases.
- New units merge into the average cost of the position.

### 7.3 Sells (reductions) — Shape 4

- Cost braces may be **omitted** on a reduction; engine books cost at **current average**.
- Prefer **`@@` total proceeds** for broker-style fills; support `@` unit price as well.
- Multi-stock sells: **one posting per commodity**; sugar is per line, not one average for the whole txn.

Example:

```beancount
2024-03-10 * "Sell PETR4 + VALE3"
  Assets:Broker:PETR4  -40 PETR4 @@ 1400.00 BRL
  Assets:Broker:VALE3  -20 VALE3 @@ 1400.00 BRL
  Assets:Cash           2800.00 BRL
  Income:Gains
```

### 7.4 Residual leg (no magic)

- At most **one** posting with **missing amount** absorbs the remainder (typically gains).
- **No** implicit default gains account; **no** auto-inserted legs.
- Unbalanced transaction without an empty residual posting → **error**.
- Explicit `{cost}` on a sell that is not the current average (beyond tolerance) → **error**.
- Selling more units than inventory → **warn** (do not invent inventory; check still passes).

---

## 8. Operating currency and prices

### 8.1 Operating currency

1. Prefer explicit option / CUE option for `operating_currency`.
2. If missing: **warn**, then after includes are resolved, walk directives and take the commodity from the **first transaction that carries a posting amount commodity**.
3. If still none: currency-denominated reports error at report time.

### 8.2 Shared PriceDB

- Built from project `prices.beancount`.
- All ledgers consult the same DB for conversion / net worth.

### 8.3 Price lookup for as-of date D

Market conversion (net worth, charts, P&L when op currency is set) uses **PriceDB only**:

1. Direct pair `base→quote` on or before D (walk back: last price ≤ D).
2. Else inverse of `quote→base`.
3. Else one intermediate hop (e.g. `SPDW→USD→BRL`), each leg direct or inverse, both ≤ D.
4. If still missing: **warn**; market value is **0** (no cost-basis fallback).
5. Do not silently use future prices for past as-of dates.

Inventory cost basis (average cost, model A) remains for booking/gains; it is **not** used to value net worth.

### 8.4 Net worth

- Include **Assets** and **Liabilities** only (not Equity/Income/Expenses).
- Convert positions to operating currency with §8.3 using **signed** unit balances
  (Beancount: liabilities are usually credit → negative units; do **not** flip sign again).
- Valuation is **market only**; unpriced positions show as 0 with a UI/CLI “no px” marker.

---

## 9. Directives (MVP)

| Directive | MVP | Plane |
|-----------|-----|--------|
| `option` | yes | → CUE |
| `include` (+ globs) | yes | Go load |
| `commodity` | yes | → CUE |
| `open` / `close` | yes | → CUE (per ledger) |
| `*` / `!` transactions, postings | yes | Go |
| metadata on `open` / `commodity` | yes (stored; CUE `#Account` / `#Commodity` tandem) | Go + CUE |
| metadata on `price` | yes (stored on PriceDB points) | Go |
| metadata on txn/posting | yes (journal stream only; **not** CUE) | Go |
| posting `closing: TRUE` | yes — expands to `balance 0` + `close` next day for that account/commodity | Go |
| cost `{}`, price `@` / `@@` | yes | Go |
| empty residual posting | yes | Go |
| `price` | yes (shared file → PriceDB; CUE `price_pairs` inventory only) | Go + CUE |
| `balance` | yes | Go |
| `pad` | yes | Go |
| `note` | yes | Go (store/display) |
| `event` | yes | Go (store/display) |
| `document` | yes (store/display; also synthesized from `<ledger>/docs/by-account`) | Go |
| `query` | no | — |
| `pushtag` / `poptag` / `pushmeta` / `popmeta` | no | — |
| `plugin` | no | — |
| `custom` / unknown | warn + skip | — |

---

## 10. Diagnostics severity

Use **`log/slog`** for warnings. `check` fails only on **errors**.

| Event | Severity |
|-------|----------|
| Unopened account used | **warn** + allow |
| Posting after `close` | **error** |
| Posting `closing: TRUE` without units | **error** |
| Posting `closing: TRUE` but `close` already written | **warn** (skip synthetic close; still assert `balance 0`) |
| Duplicate `open` same account | **error** |
| Unbalanced txn, no residual leg | **error** |
| Failed `balance` assertion | **error** |
| Over-sell (no inventory / not enough units) | **warn** (skip inventing inventory) |
| Bad average cost on reduce | **error** |
| Unknown `option` | **warn** |
| `include` literal path missing | **error** |
| `include` glob, zero matches | **warn** |
| Include cycle | **error** |
| Double-include same realpath | skip (dedupe); optional single warn |
| Missing `operating_currency` (inferred) | **warn** |
| Price missing for market conversion (value 0) | **warn** |
| `prices.beancount` empty/missing | **warn** |
| Unsupported / unknown directive | **warn** + skip |
| CUE config unify failure | **error** |

---

## 11. Includes

- Relative paths resolved against the **directory of the file containing the `include`**.
- Absolute paths allowed.
- Globs supported; zero matches → warn.
- Cycles → error.
- File identity for cycle/dedupe: realpath.

---

## 12. Verification strategy

| Phase | Method |
|-------|--------|
| Early | **Dogfood** on real multi-ledger projects |
| Ongoing | **Golden corpus** of fixtures (inputs + expected balances/errors/net worth) checked into the repo |
| Optional later | Python Beancount oracle for selected fixtures — not required for definition of done |

Golden fixtures should emphasize average-cost stock buys/partial sells, pads, includes, shared prices, multi-ledger isolation, and residual gains legs — not only STRICT lot puzzles.

---

## 13. Implementation order (when unblocked)

1. Repo scaffold: Go module, Cobra, embed CUE prelude, project root discovery + layout scan.
2. Config plane: prelude schema, empty `contapila.cue`, ledger config fact encoding (mock AST ok).
3. Parser adapter behind `Parse` once modernc grammar is available.
4. Loader (includes, streams) + booking model A + `check`.
5. PriceDB + reports (balances, journal, P&L, net worth).
6. CLI commands.
7. Read-only web server; live reload later.
8. Golden corpus expansion alongside dogfood.

---

## 14. Open at implement time (non-blocking)

- Exact CUE prelude schema field names and class set.
- Exact Cobra flag set (dates, output formats).
- HTTP routes and template structure.
- Whether `contapila init` scaffolds root + empty cue + sample ledger dirs.
- Tolerance combination rules when a transaction touches multiple commodities.
- Optional `option "booking_method"` surface once more methods exist.

---

## 15. Summary one-liner

**Contapila** = conventional multi-ledger Beancount project (`contapila.cue` + `*/main.beancount` + `prices.beancount`) with embedded CUE policy, average-cost inventory, and a single Go binary for check/reports/read-only web — semantics-first, plugins never, tree-sitter when modernc is ready.
