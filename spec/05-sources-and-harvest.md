# 05. Sources and the harvest

Evidence: 01a (datasets), 01b (competitions), 01c (web sources), 01d (formal). This document turns them into the source register, the adapters, the harvest waves and the rules for each hard source.

## 5.1 Three kinds of source

| Kind | What it gives | Examples | Role |
|---|---|---|---|
| **primary** | the organizer's own text (H1) | imo-official.org, MAA booklets, UKMT, CEMC, mathematik-olympiaden.de, USA TST archive, national olympiad sites | the canonical statement; official solutions |
| **curated** | a human collection that cites the primary source (H2) | problems.ru, AoPS contest collections and wiki, Kedlaya's Putnam archive, Kalva, Evan Chen's notes, imomath, MathNet, project-numina/olympiads, Kvant, Crux, KöMaL | the index of "what exists"; statements when no primary text is online; human solutions |
| **derived** | rows copied from the above, sometimes changed (D) or written by a model (M) | NuminaMath-1.5, Omni-MATH, OlympiadBench, OpenMathReasoning, AoPS-Instruct, the RL mixes | discovery of problems we have not found yet; xrefs; their M parts become AI solutions or nothing |

The harvest goes in this order of value: primary and curated first, so that derived rows usually match an existing problem and only add an xref.

## 5.2 The competition register

The register (`registers/competitions.yaml`, 14.1) starts from the catalogue in 01b: about 210 competitions in groups. A census of the competitions of each region adds the junior and elementary contests of 02.2, so the register grows to many hundreds of rows. The group says where a competition is held (international, national, …). The level says who it is for.

| Group | Count | Examples (code) |
|---|---|---|
| international, high school | ~45 | `imo`, `isl`, `egmo`, `apmo`, `balkan`, `jbmo`, `baltic-way`, `memo`, `ibero`, `centro`, `tot`, `sharygin`, `zhautykov`, `silk-road`, `rmm`, `igo`, `kurschak`, `cono-sur`, `pamo`, `elmo`, `usemo`, `tuymaada`, `iom`, `benelux`, `nordic`, `gulf`, `arab`, `caucasus`, `lusophon`, `francophone` |
| undergraduate | ~10 | `putnam`, `imc`, `seemous`, `schweitzer`, `vtrmc`, `ciim` |
| USA | ~20 | `amc8`, `amc10`, `amc12`, `ahsme`, `aime`, `usamo`, `usajmo`, `tstst`, `us-tst`, `us-egmo-tst`, `hmmt`, `pumac`, `bmt`, `cmimc`, `smt`, `arml` |
| national olympiads + TSTs | ~135 | `ru-allrussian`, `ru-moscow`, `ru-spb`, `cn-mo`, `cn-tst`, `cn-wmo`, `cn-gmo`, `cn-league`, `vn-vmo`, `vn-tst`, `ro-nmo`, `ro-tst`, `ir-mo`, `ir-tst`, `in-inmo`, `in-rmo`, `jp-jmo`, `kr-kmo`, `tw-tst`, `uk-bmo1`, `uk-bmo2`, `ca-cmo`, `de-bundeswettbewerb`, `de-mo`, `hu-oktv`, `pl-om`, `bg-mo`, `tr-mo`, `br-obm`, `fr-tst`, … |
| journals (source kind, not competition) | ~10 | `journal-crux`, `journal-komal`, `journal-kvant`, `journal-mr`, `journal-excalibur`, `journal-amm`, `journal-gazeta`, `journal-tzaloa`, `journal-eureka`, `journal-parabola` |

Each row has the slug, the names in each language, the group, the level, first year, gaps, the positions per paper, the official URL, whether official solutions are published, the copyright note, and the sources that hold it. Lower rounds become their own competition codes when a source holds them (`ru-allrussian-regional`, `de-mo-r3`).

**Levels.** The level of a paper is the school stage of the students it is for. The values, lowest first, are:

| Level | Students | Examples |
|---|---|---|
| `elementary` | primary school, about grades 1 to 5 (ages 6 to 11) | Kangaroo Écolier, MOEMS Division E, the primary rounds of national olympiads |
| `junior` | lower secondary or middle school, about grades 6 to 9 (ages 11 to 15) | AMC 8, MATHCOUNTS, JBMO, USAJMO, Euler Olympiad |
| `high-school` | upper secondary, about grades 10 to 12 (ages 15 to 18), and contests for all secondary students | IMO, AMC 12, All-Russian MO, most national olympiads |
| `undergraduate` | university students | Putnam, IMC, SEEMOUS |
| `open` | anyone, or no one school stage | the problem sections of journals, problem sets |

