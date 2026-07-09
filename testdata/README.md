# testdata

## `example/`

Fat **contapila project** (~1,000,008 transactions) for UI dogfood, engine scale, and anonymized BR-shaped realism.

| Ledger | Role |
|--------|------|
| `personal` | Household high-volume: banks, broker, cards-by-month, CDB maturities, multi-commodity |
| `acme` | Small company: AR/AP, sales, payroll |
| `ong` | Nonprofit: grants, donations, programs |
| `smuggle` | Fictional inventory stress (`CIGPK` packs, avg-cost) |
| `scratch/` | No `main.beancount` → ignored |

### Layout

- `contapila.cue` — commodity precision
- `prices.beancount` — shared, irregular points per month (2021-07→2026-07)
- Domain includes (personal expenses split by year for size)
- Fiction brands; topology mirrors real BR books without doxxing

### Commodities

`BRL`, `USD`, `B3_PETR4`, `B3_WEGE3`, `B3_BBAS3`, `B3_MXRF11`, `TDBR_SELIC_2029`, `BTC`, `SPDW`, `CIGPK`

```bash
cd testdata/example
contapila status
contapila check
contapila balances
contapila web
```

## `golden/`

Reserved for expected check/balances snapshots.
