# The olymmath specification

**All olympiad mathematics problems and their solutions in one place.** Human-written text comes first. AI solutions are labelled and graded. Formal Lean statements are added later. Everything is published at https://olymmath.tamnd.com (Cloudflare) and mirrored on GitHub Pages.

- Repositories: `tamnd/olymmath` (content) and `tamnd/olymmath-agent` (the Go toolchain, binary `om`). This follows the pattern of math2 and kvant.
- Date: 2026-10-31. Status: accepted for M1. Later milestones can change it.

## Documents

| # | Document | What it settles |
|---|---|---|
| 00 | README | this index |
| 01 | research | the summary of the research and the numbers behind the design |
| 01a to 01e | research notes | datasets, competitions, web sources, formal sources, AI and site |
| 02 | the goal | scope, quality promise, scale (200 to 300k problems, capacity 400k) |
| 03 | architecture | two repos, the stores, packages, the four flows |
| 04 | object model | record kinds, tags, locators, files |
| 05 | sources and harvest | competition and source registers, waves 0 to 3, polite fetching, AoPS stance, OCR |
| 06 | canonicalization and dedup | origin classes, the human-first rule, matching signals |
| 07 | solutions and agents | solution order, the solver, jury grading, labelling, cost |
| 08 | formalization | Lean imports, the fidelity pipeline, proof checks |
| 09 | quality and audit | statuses, badges, audit rules, measured quality |
| 10 | site | pages, browsing, search, rendering, hosting, deploy, API |
| 11 | legal and licensing | zones Z1/Z2/Z3, own licences, permissions, ai-train, takedown, AI labels |
| 12 | repository layout | trees of both repos, the `om` commands |
| 13 | milestones | phases P0-P4 with exit criteria and costs |
| 14 | coverage | how "all problems" is measured: closed universe, C0-C8 |
| 15 | open questions | decisions still to make, with defaults |

Reading order: start with 02, 03 and 04. Then read 05 and 06 (how problems come in), 07 and 08 (solutions and formal statements), 09 and 14 (how quality and completeness are proved), 10 and 11 (publishing), and 12 and 13 (building it).

## Settled decisions

1. A problem exists once. Each time it appears (IMO P1, ISL N1, a TST) is an appearance of that one problem.
2. The canonical statement is never model-written. The order of preference is official original, then official translation, then curated, then community, then derived.
3. Every record has a source, a capture in the raw store, and a licence zone.
4. No bot challenge or proof-of-work gate is circumvented. AoPS is handled by asking for permission and by linking.
5. AI solutions are shown after human solutions and labelled. A jury of models from more than one family grades them, and the score is the minimum. A "verified" badge needs a human.
6. "All problems" is measured against a register of competitions, editions and positions, so a missing problem shows up as an empty cell.
7. Lean v4.35.0 and Mathlib v4.35.0 are pinned, as in math2.
8. One Go renderer serves three outlets: Workers static assets, R2 behind a Worker, and the GitHub Pages mirror.

## Conventions

- Section references look like 05.6 (document 05, section 6).
- Tags are 10 Crockford base32 characters with a Luhn mod 32 check. Locators look like `imo/2024/p1`.
- Numbers from research carry the marker of their note. A number marked "estimate" is to be measured in P0-P1.
