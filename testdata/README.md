# testdata

## `example/`

Runnable **contapila project** used to dogfood layout and semantics assumptions:

| Path | Role |
|------|------|
| `contapila.cue` | Project marker + commodity precision |
| `prices.beancount` | Shared price DB for all ledgers |
| `personal/` | Main dogfood ledger (includes, avg-cost, pad, multi-commodity) |
| `empresa/` | Second ledger (isolation; same ticker ≠ same inventory) |
| `scratch/` | Dir without `main.beancount` → must be ignored |

```bash
cd testdata/example
contapila status
contapila check
contapila balances
contapila web
```

What this project is meant to stress:

- Root discovery via `contapila.cue`
- Ledger discovery: `*/main.beancount` only
- `include` + glob includes
- Shared `prices.beancount`
- Average-cost equity buys / partial sell + residual gains
- Pad + balance assertion
- Multi-ledger isolation
- Multi-commodity amounts (BRL, USD, X)
- Transaction strings: narration-only (`* "Lunch"`) and payee+narration (`* "Payee" "Narration"`)

## `golden/`

Reserved for expected check/balances/net-worth snapshots (corpus expansion).