The grades are a guide. The school system of the country decides: a paper for the last year of lower secondary is `junior`, also when that year is called grade 9 in one country and grade 8 in another. A paper for the grades of two levels has the higher level, so the AMC 10 (grade 10 and below) is `high-school`. A junior olympiad such as the JBMO is `junior`, whatever its difficulty: the level is about the students, not about how hard the problems are.

A competition whose papers are all at one level has `level` only. A competition with papers at several levels says so in three keys:

- `level` is the highest level of its papers. It is required.
- `levels` lists every level of its papers, lowest first, and ends with `level`. It is there only when the papers are at two or more levels.
- A paper of `layout` (or of the `layout` of an era) can have `level`, when the paper is at a lower level than the competition. It must be one of `levels`. A paper with no `level` is at the level of the competition.

```yaml
- code: tuymaada
  level: high-school                # the highest level
  levels: [junior, high-school]     # all the levels, lowest first
  layout:
    - {paper: junior, name: Junior, level: junior, days: 2, positions: [j1, j2, j3, j4, j5, j6, j7, j8]}
    - {paper: senior, name: Senior, days: 2, positions: [s1, s2, s3, s4, s5, s6, s7, s8]}   # high-school, the level of the competition
- code: kangaroo                    # no layout yet: levels alone says that it has papers at three levels
  level: high-school
  levels: [elementary, junior, high-school]
- code: amc8
  level: junior                     # one level
```

This follows 04.1: a level or a grade is a paper of an edition, not a competition of its own. The per-paper level is optional, so a row of the census can give `levels` before anyone writes the layout. `om check` reports a competition with no level (a warning) and a level, a `levels` list or a paper level that breaks these rules (an error). The coverage report counts each expected position at the level of its paper (14.3).

The positions of an edition come from `layout` (papers with positions) or from `per_year` (p1 to pN). When the format changed over the years, `eras` gives the format of a range of years, and `overrides` gives the positions of one year. The top-level `layout` and `per_year` are the default for the years that no era and no override covers. An era with no `to` did not end. An era with no `layout` and no `per_year` says that the format of its years is not known: the import accepts any valid position, and the coverage report expects no position. Eras must not overlap.

```yaml
- code: isl
  eras:
    - {from: 1959, to: 2005}
  layout:
    - {paper: shortlist, name: Shortlist, positions: [a1, a2, c1, c2, g1, g2, n1, n2]}
  overrides:
    1970: [p1, p2, p3]
```

The coverage report, the import (`om import`, `om coverage`) and `om edit` use the format of the year of each record. Grades and rounds that share the numbers of their positions, for example grade 9 and grade 10 of `ru-allrussian`, need the paper part of the locator (`ru-allrussian/2019/g10-d2/p3`). The records and the import do not use the paper part yet, so their editions have one flat list of positions, and the register gives them no layout.

## 5.3 The source register

`registers/sources.yaml` has one entry per source:

```yaml
- id: problems-ru
  kind: curated
  url: https://problems.ru/
  adapter: problemsru@v1
  library: github.com/tamnd/mathnet-ru-cli   # or a new problemsru-cli
  access: html
  robots: {crawl_delay: 20}
  terms: "free for non-commercial use; other sites must link to problems.ru and the problem number"
  zone: {statement: Z2, solution: Z2}       # Z1 if MCCME grants permission (11.4)
  ai_train: allowed-unknown
  size: {problems: 21000, olympiad: 8157}
  wave: 1
  state: planned
```

## 5.4 The sources, by wave

Sizes are from 01a to 01d; "est." marks an estimate. The zone is the default of document 11.

### Wave 0: the seed index (already public data, no crawling)

These give the list of problems and their provenance labels in days, before any crawler runs. Their text is evidence; the canonical text comes from waves 1 to 3.

