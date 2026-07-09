# Rules for Jules (and other AFK agents)

## Queue discipline (mandatory)

1. **Only pick issues labeled `jules`.**
2. **Never pick** issues labeled `blocked` or `waiting-on-grammar`.
3. **Never start** an issue whose title starts with `[BLOCKED]`.
4. If the issue body starts with **STOP — DO NOT IMPLEMENT**, exit without coding.

## Current state

| Label | Meaning |
|-------|---------|
| `jules` + `ready-for-agent` | OK to implement |
| `blocked` | Dependencies not done — do not touch |
| `waiting-on-grammar` | Needs modernc/ccgo Beancount tree-sitter grammar |

Today, **only issue #1** should have `jules`.

## Hard product bans

- Do **not** invent a production Beancount parser while the modernc grammar is missing (see `SPEC.md` §5.3).
- Do **not** shell out to Python Beancount.
- Do **not** ignore `SPEC.md` layout (`contapila.cue`, `*/main.beancount`, `prices.beancount`).

## After finishing an issue

1. Open a PR against `master` with tests green.
2. Do **not** re-label the next issue yourself unless the human/process says so.
3. The human removes `blocked` and adds `jules` when the next slice is actually ready.

## Spec

Authoritative design: [`SPEC.md`](./SPEC.md).
