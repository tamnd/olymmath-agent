# 15. Open questions

Each question has an owner decision and a default that the spec uses until it is decided.

| ID | Question | Default until decided |
|---|---|---|
| Q-01 | **AoPS access.** Will AoPS agree to an export, an API key or an allowlisted crawler? On what terms (links back, no AI training)? | use the public HF-derived index (AI-MO/aops and wiki scrapes) as Z2 with links; no crawling past the Cloudflare challenge (05.6) |
| Q-02 | **The name.** "OlymMATH" is already the name of a RUC-AIBOX benchmark (2025, on HF and arXiv). Keep `olymmath`, or rename (e.g. "Olymmath Atlas")? | keep `olymmath` for the repos and domain; the site says it is not the RUC benchmark and links to it (which is itself a source in 05.4) |
| Q-03 | **Workers Paid.** The design needs 100k static files per deployment, so Workers Paid ($5/month). Accept, or stay on Free with ~19.5k static files and everything else from R2? | Workers Paid |
| Q-04 | **Solutions repo split.** 1 to 2 million solution files make `olymmath` heavy for git. Split into `olymmath-solutions` (or keep community solutions only in R2 + Parquet)? | one repo until 500k solution files, then split |
| Q-05 | **Tier-1 selection** for the static deployment and the mirror (10.6). | international + USA + Putnam + top 30 nationals |
| Q-06 | **KöMaL scope.** The A and B problems are olympiad level; C and K are school level. Include C? | A, B and the advanced (A/B-level) sections; C and K left out |
| Q-07 | **Gated datasets.** Some datasets (several Chinese competition sets, some benchmarks) are gated behind an access form. Accept their terms for matching only? | request access for matching only; never re-publish their rows |
| Q-08 | **Formula search.** Tangent-CFT vectors in Vectorize, or approach0 style tree search, or skip? | phase 4; decide after search logs show demand |
| Q-09 | **MCCME permission.** If problems.ru allows Z1, its 8k olympiad solutions become full text: the largest single gain in human solutions after AoPS. | Z2 until an answer |
| Q-10 | **ISL embargo.** Publish the ISL problems as soon as they are public (after the next IMO), automatically? | yes, with a scheduled job keyed to the IMO date |
| Q-11 | **Model-only grades on human solutions.** Show the model grade of a forum solution publicly, or only use it for ranking? | use for ranking; show only when a human grade agrees |
| Q-12 | **Lower rounds.** How deep into national rounds (regional, district) to go: they are many and rarely solved online. | include when a curated source holds them (problems.ru, KöMaL); otherwise register only |
| Q-13 | **Commercial use.** Some permissions will be "non-commercial". olymmath is non-commercial; does the HF export then need an NC licence for those parts? | such parts stay site-only (not in the export) |
| Q-14 | **Reviewer pool.** Who are the first reviewers, and what is the bar to become one? | the maintainer plus invited olympiad trainers; bar: 20 accepted reviews |
