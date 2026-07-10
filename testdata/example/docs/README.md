# docs/

Contract (SPEC §4.4): account-linked documents live under a directory tree that
mirrors the account hierarchy (`:` → path separator):

```text
docs/by-account/<segment>/<segment>/…/
```

Example: `Assets:BR:Alfa:ContaCorrente` → `docs/by-account/Assets/BR/Alfa/ContaCorrente/`.

**File names** start with a date in `yyyymmdd` (optional rest after `_` or `-`):

```text
20240315_statement.txt
20230810-INV-001.txt
```

Transactions may also set metadata `document: "docs/by-account/..."`.
The engine does not yet render these links in the web UI; the layout is the contract.
