# testdata

| Path | Role |
|------|------|
| **`example/`** | Default dogfood project: **depth over volume**. Multi-ledger, rich chart, domain includes, ~few hundred txns. Use this day-to-day. |
| **`kitchensink/`** | Scale corpus: same multi-ledger ideas, **≥100k–1M transactions** for parser/booking/UI load. Not for reading by hand. |
| **`golden/`** | Reserved for expected check/balances snapshots. |

## `example/` (deep)

Ledgers: `personal` · `acme` · `ong` · `smuggle` (+ ignored `scratch/`).

| Ledger | What it exercises |
|--------|-------------------|
| personal | BR-shaped chart, cards-by-month, CDB maturities, avg-cost equities/FII/crypto/USD, FX+IOF, family loan, soft AR, payee+narration |
| acme | AR invoices + collections, AP suppliers/tax, payroll, profit distributions |
| ong | Grant receivable → deferred → income, donations, programs |
| smuggle | Unit inventory (`CIGPK`), multi-cost buys, warehouse/transit/cache, residual gains, confiscation |

```bash
cd testdata/example
contapila status
contapila check
contapila balances personal
contapila web
```

## `kitchensink/` (volume)

Same ledgers/topology spirit, generated at high volume for stress.

```bash
cd testdata/kitchensink
contapila check   # expect minutes on ~1M txns
```

Regenerate kitchensink (optional):

```bash
python3 /path/to/gen_fat_example.py 1000000
# then move output into testdata/kitchensink
```
