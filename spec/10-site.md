# 10. The site: olymmath.tamnd.com and its mirror

Evidence: 01e Part 2. The binding facts:

- Cloudflare Pages and Workers static assets: at most **20,000 files** per deployment on Free and **100,000** on paid plans; 25 MiB per file. Cloudflare now recommends Workers static assets for new projects.
- GitHub Pages: **1 GB** per published site, 100 GB/month soft bandwidth, 10-minute deploy.
- R2: $0.015/GB-month, no egress fee. D1: 10 GB per database, FTS5 supported.
- olymmath will have 200k to 300k problems (planned capacity 400k, document 02.4), plus one page per solution, edition, topic and person, so one HTML file per page does not fit any single static deployment.

math2 and kvant use Pages Free with a 19,500-file guard. olymmath cannot. It needs Workers Paid ($5/month).

## 10.1 Pages and URLs

| URL | Page |
|---|---|
| `/` | home: search box, counts, competition map, latest problems, featured problem |
| `/c/` | all competitions, grouped by level (international, national, regional, university) and by country, with a world map |
| `/c/<comp>/` | one competition: description, organizer, editions as a year grid with coverage colours |
| `/<comp>/<edition>/` | one edition: all papers and problems, the official PDF links, statistics |
| `/<comp>/<edition>/<pos>` | an appearance; it renders the problem page (and is the canonical URL for the primary appearance) |
| `/p/<tag>` | the problem by tag; redirects to the canonical locator URL |
| `/p/<tag>/s/<tag>` | one solution, with its grade record |
| `/t/<topic>/` | a topic: problems, filters |
| `/people/<tag>` | an author or contributor: proposed problems, solutions |
| `/search?q=` | search (10.4) |
| `/random?…` | a random problem matching filters (practice mode) |
| `/sources/` | each source, its licence zone, counts, last harvest |
| `/coverage/` | the coverage report (document 14) |
| `/ai/` | the AI-solutions dashboard: model × competition grids, as on MathArena |
| `/about/`, `/spec/` | the project and this specification |

Each page has machine copies at the same path with `.md` and `.json`, and the site has `/llms.txt`, as in math2.

## 10.2 The problem page

```
IMO 2024 · Problem 1                        [N] Number theory · floor · Difficulty ●●●○○ (band 6)
Also: ISL 2024 N1
──────────────────────────────────────────────────────────────────────────────────────
Determine all real numbers α such that, for every positive integer n, the integer
    ⌊α⌋ + ⌊2α⌋ + ⋯ + ⌊nα⌋
is a multiple of n.
Source: official English text (imo-official.org, PDF p.1) · verified ✓ · Other languages: fr ru vi zh …
Answer ▸ (hidden, click to show)
──────────────────────────────────────────────────────────────────────────────────────
Solutions
 ▸ Official solution (IMO 2024 Shortlist booklet)                               official
 ▸ Solution by <MSE user> (CC BY-SA, full text) · graded 7/7 (human)           human
 ▸ Solution by <AoPS user> · excerpt + link ↗ (full text if the author opts in)  human
 ▸ 12 more community solutions on AoPS ↗ (link only)
 ▸ AI solutions (2), collapsed                                                  AI · model-graded 7/7
──────────────────────────────────────────────────────────────────────────────────────
Formal: Lean 4 statement (Compfiles Imo2024P1) · proof ✓ (Mathlib v4.35.0)
Related: variant of …; same idea as …; uses …
Discuss: AoPS thread ↗ · Edit this page on GitHub ↗
```

Rules:

- The answer and all solutions are hidden until the reader opens them (spoiler-safe by default; a setting turns this off).
- Every text block shows its origin, source and licence in a small line under it.
- Problems with `disputed` text show both versions and the difference.
- Keyboard navigation: `j`/`k` next and previous problem in the edition, `s` show solutions, `/` search.
- Light and dark themes; printable view of an edition as a contest paper (A4 PDF via the browser).

## 10.3 Browsing

The browse views are the core of "nice linking":

- **Competition tree**: level → country → competition → edition → paper → problem.
- **Year grid** for each competition: rows are years, columns are positions, cells coloured by status (has canonical H1/H2 text, has human solution, has AI solution, has formal proof).
- **Faceted list** (like the Codeforces problem set): filters for competition, country, year range, topic, difficulty band, type (proof/answer), "has human solution", "has AI solution", "human-verified", "has Lean statement"; sort by difficulty, year, popularity.
- **Graph links** on each problem: appearances, variants, same idea, uses, same author.
- **Collections**: curated lists in `registers/collections/` (for example "Classic inequalities", "ISL geometry 2000 to 2024"), each a page.

