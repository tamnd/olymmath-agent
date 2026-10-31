# 01. Research summary

Five research notes back this spec. Each marks its claims as primary [P], secondary [S] or UNVERIFIED. This page gives the numbers that drive the design. The details and links are in the notes.

| Note | Subject |
|---|---|
| 01a-research-datasets.md | 104 Hugging Face datasets plus Kaggle and GitHub sets, ranked into tiers |
| 01b-research-competitions.md | about 210 competitions, the archives that hold them, copyright, and gaps |
| 01c-research-web-sources.md | websites, forums and journals: access, robots, terms, OCR, legal |
| 01d-research-formal.md | Lean formalizations, formal benchmarks, statement audits |
| 01e-research-ai-and-site.md | AI solving and grading, costs, hosting, search, rendering |

## Datasets (01a)

- **The truly human, traceable core is small.** MathNet has 27,817 rows (47 countries, 143 competitions, taken from official booklets). project-numina/olympiads has 13,088 rows with exam, year and label. AI-MO/aops has 80,661 AoPS contest-collection problems with their c13 paths. Omni-MATH has 4,428. The AoPS wiki scrapes have about 43k rows.
- **The big sets are mostly derived or synthetic.** NuminaMath-1.5 has 896,215 rows, and 311,660 of them are synthetic. By `source`: cn_k12 268,819; olympiads 197,084; orca_math 151,934; synthetic_math 148,712; aops_forum 67,841; cn_contest 29,944. OpenMathReasoning, Nemotron and the RL mixes are re-cuts with problems rewritten by models.
- Estimate: **150 to 300k distinct olympiad-level problems on HF**, of which only 40 to 60k can be traced to an exam, a year and a problem number.
- `hendrycks/competition_math` was taken down (DMCA). Do not use it.

## Competitions (01b)

- About **210 competitions** (157 rows plus about 56 countries in groups), with organizers, first years and archives.
- The hubs are imo-official, AoPS (c13 and the wiki), Kalva, imomath, problems.ru, Evan Chen, the usa-tst-public archive, the IMO Compendium, KöMaL, Crux and MathNet.
- Estimate: **~120k problems in register competitions**, ~50k of them proof problems.
- **No open licences.** ARML is the most permissive. The ISL is embargoed for a year.

## Web sources (01c)

- **AoPS** is the largest source of human solutions. Every URL, including the wiki API, is behind a Cloudflare challenge. Its robots.txt says `ai-train=no`, and the wiki has no licence. Its AJAX API is known (`fetch_category_data`, `fetch_topic`, …) but is behind the same challenge.
- **problems.ru** has ~21k problems, 8,157 of them from olympiads. Crawl-delay is 20 s. Reuse requires a link and the problem number.
- **Math SE** `contest-math` has 10,806 questions under CC BY-SA, through the API. This is the cleanest full-text human-solution source.
- **Journals:** Crux, Mathematical Reflections, KöMaL and Kvant. imomath has a proof-of-work gate. cut-the-knot has a 120 s crawl-delay. Diễn đàn Toán học is behind Cloudflare.
- **OCR:** marker for born-digital PDFs; olmOCR and MinerU for scans.
- **Legal:** ETS v. Katzman (test questions are protected). This gives zones Z1/Z2/Z3 and a list of rights holders to ask.

## Formal (01d)

- Compfiles: 521 formalized, 515 solved, Lean v4.35.0, Apache-2.0. IMOSLLean4: 171 ISL problems. Mathlib Archive/Imo: 55. formal-imo: 258 statements. PutnamBench: 672.
- Larger sets: miniF2F v2, NuminaMath-LEAN (104k), FormalMATH (5,560), Lean Workbook (~140k), Goedel-Pset (1.7M). These are autoformalized and have many defects. An audit (arXiv 2606.29493) flagged 4,833 likely defects in 10,318 problems and confirmed 398. So fidelity has to be checked by people.
- IMO 2026 had Lean solutions from at least 5 groups.

## AI and site (01e)

- Frontier models now solve most olympiad problems (MathArena USAMO 2026: 95.2% for the top model). **Grading is the hard part.** Self-judging is worse than an independent jury. Use rubrics first and calibrate on IMO-GradingBench, OPC (5,062 proofs; human-human agreement 90.4%) and ProofBench. MathArena data is CC BY-NC-SA.
- Cost per problem with Batch (k = 4): about $0.06 on Haiku 5.5, $1.20 on Sonnet 5.5, $2.40 on Opus 5.5.
- **Hosting limits:** Cloudflare Pages / Workers static assets allow 20k files on Free and 100k on Paid. GitHub Pages allows 1 GB. Pagefind struggles above ~250k pages, while D1 FTS5 fits.
- The EU AI Act Art. 50 transparency duties apply from August 2026.

## What the research decided

1. **Statements come from human sources first.** Derived datasets are used to discover problems, not as text (06).
2. **AoPS is handled by permission and linking**, never by getting past its challenge (05.6, 11).
3. **Licence zones Z1/Z2/Z3** apply to every record (11.2).
4. **The jury grades AI solutions** and the score is the minimum. A human is needed for a "verified" badge (07).
5. **Formal statements are imported with ranks** and checked for fidelity by people (08).
6. **A hybrid static + R2 site** runs on Workers Paid (10.6).