| Source | Size | Use | Zone of its text |
|---|---|---|---|
| `hf:MathNet` (MIT; 47 countries, 143 competitions) | 27,817 HF rows; 30,676 in paper | official-booklet statements with country, competition, id; the best traceable set | per item (competition copyright); paper CC BY 4.0 |
| `hf:project-numina/olympiads`, `olympiads-ref` | 13,088 | 28 exams with `exam`, `year`, `problem_label`; official solutions | Z2 |
| `hf:AI-MO/aops` | 80,661 | AoPS contest collection threads with `metadata.path` (the c13 tree), the de facto index of AoPS | Z2, `ai-train=no` |
| `hf:NuminaMath-1.5` non-synthetic rows | ~316k rows before dedup | discovery via `source` (`olympiads` 197,084; `aops_forum` 67,841; `cn_contest` 29,944; `inequalities` 7,314; `amc_aime` 5,872; `number_theory` 4,043; `olympiads_ref` 3,638) | D; solutions D |
| `hf:Omni-MATH` | 4,428 | ~100 competition labels, difficulty, domain | D |
| `hf:OlympiadBench` (math) | 6,142 (≈2.1k competition) | competition subset; the rest is gaokao (negative filter) | D |
| AoPS wiki scrapes on HF | ~43k rows with links | AMC/AIME/USAMO statements and wiki solutions with page links | Z2 |
| `hf:nvidia/OpenMathReasoning` | ~125k from `aops_c6_high_school_olympiads` | discovery only; the problems were rewritten by a model (M text), so only the AoPS topic ids are used | M |
| `hf:nvidia/Nemotron-Math-Proofs-v1`, `Nemotron-Math-v2` | 243k / 7.1M | `url` and `user_name` point to AoPS posts: discovery of forum problems | M |
| `hf:DeepStudentLlama/AoPS-Instruct`, LiveAoPSBench | ~650k QA | `original_question` is human AoPS text; dates for contamination | D |
| small human sets: MathArena (~900), IMO-Bench, AIME 1983 to 2026 (1,065), Putnam sets, HMMT, BRUMO, SMT, CMIMC, OlymMATH (RUC), AMO-Bench, Nemotron-IMO-Bench, HARP, CHAMP, PolyMath, MathOdyssey, OlympicArena | hundreds to thousands each | answers, difficulty, dates; links to sources | per set |
| regional: TÜBİTAK (tr), Indonesian olympiad set, HRM8K KMO (ko) | 2 to 8k | national problems not elsewhere | per set |
| formal: Compfiles, IMOSLLean4, Archive/Imo, formal-imo, PutnamBench, miniF2F v2 | 55 to 672 each | locators come free with the native names; document 08 | Apache/MIT |

Wave 0 also builds the alias table: every `source`, `exam`, `competition`, AoPS path and dataset label maps to a register code.

### Wave 1: primary and core curated sources

| Source | Size | Access | Notes |
|---|---|---|---|
| `imo-official` | IMO 1959 to 2026 (~400 problems); ISL after the embargo | PDF per year and language | official texts in many languages; statistics per problem (difficulty) |
| `usa-tst-archive` (GitHub `usa-tst-public`) | TST, TSTST, EGMO TST | git | problems, solutions, statistics |
| `maa-amc` | AMC 8/10/12, AHSME, AIME, USAMO, USAJMO (~6k) | PDFs | MAA permits non-commercial web posting with the notice; solutions on the AoPS wiki |
| `aops-wiki` | ~12.7k articles; every AMC/AIME/USAMO problem with wiki solutions | MediaWiki API, behind a Cloudflare challenge (5.6) | Z2; MAA notice kept |
| `kedlaya-putnam` | Putnam 1985- with solutions; older via Gleason/Alexanderson books (metadata) | HTML/PDF | Kedlaya asks for links, not copies: statements Z2, solutions link |
| `problems-ru` | ~21k, 8,157 olympiad (Moscow MO 2,003, ToT 1,854, All-Russian 1,125, Sharygin 915, …) | HTML, TeX inside; crawl-delay 20 s (~5 days) | statements and solutions with source references; ask MCCME (11.4) |
| `evanchen` | IMO 1997 to 2026, USAMO 1996-, JMO, TST, TSTST, EGMO, ELMO, USEMO; 318 files with TeX | TeX | ask first; until then Z2 (statement + link) |
| `mathnet-official` | the official booklets that MathNet used | PDF | to raise MathNet rows to H1 |
| national official sites with archives (01b §4: UKMT/BMO, CEMC, CMS, AMT, mathematik-olympiaden.de, Bundeswettbewerb, olimpiada.ru/VMO, turgor.ru (ToT), geometry.ru (Sharygin), OBM, OMM, imojp, KMS, HKMO, SMO, …) | ~60 countries with a confirmed primary source | PDF/HTML | respect robots (CMS disallows the CMO problem sets) |
| `kvant` | Kvant Задачник (~3k) | the kvant corpus (git) | ids from kvant; no new harvest |
| `mathse` | 10,806 `contest-math` questions | Stack Exchange API with a key | Z1 (CC BY-SA); the cleanest source of full human solutions |

