# testdata

| Path | Role |
|------|------|
| **`example/`** | Default dogfood: **depth over volume**. Multi-ledger, real-repo surface (includes, pad/balance/close, meta, docs). |
| **`kitchensink/`** | Scale corpus (~1M txns). Untouched by example depth work. |
| **`golden/`** | Reserved for expected snapshots. |

## `example/` (deep)

```bash
contapila -C testdata/example check   # expects OK + many metadata/query warns
contapila -C testdata/example web
```

| Ledger | Exercises |
|--------|-----------|
| personal | Root `include` of commodities+prices; cards; CDB maturities; avg-cost equities; pad→balance; close temp account; tags; meta (warn); mirrored Acme distributions |
| acme | AR/AP; root includes; pad/balance; close clearing; invoice meta |
| ong | Grants deferred |
| smuggle | `CIGPK` inventory |

Also: `docs/by-account/…` (SPEC §4.4), `links` in `contapila.cue` (SPEC §4.5, not enforced), `query`/`custom`/`document` (warn+skip).

## `kitchensink/`

High volume only — do not hand-edit for surface features.
