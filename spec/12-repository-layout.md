# 12. Repository layout and the `om` command tree

## 12.1 `tamnd/olymmath` (content)

```
olymmath/
├── README.md                 what the corpus is, the site, the counts, how to cite
├── CONTRIBUTING.md           how to fix a statement, add a solution, review a merge
├── LICENSES/                 CC-BY-SA-4.0, CC0-1.0, Apache-2.0 + REUSE.toml (per-path zones)
├── registers/
│   ├── competitions.yaml     the competition register (document 05.2): code, names, organizer, country, level, years, papers, positions
│   ├── competition-aliases.tsv   every spelling, AoPS category id and dataset label → code
│   ├── sources.yaml          the source register: adapter, licence zone, ToS note, rate limit, cursor
│   ├── topics.yaml           the topic tree (04.8)
│   ├── people.yaml           authors, reviewers, graders (opt-in)
│   ├── aliases.tsv           merged tags: old → new
│   ├── redirects.tsv         old locator → new locator
│   └── tags.reg              every tag ever minted, with the commit
├── problems/
│   ├── imo/2024/p1.md        a problem at its primary appearance (04.3)
│   ├── imo/2024/p1.fig1.svg
│   ├── isl/2024/n1.md        a stub: appearance → tag
│   ├── ru-allrussian/2019/g10-d2/p5.md
│   └── _loose/7K3/7K3QX2M9R2.md
├── solutions/
│   └── imo/2024/p1/4M2XKC9QRT.md
├── editions/
│   └── imo/2024.yaml         edition metadata: dates, host, papers, official PDF captures, cut-offs, statistics links
├── lean/                     the Lake package `Olymmath` (document 08)
│   ├── lakefile.toml
│   ├── lean-toolchain
│   └── Olymmath/Imo/Y2024/P1.lean
├── reviews/2026/11/<id>.yaml
├── runs/                     AI run summaries (no transcripts; transcripts are in R2)
├── reports/coverage/         the coverage report of each release (document 14)
├── macros/olymmath.sty       the LaTeX macro set (KaTeX subset)
└── releases/2027.01.yaml     release manifests: commit, counts, hashes of exports
```

Expected size at full scale: about 200,000 to 300,000 problem files (planned capacity 400,000) and 1 to 2 million solution files (document 02.4 has the estimate). Git holds this, but the clone is big. The README gives a sparse-checkout recipe for one competition, and the site build reads the tree with `git cat-file --batch` instead of a checkout.

If the solution count makes the repository too slow (> 3 GB packed), solutions of class H3 move to a second repository `tamnd/olymmath-solutions` with the same layout. This is open question Q-04.

## 12.2 `tamnd/olymmath-agent` (code)

```
olymmath-agent/
├── cmd/om/                   main
├── cli/                      the cobra tree
├── spec/                     this specification, 00 to 15
├── source/                   the registry and one package per adapter
│   ├── aops/  aopswiki/  imoofficial/  compendium/  problemsru/  komal/  ukmt/  maa/
│   ├── putnam/  hf/  github/  pdf/  stackexchange/  kvant/  ...
├── fetch/                    polite HTTP: robots, rate limit, retry, resume, user agent
├── raw/                      the content-addressed raw store (local + R2)
├── parse/                    shared parsers: BBCode (AoPS), MediaWiki, HTML, LaTeX, PDF text
├── ocr/                      PDF → Markdown lanes (native text, vision model) + textguard
├── locate/                   competition aliases → locators
├── match/                    MinHash LSH, embeddings, answer equality, decision rules
├── canon/                    the human-first rule; normalization
├── corpus/                   read and write the curated files; front matter schema
├── tag/                      tags (shared with math2)
├── review/                   the review queue and the terminal review tool
├── llm/  llm/claude/         one model interface; the Claude client; batch API
├── agent/  solve/  grade/    the solving and grading agents (document 07)
├── answer/                   answer forms and equality (math-verify container + Go fast path)
├── formal/  lean/            formal statements, proof checking, the pinned Lean runner (document 08)
├── audit/  coverage/         rules (09.3) and the coverage report (14)
├── site/  katex/  figures/  search/   the static site (document 10)
├── publish/                  cloudflare, ghpages, hf
├── api/  mcp/                read API and MCP server
├── bench/                    labelled sets for matching, OCR, grading
├── images/                   container images: lean, asymptote, math-verify, ocr
└── scripts/deploy-cloudflare.sh
```

## 12.3 The `om` command tree

| Command | Does |
|---|---|
| `om sources` | list the source register with state, counts and cursors |
| `om fetch <source> [--since] [--limit]` | harvest a source into the raw store |
| `om ingest <source>` | parse, locate, match, canon; write the curated layer; fill the review queue |
| `om reparse <source> --parser vN` | rebuild candidates from the raw store with a new parser |
| `om show <tag\|locator>` | one problem with all its records |
| `om find <text>` | search by text, formula or locator |
| `om review` | the terminal review queue |
| `om merge <tag> <tag>` / `om split <tag>` | manual identity changes, with a review record |
| `om solve --select <query> --budget <usd> [--model] [--batch]` | run the solver agents |
| `om grade --select <query> [--human]` | run the grader agents, or record a human grade |
| `om formalize --select <query>` | draft formal statements |
| `om lean check [--all\|<path>]` | check formal statements and proofs at the pinned toolchain |
| `om import formal <compfiles\|minif2f\|putnambench\|...>` | import a formal dataset and link it (08.5) |
| `om audit [--rule]` | run the audit rules |
| `om coverage` | write the coverage report |
| `om site build [--out]` / `om site serve` | build and preview the site |
| `om publish cloudflare\|ghpages\|hf` | deploy |
| `om serve [--mcp]` | the read API and the MCP server |

A command that is not written yet exits with code 3 and names its milestone, as in math2-agent.
