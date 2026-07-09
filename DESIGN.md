# Contapila web UI

## Theme
- daisyUI 5 + Tailwind CSS 4 (CDN in MVP single-binary server)
- Themes: `light` (default), `dark` (toggle + prefers-color-scheme via daisyUI)
- Restrained color strategy: base surfaces dominate; `primary` only for active nav / key total
- Semantic colors only: `base-*`, `primary`, `error`, `warning`, `success`, `info`

## Typography
- System UI stack (`font-sans` / daisyUI default)
- Fixed rem scale; tabular nums for money (`tabular-nums`)

## Layout
- App shell: top navbar + horizontal page menu under title
- Content max-width readable for tables; full width on xl for data
- Mobile: stack navbar, scrollable tabs
- Period filters (journal / P&amp;L): Fava-style `time` string — `2024`, `2024-03`, `month-1`, `2020 - 2024-06`

## Components (daisyUI)
navbar, menu, table, alert, badge, stats, theme-controller, breadcrumbs, button (default)

## Motion
None beyond 150ms hover/focus on interactive controls; respect prefers-reduced-motion (browser defaults)