## 10.4 Search

- **Main site**: D1 with FTS5 behind a Worker. One row per statement and per solution heading. LaTeX is tokenized into words plus normalized formula tokens (`\frac{a}{b}` → `frac a b`) so that a formula query matches. The index for 400k problems and their solutions fits well inside one 10 GB database.
- **Exact lookups**: a locator or a tag typed in the box goes straight to the page.
- **Similar problems**: each problem page lists the 5 nearest problems by the embedding of 06.5, precomputed at build time.
- **Mirror**: Pagefind, sharded by competition family and merged in the browser (`mergeIndex`), because Pagefind is reported to fail above about 250k pages in one index.
- **Formula search** (phase 4): Tangent-CFT style vectors in Cloudflare Vectorize. Open question Q-08.

## 10.5 Rendering

- The site generator is part of `om` (`site` package), in Go, as in math2-agent and kvant-solver. It writes HTML from Go templates. No JavaScript framework is needed for content pages; small islands of JS for search, filters and spoiler toggles.
- **Math**: KaTeX renders at build time on the server (the `katex` package of math2-agent, which runs KaTeX in goja). Output is HTML+MathML. A formula that KaTeX cannot parse is logged by the build, and the page loads MathJax 4 for that formula only.
- **Figures**: Asymptote (`[asy]` from AoPS) is compiled to SVG at build time in a sandboxed container with a timeout, with `olympiad.asy` and `cse5.asy` vendored; TikZ via dvisvgm; scanned figures are cropped PNG/WebP. SVGs are cached by the hash of their source.
- Fonts and CSS are self-hosted. Pages work without JavaScript, except search.

## 10.6 Hosting: one renderer, three outlets

The generator renders **every** page to HTML. The outlets differ only in which files they take:

| Outlet | Files | Serves |
|---|---|---|
| **Workers static assets** (main, `olymmath.tamnd.com`) | the shell, CSS, fonts, JS, all competition/edition/topic index pages, and the problem pages of the tier-1 competitions, up to 95,000 files | free static requests, fastest path |
| **R2 + a small Worker** (same host) | every other page, as prebuilt HTML objects at their URL path | a request that misses the static assets goes to the Worker, which reads `r2://olymmath-site/<path>.html`, sets headers and caches at the edge. The Worker does no templating |
| **GitHub Pages** (mirror, `tamnd.github.io/olymmath`) | the tier-1 subset that fits under 900 MB, with Pagefind search | the mirror for people who cannot reach Cloudflare; it says which competitions it leaves out |

Tier 1 is decided by a rule in `registers/site-tiers.yaml` (initially: international olympiads, ISL, USAMO/USAJMO/TST, AMC/AIME, Putnam, and the national olympiads of the 30 strongest IMO countries). Moving a competition between tiers changes no URL.

Storage cost at full scale: about 1 million pages (problems, solutions, indexes) × 15 KB ≈ 15 GB in R2 ≈ $0.23/month, plus Workers Paid $5/month. Reads are well inside the included 10 million requests a month at the expected traffic.

## 10.7 Build and incremental deploy

- `om site build` writes `build/site/` and a manifest of each file and its content hash.
- `om publish cloudflare` compares the manifest with the last deploy. It uploads changed R2 objects only and deploys the Worker assets when tier-1 files changed. A full build of 1 million pages must finish in under 30 minutes on the build server; KaTeX output and figures are cached by hash, so a normal incremental build takes a few minutes.
- The build server runs `scripts/deploy-cloudflare.sh` every 5 minutes, like math2: it does nothing when the commits of `olymmath` and `olymmath-agent` are unchanged.
- The GitHub Actions workflow `pages` builds the mirror on each push to `main` and nightly.
- `om publish hf` pushes the weekly Parquet export to `tamnd/olymmath` on Hugging Face (statements, solutions, appearances, grades, formal records), split by licence zone (document 11).

## 10.8 API and MCP

`om serve` gives read methods over HTTP (and the Worker exposes the same JSON files statically): `GET /api/problem/<tag>`, `GET /api/edition/<comp>/<edition>`, `GET /api/search?q=`, `GET /api/random?filters`. The MCP server offers the same as tools (`get_problem`, `search_problems`, `list_edition`, `random_problem`, `get_solutions`), so that agents and tutors can use the corpus. Solutions are returned only when asked, so that a practice tool can hide them.