### Wave 2: the long tail of curated sources and journals

| Source | Size | Access | Notes |
|---|---|---|---|
| `aops-collections` (full c13 tree + threads) | ~80k contest problems, their reply threads (est. 1M+ posts) | AJAX JSON behind a Cloudflare challenge (5.6) | the biggest source of human solutions; Z2 |
| `imomath` | ~8.5k problems, 42 countries | HTML behind a JS proof-of-work gate | ask the owner (5.6) |
| `kalva` (prase.cz mirror) | ~4,100 olympiad problems, ~2,300 with solutions | static HTML | old IMO/USAMO/Putnam/APMO second sources; Z2 |
| `komal` | 20 to 30k (A, B olympiad level; C, K school level; all in scope) | HTML (current) + scans 1893 to 2001 | English statements for 40+ years; Hungarian solutions |
| `crux` | ~5k proposals + Olympiad Corner, 1975 to 2026 | open PDFs; old volumes are scans | OCR; personal use only → ask CMS |
| `mathreflections` | ~3k, 2006 to 2026 | born-digital PDFs | ask AwesomeMath |
| `excalibur`, `parabola`, `pme`, `ssmrmh`, `gazeta`, `tzaloa`, `eureka` | hundreds to thousands each | PDF | OCR where scanned |
| `cut-the-knot` | olympiad inequalities and geometry | HTML, crawl-delay 120 s (~14 days) | Z2/Z3 |
| `imogeometry-blog` | the map of which years exist for each olympiad | HTML | used for the register (positions and years), not for text |
| regional forums: `diendantoanhoc` (vi, 638k posts), `mathchina`, AoPS language forums | large | HTML behind Cloudflare | Z3 until permission; discovery of Vietnamese and Chinese problems |

### Wave 3: metadata and link-only

AMM, Mathematics Magazine, College Mathematics Journal, Math Horizons (paywalled, Z3: locator, title, link, and the problem number); 中等数学; THTT; zbMATH Open (CC BY-SA metadata for problem books, MSC 97U40 and 00A07); Wikipedia list pages; Brilliant; Reddit r/mathriddles; problem books (metadata and links only).

### Not used as problem sources

Synthetic and model-written sets (NuminaMath `synthetic_math`, `synthetic_amc`, `orca_math`, `metamath`; DeepTheorem variants; olympiad-style synthetic generators; AstralMath; PolyUniMath) and K-12 sets (`cn_k12`, gaokao). The RL mixes (OpenR1, Skywork-OR1, DAPO, DeepScaleR, STILL, Polaris, AceReason, Big-Math) are re-cuts of the above. They are read once to confirm they add no new problem (a match rate report), then dropped. `hendrycks/competition_math` is not used (DMCA takedown); MATH problems are AMC/AIME problems and come from their primary sources.

## 5.5 The fetcher

`fetch` is shared by all adapters:

- **Identity.** User agent `olymmath-bot/<version> (+https://olymmath.tamnd.com/about/bot; <contact>)`. One identity, never rotated, never disguised as a browser.
- **robots.txt.** Read and obeyed, including `Crawl-delay`. Content signals (`ai-train=no`) are stored on the source and enforced at export (09.3 X02).
- **Rate.** Per host, the lower of the robots delay and the register limit; default 1 request per 2 seconds; exponential back-off on 429 and 5xx; stop the source on repeated 403.
- **Resume.** Each source has a cursor in the work store. A run can stop at any point and resume.
- **Conditional requests.** ETag and Last-Modified, so a nightly refresh of a live source costs little.
- **Captures.** Every response body goes to the raw store with its hash (04.5).

## 5.6 Hard sources: Cloudflare challenges, bot gates and permission

Several of the most valuable sites (AoPS community and wiki, imomath, Diễn đàn Toán học, ProofWiki, Stack Exchange HTML) answer automated clients with a bot challenge. **olymmath does not circumvent bot challenges or proof-of-work gates** (no headless browser solving challenges, no TLS fingerprint impersonation, no puzzle solvers). A challenge is the site saying no to automated access, and AoPS's terms reserve the right to limit access. The plan is:

