# 07 — Design

Source: `design/DockDuck.html` (Claude Design export; open in a browser, 13 boards at 1440×900). Brief: dark primary + full light theme, Lucide icons (implemented via Iconify `@iconify-json/lucide`, all in `src/icons.ts`), Geist + mono font, minimal/Linear-Raycast feel, logs/terminal always dark.

| # | Board | Phase |
|---|---|---|
| 01 | Services + live logs | 4 / 5 |
| 02 | Compose editor | 6 |
| 03 | Environment editor | 6 |
| 04 | Logs viewer | 5 |
| 05 | Terminal (exec) | 7 |
| 06 | Docker contexts | 3 |
| 07 | Project roots | 3 |
| 08 | Registries | 8 |
| 09 | Settings | 9 |
| 10 | Dialogs and states | 3–4 |
| 11 | Sidebar | 3 |
| 12 | Project header | 3 |
| 13 | Status bar | 3 |

Board numbers follow the page order inside the file (verified against board titles).

Gaps vs the requirements, to design/extend when reached: global **Ports** view (feature 11) and the **app login** screen (feature 10a) are not in the brief's board list.

Implementation rule: extract colour/spacing/typography tokens into `apps/web/src/styles/tokens.css` once in Phase 3; components never hard-code colours.
