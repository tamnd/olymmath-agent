# 03. Architecture

## 3.1 Two repositories

olymmath follows the pattern of math2 and kvant: one repository for content, one for code.

| Repository | Holds | Licence |
|---|---|---|
| `tamnd/olymmath` | the curated layer: registers, problem files, solution files, figures, formal statements and proofs (`lean/`), reviews, coverage reports, release manifests | per-file licence zones (document 11); project-written text CC BY-SA 4.0; metadata CC0; Lean code Apache-2.0 |
| `tamnd/olymmath-agent` | the Go toolchain `om`: source adapters, fetcher, raw store, parser, matcher, solver and grader agents, Lean runner, audit, site builder, publisher, API and MCP server | Apache-2.0 |

Large data that does not belong in git:

| Store | Holds | Where |
|---|---|---|
| raw store | every capture, content-addressed | Cloudflare R2 bucket `olymmath-raw`; local cache `~/.cache/olymmath/raw` |
| work store | candidates, match index, embeddings, run transcripts | SQLite + Parquet files, on the build server; snapshots to R2 |
| releases | Parquet + JSONL exports of the curated layer | Hugging Face dataset `tamnd/olymmath`, GitHub release assets |

## 3.2 The boxes

```
                    ┌──────────────────────────── olymmath-agent (om) ───────────────────────────┐
 web sites ─┐       │                                                                            │
 forums  ───┤ fetch │  sources/*  ─▶ raw store ─▶ parse ─▶ locate ─▶ match ─▶ canon ─▶ curated  │
 PDFs    ───┤──────▶│  (adapters)    (R2, CAS)    (per      (aliases)  (K,T,E,A)  (06.3)   layer  │──▶ tamnd/olymmath (git)
 HF sets ───┤       │                              source)                 │                      │
 GitHub  ───┘       │                                                      ▼                      │
                    │                                               review queue ◀── people       │
                    │                                                                            │
                    │  solve (agents) ─▶ grade (agents + people) ─▶ solutions/grades            │
                    │  formalize ─▶ lean check (pinned toolchain) ─▶ formal/proof records       │
                    │  audit (rules) ─▶ coverage report                                          │
                    │  build ─▶ site + JSON + llms.txt + search shards ─▶ Cloudflare / GH Pages  │
                    │  serve: HTTP API + MCP                                                     │
                    └────────────────────────────────────────────────────────────────────────────┘
```

| Box | Package(s) in `olymmath-agent` | Document |
|---|---|---|
| Source registry and adapters | `source`, `source/<name>` | 05 |
| Fetcher (polite, resumable) | `fetch` | 05.5 |
| Raw store | `raw` | 04.5 |
| Parse, locate, match, canon | `parse`, `locate`, `match`, `canon` | 06 |
| Curated layer I/O | `corpus` (read and write problem and solution files) | 04 |
| AI solving and grading | `agent`, `solve`, `grade`, `llm`, `llm/claude` | 07 |
| Formalization | `formal`, `lean` | 08 |
| Audit and coverage | `audit`, `coverage` | 09, 14 |
| Site | `site`, `katex`, `figures`, `search` | 10 |
| Publish | `publish/cloudflare`, `publish/ghpages`, `publish/hf` | 10.7 |
| API and MCP | `api`, `mcp` | 10.8 |

## 3.3 Reuse of existing code

The tamnd organization already has many single-source CLIs built on `github.com/tamnd/any-cli/kit`. Each has a library package with an HTTP client and typed records. `om` imports them as libraries and does not reimplement their HTTP work. Each adapter in `source/<name>` is a thin layer: it calls the library, wraps each record as a capture and declares the parse function.

| Existing repo | Used for |
|---|---|
| `aops-cli` | AoPS wiki (MediaWiki API) and community forum topics and posts |
| `imo-cli`, `imo-shortlist-cli` | imo-official.org problems, results, shortlist booklets |
| `imomath-cli` | imomath.com archives |
| `putnam-cli`, `usamo-cli`, `mathcounts-cli` | Putnam archive, USAMO, MATHCOUNTS |
| `mathnet-cli`, `mathnet-ru-cli`, `kvant-mccme-cli`, `kvant` corpus | Russian sources; the kvant corpus problem section |
| `mathchina-cli`, `chinamath-cli` | Chinese competition sources |
| `mathse-cli`, `mathoverflow-cli` | Stack Exchange (CC BY-SA) |
| `mathreflect-cli` | Mathematical Reflections |
| `hf-cli` | Hugging Face hub metadata, dataset schemas, Parquet files |
| `arxiv-cli` | papers that release datasets or solutions |
| `kvant-solver` (`solve`, `rubric`, `answerguard`, `textguard`, `pricing`, `fleet`) | the solve-and-grade loop, the answer checker, the no-rewrite guard for OCR text, cost tracking |
| `math2-agent` (`tag`, `katex`, `lean`, `llm`, `site`) | tags, server-side KaTeX, the pinned Lean runner, the Claude client, the site builder patterns |

Where an existing CLI is only a scaffold (most are at the "page" example stage), the work to model its real records is done in that CLI repository first, then imported. That keeps one HTTP client per site in the whole organization.

## 3.4 The four flows

1. **Harvest** (daily for live sources, once for archives): `om fetch <source>` → raw store. Idempotent; a source has a cursor.
2. **Build the curated layer** (on each harvest, and on each parser change): `om ingest <source>` runs parse, locate, match and canon, writes problem and solution files, opens a pull request on `tamnd/olymmath` with the diff, and fills the review queue.
3. **Solve and grade** (budgeted batches): `om solve --select <query> --budget <usd>` and `om grade`. Results are solution files with `origin: ai` and grade records. Document 07.
4. **Publish** (on each push to `main` of either repository, and nightly): `om site build`, then `om publish cloudflare` and `om publish ghpages`, and weekly `om publish hf`.

## 3.5 Cross-cutting decisions

- **Go, one binary.** Same toolchain as math2-agent and kvant-solver (Go 1.27+). No Python in the main path. Python tools that have no Go equivalent (math-verify, some OCR engines) run as pinned containers behind a narrow interface.
- **Determinism.** Each stage records its version. The curated layer is a function of the raw store, the registers, the human reviews and the stage versions.
- **Humans review in git.** Each automatic change to the curated layer is a pull request. Reviews are files in `reviews/`, as in math2. A merge, a split, a canonical-text change of a problem with a human solution, and each `verified` status need a human review.
- **Models are tools, not sources.** A model can parse, translate, classify, solve and grade. A model never creates a problem and never edits a human statement beyond markup (06.3, 06.4).
- **Everything is labelled.** Each piece of text on the site shows its origin class, its source and its licence.
- **Polite harvesting.** Each adapter has a rate limit and identifies itself with a user agent that names the project and a contact address. robots.txt is respected. Document 11 has the legal policy.
