# 13. Milestones

Five phases. Each ends with exit criteria measured by the coverage report (14) and the quality report (09.4). Dates are relative to the start (week 0). Costs are estimates (07.6, 10.6).

## P0. Skeleton and registers (weeks 0 to 3)

- Create `tamnd/olymmath` and `tamnd/olymmath-agent` with the layouts of document 12; REUSE files; DCO.
- `om` with `tag`, `locator`, `store` (raw store on R2 `olymmath-raw`), `fetch`, `audit` (T, S, X families), `site build` for a few pages.
- `registers/competitions.yaml` from 01b (~210 competitions, editions and positions for the 30 biggest); `registers/sources.yaml` from 01a/01c; the alias table.
- Wave 0: load MathNet, project-numina/olympiads, AI-MO/aops (index only), Omni-MATH, OlympiadBench, the formal sets. Produce the first "what exists" table, with no canonical text yet.
- Send the P0 permission requests (11.4).

**Exit:** the register validates; wave 0 is in the raw store; the first coverage report lists the closed-universe positions (expected ~120k) and how many have a candidate.

## P1. Wave 1 and the first site (weeks 3 to 10)

- Adapters for imo-official, ISL, usa-tst-archive, MAA AMC, AoPS wiki (via the HF scrapes until permission), Kedlaya Putnam, problems.ru, Math SE, kvant.
- `normalize`, `match` (K and T signals), canonical selection, the review queue.
- Site v1 on Workers static assets + R2 Worker: competition tree, year grids, problem pages, D1 search, KaTeX, Asymptote. Mirror on GitHub Pages. Workers Paid ($5/month).
- Import formal statements of Compfiles, IMOSLLean4, Archive/Imo, PutnamBench (08.1).

**Exit:** IMO, ISL (to the embargo), USAMO, USAJMO, TST, AMC/AIME, Putnam: C0 (position has a problem) = 100% and C1 (canonical H1/H2) ≥ 98%; statement accuracy sample ≥ 99.5%; site live at olymmath.tamnd.com.

## P2. Nationals, journals, dedup at scale (weeks 10 to 24)

- Wave 2: national olympiads of the top 30 IMO countries from primary sources; Kalva; KöMaL; Crux; Mathematical Reflections; the OCR lanes (05.7).
- AoPS collections at full scale if the agreement exists; otherwise the HF-derived index with links.
- `match` with the E and A signals; the derived datasets (NuminaMath non-synthetic, Omni-MATH, OpenMathReasoning ids) matched in full, with match-rate reports.
- Topic and difficulty for every problem.

**Exit:** C0 ≥ 95% for tier-1 national olympiads; open-universe discovery rate from derived sets below 1% new problems per 10k rows (the sign that the human sources are covered); duplicate rate ≤ 0.5%; 150k+ problems.

## P3. AI solutions and grading (weeks 16 to 32, overlaps P2)

- Rubrics for P4 problems; the grader calibrated on IMO-GradingBench, OPC and ProofBench (κ ≥ 0.8 target).
- Pilot: 500 problems across difficulty bands to measure real cost; then P1-P3 sets of 07.2.
- The `/ai/` dashboard; the HF export (zone-split).
- Human grading of a sample; reviewers' badges.

**Exit:** every problem with no human solution has at least one graded AI solution or is listed as "open for AI"; the model to human grading agreement is published. Budget: ~$35k generation + about the same for grading (07.6), released in steps after the pilot.

## P4. Formalization and full coverage (week 32 onwards)

- Autoformalization of answer-type problems with the fidelity pipeline (08.4); round-trip checks; human review of `faithful`.
- Proof search with open provers on formal statements; imports of new group results (IMO 2026 and later).
- Wave 3 (metadata and link-only); remaining nationals; regional olympiads; book problems.
- Formula search (Q-08).

**Exit:** C0 ≥ 95% for all register competitions with any public source; 200k to 300k problems; formal statements for all IMO, ISL, USAMO and Putnam problems with fidelity records.

## Running costs (steady state)

| Item | Per month |
|---|---|
| Cloudflare Workers Paid (assets, Worker, D1, R2 ~15 GB site + raw store) | ~$5 to 15 |
| Build server | existing |
| AI solutions and grading for new problems (a few hundred a month) | ~$200 to 500 |