1. **Ask.** Write to AoPS, the imomath owner and VMF for a data agreement: an export, an API key or an allowlisted crawler identity, for statements and the thread index, with links back. Offer what olymmath gives back: links to every thread, attribution, the dedup map, and the coverage report (open question Q-01).
2. **Meanwhile, use what is already public.** `AI-MO/aops` (80,661 contest-collection problems with their c13 paths and topic ids), the AoPS-wiki scrapes on HF, AoPS-Instruct, and the Nemotron AoPS urls give the full AoPS index and the statements, under the dataset licences and the AoPS terms. Their text is Z2 (statement + link) and is never put in the Hugging Face export (`ai-train=no`).
3. **Link to every thread.** For each problem with an AoPS topic id, the site links to the thread. The thread is the "discussion and community solutions" of the problem. This is the use AoPS's own content signal allows (`use=reference`).
4. **Contributors opt in.** An AoPS author can claim their solutions and license them CC BY-SA on olymmath (a form that records the AoPS user name and a verification post). Those become Z1.

For sites without a challenge, the adapter crawls normally under 5.5.

## 5.7 PDFs and OCR

Many national olympiads, all journals and the official IMO texts are PDF. The `ocr` package reads a PDF in one of these lanes:

| Lane | When | Tool |
|---|---|---|
| native | born-digital PDF with a text layer | `pdftotext` + layout, then `marker` for math |
| TeX | the source TeX exists (Evan Chen, USA TST archive, some booklets) | use the TeX; skip OCR |
| vision | scans | `olmOCR` or `MinerU`, then a vision-model pass that compares the LaTeX rendering with the page image |

`om ocr -lanes native` implements the native lane with no model. It reads the text layer of each page with `pdftotext`, cleans the layout (ligatures, soft hyphens, runs of spaces), and puts the math of the text layer into TeX by fixed rules (`ocr.TeX`): Unicode superscripts and subscripts, vulgar fractions, Greek letters, the math alphabets, roots and the math symbols become TeX between dollars, and prose stays as it is. Its extraction is `native/pdftotext@<version> (om-tex/1)`. A page is flagged, so its problems stay below the confidence threshold, when its fonts have no Unicode map (`encoding`: private use characters or U+FFFD) or when its words have two heights (`layout`: an exponent or an index that the text layer writes on the line, as `A−1` for $A^{-1}$). Such pages go to the vision lane.

The TeX lane has no OCR: the adapter reads the source TeX and puts it into Markdown (`archive.Markdown`, used for the Putnam and USA TST archives).

`om ocr` with the other lanes implements the vision lane. It renders each page to an image and asks one of two kinds of reader for Markdown with TeX math:

| Reader | `-lanes` | Tool |
|---|---|---|
| model clients | `codex,gemini,claude` (the default) | the command-line client of each model, with its own login. A client whose allowance is spent rests until the time that it gives |
| local servers | `local` | vision servers of the user through the OpenAI chat-completions API (package `vision`): GLM-OCR under MLX on this Mac, and olmOCR 2 under vLLM on the GPU host gpc through `ssh` and `curl`. Targets are `MODEL=URL` in `-local` or `$OLYMMATH_LOCAL`. The targets of one model share the pages in turns, a 429 rests a target, a target that does not answer rests for 5 minutes, and a health check asks each target for one token. The cache keeps the answers of each model apart |

Every OCR text records `extraction: <lane>/<tool>@<version>`, a confidence, and the page image crop. It goes through the no-rewrite guard of 06.4 step 6. OCR statements below the confidence threshold go to review before they can be canonical. A text of the native lane whose formulas a model put into TeX from the page image records its extraction in the same way, such as `native/pdftotext@26.10.0 (claude-opus-5-5)`.

## 5.8 Adapters and libraries

Each adapter imports the library of an existing CLI (03.3) or a new one. New CLI repositories are created when no library exists, following the any-cli kit pattern: `problemsru-cli`, `komal-cli`, `crux-cli`, `kalva-cli`, `ukmt-cli`, `maa-amc-cli`, `usatst-cli`, `evanchen-cli`. The adapter itself only maps library records to captures and candidates. Adding a source is: a register row, an adapter of about 200 lines, an alias table update, and a fixture test with 20 real captures.
