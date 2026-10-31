# 01a. Research: datasets on Hugging Face and similar hubs

Date of survey: 2026-10-31. Scope: datasets of informal competition and olympiad math on Hugging Face (HF), plus Kaggle and GitHub. Formal Lean sets appear only when they also carry informal text.

## 0. Method and conventions

- Metadata comes from the HF Hub API, `https://huggingface.co/api/datasets/<id>`. That gives the creation date, licence tag, language tags and gating.
- Row counts and column names come from the datasets-server, `https://datasets-server.huggingface.co/info?dataset=<id>`.
- Value counts come from `https://datasets-server.huggingface.co/statistics?dataset=<id>&config=<c>&split=<s>`.
- Dataset cards come from `https://huggingface.co/datasets/<id>/raw/main/README.md`.
- No downloaded content was executed. Only JSON and README text were read, with curl and jq.
- [P] marks a primary source: the HF API, the datasets-server, the card written by the dataset owner, or the owner's own GitHub. [S] marks a secondary source: a paper summary, a blog or a mirror. UNVERIFIED means I could not confirm the fact through the API or the card.
- "Rows" is the sum over all splits and configs, from the datasets-server [P], unless a note says otherwise.
- "Distinct human problems" is my own estimate. It counts problem statements that a human competition author wrote. It excludes paraphrases, translations and synthetic items. Treat it as a planning number, not a measurement.
- The dataset URL is always `https://huggingface.co/datasets/<id>`. The tables give the id only, to save space. Every id was resolved through `https://huggingface.co/api/datasets/<id>` [P].
- Licence "none" means the Hub has no `license:` tag and the card states no licence [P].
- Dates are the Hub `createdAt` values [P].

## 1. Official and curated olympiad collections (core sources)

| # | HF id | Publisher, created | Rows | Est. distinct human problems | Origin; who wrote the solutions | Lang | Key columns | Licence | Quality notes; overlap |
|---|---|---|---|---|---|---|---|---|---|
| 1 | `ShadenA/MathNet` | ShadenA (MathNet team), 2026-04-23 | 27,817 in config `all` (55,634 across all configs: `all` plus per-country configs, which repeat the rows) [P] | ~27.8K | Official national and regional olympiad booklets. Solutions are the booklets' own, written by humans [P card] | 17 tags: en, pt, es, fr, it, sr, sl, de, zh, ro, ko, nl, ru, mn, mk, pl, hu [P] | `id, country, competition, language, problem_markdown, solutions_markdown, final_answer, topics_flat, images, problem_type` | cc-by-4.0 [P] | Best single human source. It has a stable `id` and per-country configs (e.g. Brazil 1,281, Baltic_Way 853, Argentina 532) [P]. It overlaps the Numina olympiads set and AoPS for IMO, Balkan and Baltic. |
| 2 | `ShadenA/MathNet-Retrieve` | ShadenA, 2026-06-16 | 441,264 [P] | 0 new | Retrieval pairs (query, corpus, score) built on MathNet [P] | multilingual | `_id, corpus-id, query-id, score, text` | cc-by-4.0 | Useful as a duplicate and near-duplicate signal, not as new problems. |
| 3 | `project-numina/olympiads` | Project Numina, 2024-11-21 | 13,088 [P] | ~13K | PDFs of 28 official exams, parsed. Official human solutions [P card] | en (some translated) | `exam, year, problem_label, problem, solution, problem_type, tier, metadata` | none [P] | Exam counts [P]: HMMT 4,996, IMO-SL 1,288, IMO 1,101, French_envois 677, BalticWay 673, AllSovietUnion 557, Dutch_TST 417, French_tests 351, JBMO-SL 348, USAMO 324, EGMO 247, APMO 207. Has an exam, year and label key. The old id `AI-MO/olympiads-0.1` redirects here [P]. |
| 4 | `project-numina/olympiads-ref-base` | Project Numina, 2024-12-20 | 13,087 [P] | same 13K | The same exams, with raw and cleaned text plus validity flags [P] | en | `problem_raw, solution_raw, problem, solution, answer, problem_is_valid, solution_is_valid, problem_type, question_type, metadata` | none | 1:1 with #3. Use it for the raw vs cleaned pair. In NuminaMath-1.5 it appears as `olympiads_ref`, 3,638 rows [P]. |
| 5 | `beichen0426/olympiads-ref` | beichen0426, 2026-01-04 | no info (viewer off) [P] | 0 new | Mirror of #4. UNVERIFIED | ? | ? | none | Mirror. Skip it. |
| 6 | `project-numina/aops` | Project Numina, 2024-01-07 (old id `AI-MO/aops`) | 80,661 [P] | ~60 to 80K (all levels) | AoPS forum threads. Solutions are human forum posts (`candidates`) [P] | en | `problem, solution, candidates, metadata` (`metadata.path` = forum category), `tags` | none (AoPS terms apply) | Largest raw AoPS dump on HF with a category path. Overlaps OpenMathReasoning, AoPS-Instruct, Omni-MATH and Numina `aops_forum`. |
| 7 | `KbsdJames/Omni-MATH` | Gao et al. (PKU), 2024-09-14 | 4,428 [P] | ~4.4K | Problems from AoPS contest pages and official sites. Human solutions [P card; S paper arXiv 2410.07985] | en | `domain, difficulty, problem, solution, answer, source` | apache-2.0 [P] | `source` names the competition (e.g. putnam 101, imc 68, apmo 61, china_national_olympiad 38, usa_team_selection_test 35) [P]. Difficulty runs 1 to 10. |
| 8 | `martheballon/Omni-MATH-2` | martheballon, 2026-01-26 | 4,428 [P] | same | Omni-MATH with an `id` and `tags`. Some fixes claimed. UNVERIFIED | en | adds `id, tags` | apache-2.0 | Prefer it over #7 only if the fixes check out. |
| 9 | `Hothan/OlympiadBench` | He et al. (OpenBMB/THU), 2024-07-16 | 8,476 total; 6,142 math across 12 configs [P] | ~2.1K COMP + ~4.0K CEE (gaokao) | Olympiad and gaokao problems. Expert solutions [P; GitHub https://github.com/OpenBMB/OlympiadBench, MIT] | zh, en | `question, solution, final_answer, context, image_1..9, subfield, difficulty, modality, answer_type` | apache-2.0 on HF; MIT on GitHub [P] | Math COMP is only ~2.1K. CEE is gaokao, which is not olympiad. Includes multimodal items. |
| 10 | `math-ai/olympiadbench` | math-ai, 2025-02-13 | 674 [P] | subset | OE_TO_maths_en_COMP only [P] | en | as #9 | none | Mirror subset. |
| 11 | `ChuGyouk/OlympiadBench-Math-Ko` | ChuGyouk, 2024-10-04 | 674 [P] | 0 new | Korean translation of #10 [P] | ko | adds `question_ko` | mit | Translation. Tier B. |
| 12 | `RUC-AIBOX/OlymMATH` | RUC AI Box, 2025-03-27 | 550 rows [P]; 350 problems (200 natural-language, 150 Lean) per card v3 [S/P card] | ~200 | Taken by hand from printed publications. Bilingual, human checked [P card; S arXiv 2503.21380] | zh, en | `problem, answer, subject, en_informal, zh_informal, en_nl_proof, zh_nl_proof, formal_statement, formal_proof` | mit | High quality, and pairs the informal text with Lean. Offline-sourced, so overlap with AoPS is low. |
| 13 | `RUC-AIBOX/OlymMATH-eval` | RUC AI Box, 2025-05-08 | 579,200 [P] | 0 new | Model responses on OlymMATH [P] | zh, en | `response, correctness, ...` | mit | Model traces. Tier C. |
| 14 | `lkevincc0/IMO_Shortlist_2006_2024_423` | lkevincc0, 2026-03-05 | 423 [P] | 423 | IMO Shortlist 2006 to 2024, with solutions (likely official) [P card] | en | `id, year, problem, solution` | apache-2.0 [P] | A redistribution licence on IMO SL material is doubtful. Fully covered by #3. |
| 15 | `lkevincc0/Putnam_1995_2025_372` | lkevincc0, 2026-03-05 | 372 [P] | 372 | Putnam 1995 to 2025 with solutions (Kedlaya archive style, UNVERIFIED) | en | `id, year, problem, solution` | apache-2.0 | Overlaps #29-#33. |
| 16 | `jonaskg/imo-problems-complete` | jonaskg, 2025-07-22 | no info [P] | ~400 IMO, UNVERIFIED | IMO problems. UNVERIFIED | en | ? | mit | Viewer off. Check the card before use. |
| 17 | `di-zhang-fdu/IMO_pdfs` | di-zhang-fdu, 2025-07-07 | 65 PDFs [P] | ~400, UNVERIFIED | PDFs of IMO papers [P] | multi | `pdf` | none | Raw PDFs. Useful for OCR cross-checks. Do not execute or render untrusted files. |
| 18 | `sandropa/aops-problems` | sandropa, 2026-04-10 | 43,218 [P] | ~15 to 25K | Scrape of the AoPS wiki (`contest, name, link, problem_html, solution_1..5`) [P] | en | `category, contest, link, name, problem, problem_html, solution_1..5, source, which` | mit (licence doubtful; the AoPS wiki has its own terms) | Wiki pages are mostly AMC, AIME, USAMO and IMO with human solutions. Each row has a stable link, so it is good for provenance. |
| 19 | `hudsongouge/AoPS-Scrape` | hudsongouge, 2026-07-09 | 69,502 [P] | unclear; likely AoPS course and Alcumus items | AoPS class or lesson problems (`class_id, lesson_id, aops_problem_id, content_hash`) [P] | en | `aops_problem_id, content_hash, answer, alt_answers, ...` | other | Probably proprietary AoPS course content. Legal risk. Use the IDs and hashes only as a dedup signal. |
| 20 | `di-zhang-fdu/AOPS` / `pkuAI4M/AOPS` | di-zhang-fdu, 2024-09-13; pkuAI4M, 2024-09-02 | 3,886 each [P] | ~3.9K | AoPS wiki AMC/AIME with `link` and `letter` answer [P] | en | `problem, solution, answer, letter, link` | none | Two identical mirrors. Contained in #18. |
| 21 | `sparsh35/aops` | sparsh35, 2024-04-19 | 7,143 [P] | ~7K | AoPS wiki scrape with `link` [P] | en | `id, no, link, problem, solution, answer` | apache-2.0 | Overlaps #18 and #20. |
| 22 | `jablonkagroup/us-olympiad-problems` | jablonkagroup, 2025-05-09 | 1,219 [P] | ~1.2K, UNVERIFIED | US olympiad problems (mix of chemistry and math, UNVERIFIED) [P columns] | en | `problem_statement, solution, options, correct_answer` | none | Check the subject before ingest. |
| 23 | `Navvye/Olympiad-Problems-CSV` | Navvye, 2024-08-06 | no info [P] | UNVERIFIED | CSV of olympiad problems. UNVERIFIED | en | ? | mit | Small. Low priority. |
| 24 | `devarajns/olympiadmath-corpus` / `introvoyz041/olympiads` | 2026-08-20 / 2026-04-25 | no info [P] | UNVERIFIED | Likely re-uploads of Numina olympiads. UNVERIFIED | ? | ? | none | Mirrors. Skip them. |

## 2. Benchmarks with human-written problems (small, high quality)

| # | HF id | Publisher, created | Rows | Est. distinct human problems | Origin; solutions | Lang | Key columns | Licence | Notes |
|---|---|---|---|---|---|---|---|---|---|
| 25 | `MathArena/*`: aime_2023_I (15), aime_2024_I (15), aime_2025 (30), aime_2026 (30), hmmt_feb_2025 (30), hmmt_feb_2026 (33), hmmt_nov_2025 (30), brumo_2025 (30), cmimc_2025 (40), smt_2025 (53), usamo_2025 (6), usamo_2026 (6), imo_2025 (6), imc_2025 (10), putnam_2025 (12), miklos_2025 (10), kangaroo_2025 (168), apex_2025 (12), apex-shortlist (47), final_answer_comps (139), arxivmath (103), project_euler (50) | ETH SRI Lab (MathArena), 2025-04 to 2026-03 | ~900 total [P] | ~800 competition (arxivmath and project_euler are not olympiad) | Copied from official contest papers. The proof sets carry `grading_scheme`, `points`, and for USAMO a `sample_solution` [P] | en | `problem_idx, problem, answer, problem_type, grading_scheme, source, competition` | cc-by-nc-sa-4.0 (arxivmath cc-by-sa-4.0) [P] | Clean, recent and dated. NC licence: fine to cite and link, but take care before redistribution. Site https://matharena.ai [P]. |
| 26 | `Hwilner/imo-answerbench` (also `OpenEvals/IMO-AnswerBench`, `dots-studio/IMO-AnswerBench-Verified`) | Google DeepMind IMO-Bench re-uploads, 2025-11-05; 2026-01-23; 2026-08-21 | 400 each [P] | ~400 (some are modified versions of originals) | IMO-Bench answer split. Problems from IMO and other sources, some rewritten to resist memorisation [P card; S arXiv 2511.01846] | en | `Problem ID, Problem, Short Answer, Category, Subcategory, Source` | cc-by-4.0 | The `Source` column traces each item. The "Verified" copy fixes answers (UNVERIFIED which ones). |
| 27 | `Hwilner/imo-proofbench` / `lm-provers/IMOProofBench` | 2025-11-05 | 60 each [P] | 60 | IMO-Bench proof split, with grading guidelines [P] | en | `Problem, Solution, Grading guidelines, Level, Source` | cc-by-4.0 / none | Good template for a rubric schema. |
| 28 | `Hwilner/imo-gradingbench` | 2025-11-05 | 1,000 [P] | 0 new (graded model responses) | IMO-Bench grading split. Model answers with human grades [P] | en | `Response, Points, Reward, Grading guidelines, Problem Source` | cc-by-4.0 | Tier C. Useful for training graders. |
| 29 | `amitayusht/PutnamBench` | Tsoukalas et al. (UT Austin), 2024-06-11 | 522 [P] (GitHub now ~660, UNVERIFIED) | ~520 | Putnam 1962 to 2023. Informal statement and solution plus Lean 4, Isabelle and Coq [P; GitHub https://github.com/trishullab/PutnamBench] | en | `name, informal_statement, informal_solution, lean4_statement, isabelle_statement, coq_statement, tags` | none on HF (GitHub: Apache-2.0 and MIT, UNVERIFIED) | Formal plus informal. The HF copy is older than the GitHub repo. |
| 30 | `Putnam-AXIOM/putnam-axiom-dataset-ICML-2025-522` | Putnam-AXIOM (Stanford), 2025-05-06 | 1,122 [P] | 522 originals | Putnam 1938 to 2023 originals plus functional variations [P card; S arXiv 2508.08292] | en | `original_problem, original_solution, problem, solution, variation, year, type` | none | Filter on `variation==0` for originals (UNVERIFIED flag semantics). The v1 repo is gated [P]. |
| 31 | `blackhao0426/PutnamGAP` | blackhao0426, 2026-04-09 | 1,051 [P] | ~1,051 originals, UNVERIFIED (variants are in JSON columns) | Putnam with renamed-variable variants [P] | en | `index, question, solution, type, difficulty, variant_*_json` | cc-by-4.0 | Originals usable. The variants are Tier C. |
| 32 | `yangzhch6/Putnam-Informal-1995-2024` | yangzhch6, 2025-10-27 | 360 [P] | 360 | Putnam 1995 to 2024 informal problems and solutions [P] | en | `id, question, solution` | none | Overlaps #15. |
| 33 | `yangzhch6/DeepInformal-Putnam-1995-2024` | yangzhch6, 2025-10-28 | 356 [P] | 0 new | Adds synthetic thinking traces [P] | en | `ground_truth_proof, synthetic_thinking_response` | none | Tier C. |
| 34 | `tbetton/putnam` | tbetton | no info [P] | UNVERIFIED | Putnam. UNVERIFIED | en | ? | ? | Low priority. |
| 35 | `Jiahao004/HMMT_FIMO_Putnam` | Jiahao004, 2025-06-06 | no info [P] | UNVERIFIED | HMMT, FIMO and Putnam theorem-style eval for DeepTheorem [S] | en | ? | mit | Viewer off. |
| 36 | `MathOdyssey/MathOdyssey` | Fang et al., 2024-11-19 | 389 [P] | ~389 | Written by experts (olympiad, high school and university levels) [S arXiv 2406.18321] | en | `problem_number, problem_statement, answer, reasoning, label` | none | New problems written by humans. Small. |
| 37 | `meituan-longcat/AMO-Bench` | Meituan LongCat, 2025-10-29 | 50 [P] | 50 | New problems written by experts at IMO level, answer-based [P card] | en | `question_id, prompt, solution, answer, answer_type` | mit | Original, not from past contests. |
| 38 | `ByteDance-Seed/BeyondAIME` | ByteDance Seed, 2025-06-17 | 100 [P] | 100 (revised) | From high-school and university contests, "manually revised to be unique" [P card] | en | `problem, answer` | cc0-1.0 | Revised versions. Provenance is not given per item. |
| 39 | `nvidia/Nemotron-IMO-Bench` | NVIDIA, 2026-09-03 | 200 [P] | 200 | New problems and proofs written with Titu Andreescu, never published before [P card] | en | `id, problem, proof, subject` | cc-by-4.0 | High-value human proofs. The card asks that it not be used for training. |
| 40 | `notadib/math-contests-2026` | notadib, 2026-06-25 | 534 [P] | ~534 | 2026 contest problems with `source`, `country`, `date`, `reference_solutions` and an answer audit [P] | en | `problem_id, statement, source, country, date, reference_answer, reference_answer_audit, reference_solutions, difficulty_*` | cc-by-4.0 | Fresh 2026 problems with good provenance. |
| 41 | `notadib/AOBench` | notadib, 2026-09-21 | 114 [P] | 114 | Olympiad problems with a `url`, `country` and `year` [P] | en | `problem_id, statement, solutions, url, country, year, dataset, source` | cc-by-4.0 | Small and traceable. |
| 42 | `palaestraresearch/ucmo` | Palaestra Research, 2026-04-16 | 429 [P] | ~429 | Contest problems with `contest_date`, the original grading scheme and a translated one, and contested alternatives [P] | en | `id, contest_date, answer, grading_scheme_original, grading_scheme_translated, ...` | mit | Rich provenance fields. |
| 43 | `humanfia-lab/IMO2026` / `lmms-lab/imo-2025` | 2026-07-18 / 2025-07-18 | no info / 6 [P] | 6 + 6 | IMO 2025 and IMO 2026 papers. The 2025 copy has solutions (author unknown, UNVERIFIED) | en | `id, problem, solution` | other / mit | Use the official imo-official.org text as master. |
| 44 | `HAGeo-IMO/HAGeo-409` | HAGeo-IMO, 2025-11-26 | 409 [P] | ~409 geometry | Olympiad geometry in natural language plus AlphaGeometry DSL [P] | en | `Problem_ID, Natural_Language, AlphaGeometry, HAGeo, Difficulty_Score` | mit | Geometry only, with a formal-DSL bridge. |
| 45 | `YauheniShe/OlyGeo` | YauheniShe, 2026-10-01 | 10,400 [P] | <10K; includes augmented items | Olympiad geometry with `competition, country, human_reviewed, augmentation_rule` [P] | en | `canonical_task_id, competition, country, human_reviewed, augmentation_*` | other | Filter on `augmentation_rule` null and `human_reviewed` true (UNVERIFIED semantics). |
| 46 | `jojo23333/LiveAoPSBench-2024` | Mahdavi et al. (UBC), 2025-01-27 | 5,328 [P] | ~5.3K | AoPS posts made in 2024, with `post_time`. Answers taken from forum posts by LLM [P; S arXiv 2501.14275] | en | `idx, question, solution, answer, post_time` | apache-2.0 | Has timestamps. Problems are human. Solutions were extracted by LLM. |
| 47 | `LARG/OMIBench` | LARG, 2026-01-06 | 1,322 [P] | ~1.3K, multimodal | Olympiad problems with images [P] | en | `question, solution, answer, image_list, known_issue` | cc-by-4.0 | Has a `known_issue` column, which is nice. Source tracing UNVERIFIED. |
| 48 | `GAIR/OlympicArena` | GAIR, 2024-06-10 | 10,615 (all subjects) [P] | ~2 to 3K math, UNVERIFIED | Olympiad problems across 7 subjects [P; S arXiv 2406.12753] | en, zh | `problem, solution, answer, subject, figure_urls, language` | cc-by-nc-sa-4.0 | Multi-subject, so filter on `subject=Math`. |
| 49 | `AI4Math/IneqMath` | AI4Math, 2025-05-15 | no info [P] | ~1.2K, UNVERIFIED | Olympiad inequalities, with test items written by experts [S arXiv 2506.07927] | en | ? | cc-by-sa-4.0 | Inequality focus. |
| 50 | `openbmb/MA-ProofBench` | OpenBMB, 2026-06-10 | 200 [P] | ~200 | Informal and formal Lean statements [P] | en | `informal_statement, formal_statement, header, topic` | mit | Lean set that also has informal text. |
| 51 | `Goedel-LM/MathOlympiadBench` | Goedel-LM, 2025-07-15 | 360 [P] | ~360 | Olympiad problems with Lean 4 and an `informal_prefix` [P] | en | `problem_id, informal_prefix, formal_statement, category, tags` | apache-2.0 | Formal plus informal. |
| 52 | `hoskinson-center/proofnet` / `hoskinson-center/minif2f-lean4` | Hoskinson Center, 2022-11-17 | 371 / ~488 [P] | 371 undergrad / ~244 olympiad (AMC, AIME, IMO in miniF2F) | Textbook and competition problems with Lean statements [P] | en | `nl_statement, nl_proof, formal_statement` | mit / apache-2.0 | miniF2F gives informal text for AMC, AIME and IMO items. |
| 53 | `wenjiema02/ProofBench`, `lm-provers/ProofBench` | 2025-10-13; 2026-01-09 | 899; 290 [P] | ~145; ~290 | Competition proofs with expert ratings of model proofs [P] | en | `problem, reference_solution, marking_scheme, model_solution, expert_rating` | mit / none | Rubrics and grading. Tier B/C. |
| 54 | `HuggingFaceH4/MATH-500`, `EleutherAI/hendrycks_math` | HF H4, 2024-11-15; EleutherAI, 2023-09-14 | 500; 12,500 [P] | 12.5K (AMC, AIME and AoPS-based) | Hendrycks MATH (2021). Human solutions from AoPS [S arXiv 2103.03874] | en | `problem, solution, level, type` | none / mit | The original `hendrycks/competition_math` is DMCA'd: no data, 0 downloads [P]. Licence risk. Overlaps AoPS and AMC heavily. |
| 55 | `project-numina/aimo-validation-aime`, `-amc`, `-math-level-5` | Numina, 2024-07-09 | 90; 83; 721 [P] | ~890 | AIME 2022 to 24, AMC 2022 to 23 and MATH L5, with `url` [P] | en | `id, problem, answer, url` | apache-2.0 / none | Small and linked. |
| 56 | AIME full sets: `Pandores/aime-1983-2026`, `gneubig/aime-1983-2024`, `di-zhang-fdu/AIME_1983_2024`, `Maxwell-Jia/AIME_2024`, `math-ai/aime25`, `math-ai/aime26`, `kaggle-aimo/aime_filtered` | various, 2024 to 2026 | 1,065; 933; 933; 30; 30; 30; 874 [P] | 1,065 total | Official AIME. Pandores has `solutions` and `all_answers` [P] | en | `year, part, problem, answer, solutions` | none / cc0-1.0 / mit / apache-2.0 | Pandores is the most complete (1983 to 2026). The others are subsets. |
| 57 | AMC sets: `edev2000/amc12-full`, `kaggle-aimo/amc_filtered`, `math-ai/amc23` | 2026-03-17; 2024-06-23; 2025-02-13 | 1,199; 1,081; 40 [P] | ~1.2K AMC12 | AMC problems [P] | en | `problem_id, question, answer, difficulty` | none | Answer-only. Pair them with AoPS wiki solutions (#18). |
| 58 | `TIGER-Lab/TheoremQA`, `UGMathBench/ugmathbench`, `JVRoggeveen/HARDMath2`, `ethz-spylab/RealMath` | various | 800; 5,061; 211; 1,286 [P] | not olympiad | University or research math | en | | mit / gpl-3.0 / none | Out of scope. Listed to rule them out. |
| 59 | `skylenage-ai/HLE-Verified` / `cais/hle` | 2026-02-13 / 2025-01-23 | 2,500 / gated [P] | small olympiad share | HLE questions written by experts. The verified copy adds validity flags [P] | en | `question, answer, problem_is_valid, ...` | none / mit (gated) | Not olympiad-focused. Gated. |
| 60 | `ulamai/UnsolvedMath` | ulamai, 2026-01-24 | 19,031 [P] | 0 olympiad | Open problems | en | | cc-by-4.0 | Out of scope. |
| 61 | `nguyen599/AstralBench` | nguyen599, 2026-02-13 | 50 [P] | 50, UNVERIFIED | Olympiad answer-style benchmark [P] | en | `id, problem, answer` | cc-by-sa-4.0 | Origin UNVERIFIED. |
| 62 | `MathLLMs/MathVision` | MathLLMs, 2024-02-22 | 3,344 [P] | ~3.3K visual contest items | Real competition problems with images (Kangaroo, AMC, etc.) [S arXiv 2402.14804] | en | `question, image, options, answer, solution, level, subject` | mit | Visual items. Overlaps Kangaroo. |

## 3. AoPS-derived large corpora (Tier B)

| # | HF id | Publisher, created | Rows | Est. distinct human problems | Origin; solutions | Lang | Key columns | Licence | Notes |
|---|---|---|---|---|---|---|---|---|---|
| 63 | `nvidia/OpenMathReasoning` | NVIDIA, 2025-04-22 | cot 3,201,061; tir 1,718,466; genselect 565,620; additional_problems 193,170 [P] | ~306K problems in the solution splits + 193K extra (card); about 125K from the HS-olympiad category [P] | AoPS forum problems extracted by LLM. Solutions generated by DeepSeek-R1 and QwQ [P card; S arXiv 2504.16891] | en | `problem, problem_source, problem_type, expected_answer, generated_solution, generation_model, pass_rate_72b_tir` | cc-by-4.0 | `problem_source` counts in the cot split, per row [P]: aops_c6_high_school_olympiads 124,998; aops_c4_high_school_math 58,999; aops_c7_college_math 44,178; MATH_training_set 9,112; aops_c5_contests_amp_programs 7,243. Rows repeat each problem over many solutions. The problems are rewritten by LLM, so source links are lost. |
| 64 | `nvidia/Nemotron-RL-math-OpenMathReasoning` | NVIDIA, 2025-11-14 | 112,897 [P] | subset of #63 | RL subset [P] | en | `question, expected_answer` | cc-by-4.0 | Subset. |
| 65 | `nvidia/Nemotron-Math-v2` | NVIDIA, 2025-12-15 | 7,085,839 [P] | subset of AoPS / OMR, UNVERIFIED | Synthetic traces. Has `url`, `user_name` and `user_url` pointing to AoPS posts [P columns] | en | `problem, expected_answer, data_source, url, user_name, user_url, messages` | cc-by-4.0, cc-by-sa-4.0 | The `url` and `user_url` columns are a rare chance to recover AoPS provenance. |
| 66 | `nvidia/Nemotron-Math-Proofs-v1` / `-v3-SFT` / `-v3-RL` | NVIDIA, 2025-12-14; 2026-09-03 | 243,314; 7,355; 9,597 [P] | ~?, UNVERIFIED | AoPS proof problems with `url, user_name` and Lean statements. Proofs are synthetic [P] | en | `problem, source, url, user_name, formal_statement, messages` | cc-by-sa-4.0 / cc-by-4.0 | Problems carry AoPS URLs. Proofs are Tier C. |
| 67 | `DeepStudentLlama/AoPS-Instruct` | Mahdavi et al. (UBC), 2025-01-27 | 1,198,115 [P] (card: ~650K QA) | ~650K posts; olympiad share UNVERIFIED | AoPS posts. Both `original_question` and `rewritten_question` (rewritten by Qwen), plus answers [P; S arXiv 2501.14275] | en | `original_question, original_answers, rewritten_question, rewritten_answers, metadata` | mit (AoPS terms apply) | The original text is kept, which is good. `metadata` has the AoPS topic id (UNVERIFIED). |
| 68 | `mlfoundations-dev/aops_forum_filtered` | mlfoundations-dev, 2025-01-23 | 17,755 [P] | subset of Numina `aops_forum` | Filter of Numina aops_forum [P] | en | `problem, solution, source, messages` | none | Subset. |

## 4. Training mixes with provenance labels (Tier B; partly C)

| # | HF id | Publisher, created | Rows | Est. distinct human problems | Origin; solutions | Lang | Key columns | Licence | Notes |
|---|---|---|---|---|---|---|---|---|---|
| 69 | `project-numina/NuminaMath-1.5` | Project Numina, 2025-02-10 | 896,215 [P] | ~300K from non-synthetic olympiad-like sources (see §7) | Mix: Chinese olympiad PDFs (translated), AoPS, cn_k12, official refs, plus synthetic items. Solutions are mostly human (translated or cleaned); synthetic items have GPT-4o solutions [P card] | en | `problem, solution, answer, problem_type, question_type, problem_is_valid, solution_is_valid, source, synthetic` | apache-2.0 | The `synthetic` flag: False 584,555, True 311,660 [P]. The key upstream for many sets below. |
| 70 | `project-numina/NuminaMath-CoT` / `-TIR` | Project Numina, 2024-07-15 / 07-16 | 859,594 / 72,540 [P] | superseded by 1.5 | Version 1.0. Solutions reformatted to CoT by GPT-4o [P card] | en | `source, problem, solution, messages` | apache-2.0 | `source` counts [P]: cn_k12 276,554; synthetic_math 167,874; orca_math 153,314; olympiads 150,563; synthetic_amc 62,108; aops_forum 30,192; math 7,477; gsm8k 7,342; amc_aime 4,070. |
| 71 | `project-numina/PolyUniMath` | Numina, 2025-08-25 | 4,290,861 [P] | low | Multilingual translations and documents (`lang, doc_id, paper_score`) [P] | multi | | none | Mostly not olympiad. Tier C. |
| 72 | `nlile/NuminaMath-1.5-RL-Verifiable` | nlile, 2025-03-25 | 131,063 [P] | subset of #69 | Filtered to verifiable answers [P] | en | same as 1.5 | apache-2.0 | Subset. |
| 73 | `open-r1/OpenR1-Math-220k` / `OpenR1-Math-Raw` | HF Open-R1, 2025-02-10 / 02-12 | 450,258 (all configs) / 516,499 [P] | subset of Numina 1.5 (~220K problems) | Numina 1.5 problems plus R1 traces [P card] | en | `problem, solution, answer, source, generations, correctness_*` | apache-2.0 | Traces are Tier C. Problems are Numina. |
| 74 | `Elliott/Openr1-Math-46k-8192` | Elliott, 2025-04-20 | 45,792 [P] | subset | Filtered OpenR1 [P] | en | `prompt, reward_model, data_source` | mit | Subset. |
| 75 | `SynthLabsAI/Big-Math-RL-Verified` | SynthLabs, 2025-02-20 | gated, no info [P]; card says ~251K [S arXiv 2502.17387] | mostly Numina + Omni + HARP + new "Big-Math Reformulated" | Mix [S] | en | `problem, answer, source, domain, llama8b_solve_rate` (UNVERIFIED) | apache-2.0 | Gated. Has a reformulated subset (Tier C). |
| 76 | `zwhe99/DeepMath-103K` | Tencent AI Lab, 2025-04-14 | 103,022 [P] | mixed | Hard, decontaminated problems from several sources; three R1 solutions each [P card; S arXiv 2504.11456] | en | `question, final_answer, difficulty, topic, r1_solution_1..3` | mit | Has no source column, so provenance is lost. Use it only for difficulty labels. |
| 77 | `Skywork/Skywork-OR1-RL-Data` | Skywork, 2025-04-12 | 119,112 (math + code) [P] | subset of Numina 1.5 | `data_source` like `train-math-numinamath1.5_olympiads` [P] | en | `prompt, data_source, reward_model, extra_info` | none | The source string maps straight back to the Numina source. |
| 78 | `BytedTsinghua-SIA/DAPO-Math-17k` | ByteDance/Tsinghua, 2025-03-17 | 1,791,700 rows (17K problems × ~100 repeats) [P] | ~17K (AoPS + official sites, per paper [S arXiv 2503.14476]) | Answers changed to integers [S] | en | `prompt, data_source, reward_model` | apache-2.0 | The rewrites alter answers. Tier B. |
| 79 | `agentica-org/DeepScaleR-Preview-Dataset` | Agentica, 2025-02-09 | 40,315 [P] | ~40K from AIME 1984 to 2023, AMC, Omni-MATH and STILL [P card] | Mix [P] | en | `problem, answer, solution` | mit | No source column. |
| 80 | `RUC-AIBOX/STILL-3-Preview-RL-Data` | RUC AI Box, 2025-01-26 | 29,925 [P] | mix (MATH, Numina, AIME) | [P card] | en | `question, answer, messages` | none | No source column. |
| 81 | `POLARIS-Project/Polaris-Dataset-53K` | POLARIS, 2025-06-12 | 53,291 [P] | subset of DeepScaleR + AReaL | [P card] | en | `problem, answer, difficulty` | apache-2.0 | No source column. |
| 82 | `nvidia/AceReason-Math` | NVIDIA, 2025-06-02 | 49,585 [P] | Numina + DeepScaleR | [P card] | en | `problem, answer` | cc-by-4.0 | No source column. |
| 83 | `ycchen/Crystal-Math-Preview` | ycchen, 2026-02-09 | 4,149 [P] | ~4.1K | Keeps the `original_problem` and `original_solution` alongside a cleaned version, with `source` and a per-item `license` [P] | en | `original_problem, problem, source, license, *_pass_rate` | other | A good model for recording per-item licences. |
| 84 | `AIMO-Corpus/PolyMath` | AIMO-Corpus, 2026-02-09 | 22,180 [P] | ~22K, UNVERIFIED | Problems with `metadata` [P] | en | `id, problem, solution, answer, metadata` | cc-by-4.0 | Do not confuse it with Qwen/PolyMath. Origin UNVERIFIED. |
| 85 | `Metaskepsis/Olympiads` | Metaskepsis, 2025-01-08 | 32,926 [P] | subset of Numina `olympiads` | [P] | en | `id, problem, solution, answer, source` | mit | Subset. |
| 86 | `Emulated-Inc/olympiad-math-training-pool`, `Emulated-Inc/forum-competition-math-training-pool` | Emulated Inc, 2026-09-12 | 184,380; 245,504 [P] | subsets of Numina 1.5 | Same schema as Numina 1.5 [P] | en, zh | Numina 1.5 columns | apache-2.0 | Re-cut Numina. |
| 87 | `PrimeIntellect/verifiable-math-problems`, `PrimeIntellect/NuminaMath-QwQ-CoT-5M` | Prime Intellect, 2025-02 / 2025-01 | 777,457; 5,138,102 [P] | Numina-derived | `source, in_source_id` [P] | en | | none / mit | `in_source_id` helps link rows back to their upstream source. |
| 88 | `allenai/RLVR-MATH`, `hkust-nlp/dart-math-hard`, `open-thoughts/OpenThoughts3-1.2M`, `simplescaling/s1K`, `s1K-1.1`, `GAIR/LIMO`, `EricLu/SCP-116K` | various | 7.5K; 585K; 1.2M; 1K; 1K; 817; 259K [P] | none new | Traces or subsets of MATH, Numina and OlympicArena [P cards] | en | | various | Tier C. They only show which problems are "popular". s1K has `source_type`. |
| 89 | `TamasSimonds/imo-dataset`, `TamasSimonds/olympiad-proof-problems` | 2025-08 | 370; 39,764 [P] | ~370; subset of Numina proofs | `prompt, completion, source` [P] | en | | mit | Completion is model-written. |
| 90 | `Jiahao004/DeepTheorem` | Tencent, 2025-05-22 | 120,754 [P] | none new | Theorems rewritten from web and olympiad sources; variants have positive or negative truth values [P] | en | `informal_theorem, proof, source, truth_value, ori_question, ori_solution` | mit | Variants are Tier C. `ori_question` could be the human original. |
| 91 | `mihailgribov/olympiad_style_integer_math_problems` | mihailgribov, 2026-02-10 | 59,486 [P] | 0 | Synthetic, built from lemma graphs [P] | en | | cc-by-4.0 | Tier C. |
| 92 | `nguyen599/AstralMath-v1` | nguyen599, 2026-02-09 | 431,861 [P] | mixed | Mix with per-row `license` and `source` [P] | en | | multi | Tier C. |
| 93 | `lm-provers/FineProofs-SFT`, `m-a-p/OProofs`, `KokunoYumeto/open-math-proof-checks` | 2026 | 12,058; 6.8M; 10,077 [P] | none new | Proof traces. FineProofs has `competition` and `source`. open-math-proof-checks has authorship and attribution audit columns [P] | en | | apache-2.0 / cc0-1.0 | Tier C. Look at open-math-proof-checks for audit-schema ideas. |
| 94 | `internlm/Lean-Workbook` | InternLM, 2024-06-06 | 25,214 [P] | ~? AoPS-derived | Natural-language statements plus Lean [P] | en | `natural_language_statement, formal_statement` | apache-2.0 | Statements were auto-formalised from AoPS-like sources. |
| 95 | `EleutherAI/proof-pile-2`, `hoskinson-center/proof-pile`, `wellecks/naturalproofs-gen` | 2022 to 2023 | large / 362,502 / 29,396 [P] | ~0 olympiad | Web, arXiv and textbook math | en | | various | Out of scope. |

## 5. Multilingual and regional

| # | HF id | Rows | Est. distinct human problems | Notes |
|---|---|---|---|---|
| 96 | `Qwen/PolyMath` (Alibaba Qwen, 2025-04-25) | 9,000 [P] = 500 problems × 18 languages | ~500 (125 per level) | Translations checked by experts. The top levels draw on olympiad and frontier sources [P card; S arXiv 2504.18428]. Licence: none on the Hub. |
| 97 | `fedric95/AIME2025-Multilingual`, `lightonai/aime25_multilingual`, `ytu-ce-cosmos/aime25-tr`, `iapp/aimo-validation-aime-th` | 180; 180; 30; 90 [P] | 0 new | Translations of AIME. |
| 98 | `ytu-ce-cosmos/tubitak-science-olympiad-tr` (2026-03-21) | 2,698 [P] | ~2.7K, but includes science subjects | Turkish TÜBİTAK olympiad items, with images. cc-by-4.0 [P]. Filter to math. |
| 99 | `hafidhsoekma/math-olympiad-indonesian-benchmark` (2025-08-20) | 2,204 [P] | ~2.2K | Indonesian olympiad problems with `source` [P]. No licence. |
| 100 | `HAERAE-HUB/HRM8K` (2025-01-13) | 8,011 [P] | ~1.5K KMO, UNVERIFIED | Korean. Includes KMO and KSM subsets plus translated sets [P card]. mit. |
| 101 | `ura-hcmut/Vietnamese-MATH` | 12,500 [P] | 0 new | Vietnamese translation of MATH. |
| 102 | `Vikhrmodels/russian_math` | gated [P] | UNVERIFIED | Russian. Gated. |
| 103 | `weitianwen/cmath`, `RUCAIBox/gaokao-bench`, `hails/agieval-gaokao-mathqa`, `choucsan/Gaokao-Compass-11M` | 1,698; 1,781; 351; ? [P] | not olympiad | Chinese school and gaokao exams. Out of scope, except as a negative filter. |
| 104 | `CohereLabs/MGSM` | ~2.7K [P] | 0 | Grade-school. Out of scope. |

## 6. Kaggle and GitHub

| # | Location | What | Size | Notes |
|---|---|---|---|---|
| K1 | Kaggle AIMO Progress Prize 1, https://www.kaggle.com/competitions/ai-mathematical-olympiad-prize [S] | `train.csv` of 10 reference problems; hidden test set of 50 | 10 public | The rules restrict reuse of the data (UNVERIFIED). |
| K2 | Kaggle AIMO Progress Prize 2, https://www.kaggle.com/competitions/ai-mathematical-olympiad-progress-prize-2 [S] | 10 reference problems; 50 hidden | 10 public | Same pattern. NVIDIA's winning entry led to OpenMathReasoning (`used_in_kaggle` column [P]). |
| K3 | Kaggle AIMO Progress Prize 3 (2025 to 26), https://www.kaggle.com/competitions/ai-mathematical-olympiad-progress-prize-3 [S] | reference problems; hidden test | UNVERIFIED | Counts UNVERIFIED. |
| K4 | Kaggle community datasets (mirrors of MATH, AIME 1983 to 2024, Numina) [S] | mirrors | none | No new problems. The `kaggle-aimo/*` HF ids (#56, #57) are copies. |
| G1 | https://github.com/OpenBMB/OlympiadBench [P] | OlympiadBench source | 8,476 problems (math + physics) | MIT. |
| G2 | https://github.com/hendrycks/math [P] | MATH | 12,500 | MIT code. The HF copy was DMCA'd. |
| G3 | https://github.com/KbsdJames/Omni-MATH [P] | Omni-MATH | 4,428 | apache-2.0. |
| G4 | https://github.com/RUCAIBox/OlymMATH [P] | OlymMATH | 350 | mit. |
| G5 | https://github.com/trishullab/PutnamBench [P] | PutnamBench | ~660, UNVERIFIED | Newer than the HF copy. |
| G6 | https://github.com/eth-sri/matharena [P] | MathArena | ~900 | Code plus contest data. |
| G7 | https://github.com/google-deepmind/superhuman (IMO-Bench) [S] | IMO-Bench | 400 + 60 + 1,000 | Official upstream of #26-#28. Path UNVERIFIED. |
| G8 | https://github.com/openai/miniF2F, https://github.com/zhangir-azerbayev/ProofNet [P] | Lean plus informal | 488; 371 | Formal with informal text. |
| G9 | https://github.com/QwenLM/PolyMath [P] | PolyMath | 500 × 18 | none |
| G10 | https://github.com/project-numina/aimo-progress-prize [S] | Numina pipeline | none | Describes the source scraping (UNVERIFIED path). |

## 7. NuminaMath source value counts (for dedup and provenance)

From the datasets-server `/statistics` endpoint for `project-numina/NuminaMath-1.5`, config `default`, split `train`, column `source` [P]. URL: https://datasets-server.huggingface.co/statistics?dataset=project-numina/NuminaMath-1.5&config=default&split=train

| source | rows | what it is (per card [P]) | tier | olympiad? |
|---|---|---|---|---|
| cn_k12 | 268,819 | Chinese K-12 exercises, translated | B | no |
| olympiads | 197,084 | Chinese olympiad and competition PDFs, translated | B | yes, mostly |
| orca_math | 151,934 | Orca-Math synthetic word problems | C | no |
| synthetic_math | 148,712 | GPT-4 rewrites of MATH | C | no |
| aops_forum | 67,841 | AoPS forum | B | yes, mostly |
| cn_contest | 29,944 | Chinese contest problems | B | yes |
| metamath | 11,014 | MetaMath (augmented GSM8K and MATH) | C | no |
| inequalities | 7,314 | Inequality collection | B | yes |
| amc_aime | 5,872 | AMC and AIME | A/B | yes |
| number_theory | 4,043 | Number theory collection | B | yes |
| olympiads_ref | 3,638 | Official solutions (Numina olympiads, #3) | A | yes |
| **total** | 896,215 | `synthetic` False 584,555, True 311,660 [P] | | |

The olympiad-relevant non-synthetic sources sum to about 316K rows: olympiads, aops_forum, cn_contest, inequalities, amc_aime, number_theory and olympiads_ref. Many rows fail `problem_is_valid` or `solution_is_valid`, or are near-duplicates, so the real count is lower.

NuminaMath-CoT 1.0 `source` counts are in row 70. `synthetic_amc` (62,108) is a 1.0-only synthetic source. Treat it as Tier C.

Other provenance columns worth keeping, all [P]:
- **OpenMathReasoning `problem_source`** (cot split, counted per row): aops_c6_high_school_olympiads 124,998; aops_c4_high_school_math 58,999; aops_c7_college_math 44,178; MATH_training_set 9,112; aops_c5_contests_amp_programs 7,243.
- **Omni-MATH `source`**: about 100 competition labels (putnam, imc, apmo, china_national_olympiad, jbmo_shortlist, ...).
- **project-numina/olympiads `exam`**: 28 exams, with `year` and `problem_label`.
- **MathNet**: `country`, `competition`, `id`.
- **Skywork `data_source`**: `train-math-numinamath1.5_<source>`.
- **Nemotron-Math-v2 and Proofs**: `url`, `user_name`, `user_url` (AoPS links).
- **AoPS-Scrape**: `aops_problem_id`, `content_hash`.
- **sandropa/aops-problems, ucmo, math-contests-2026**: `link`, `url`, `contest_date`.

## 8. Tier list

### Tier A: human-written and traceable to a source (ingest as canonical problems)

- **Large:**
  - MathNet (#1)
  - project-numina/olympiads and olympiads-ref-base (#3, #4)
  - AoPS wiki scrapes with links (#18, #20, #21)
  - AI-MO/aops raw forum, with `metadata.path` (#6). This one is A for traceability, but needs heavy filtering.
- **Mid:**
  - Omni-MATH (#7, #8)
  - OlympiadBench COMP (#9)
  - LiveAoPSBench (#46)
  - Pandores AIME (#56)
  - AMC12 sets (#57)
  - Putnam sets (#15, #29, #30 originals, #31 originals, #32)
  - IMO_Shortlist_423 (#14)
  - OlyGeo, human-reviewed rows only (#45)
  - HAGeo (#44)
  - MathVision (#62)
  - TÜBİTAK math (#98)
  - Indonesian benchmark (#99)
  - HRM8K KMO (#100)
- **Small and new:**
  - MathArena (#25)
  - IMO-Bench (#26, #27)
  - OlymMATH (#12)
  - AMO-Bench (#37)
  - Nemotron-IMO-Bench (#39)
  - MathOdyssey (#36)
  - math-contests-2026 (#40)
  - AOBench (#41)
  - ucmo (#42)
  - IMO 2025 and 2026 (#43)
  - MA-ProofBench and Goedel MathOlympiadBench informal text (#50, #51)
  - miniF2F and ProofNet (#52)
  - PolyMath originals (#96)

### Tier B: derived or mixed (ingest the problems only after dedup against Tier A; keep the source label)

- NuminaMath-1.5 non-synthetic sources (#69) and CoT 1.0 (#70)
- OpenMathReasoning problems (#63, #64)
- AoPS-Instruct, `original_question` only (#67)
- Nemotron-Math-v2 and Proofs problems, via their URLs (#65, #66)
- OpenR1 (#73), Big-Math (#75), Skywork (#77), DAPO (#78)
- DeepScaleR, STILL, Polaris, AceReason (#79-#82)
- Crystal-Math (#83), AIMO-Corpus PolyMath (#84), Metaskepsis (#85), Emulated pools (#86), PrimeIntellect (#87)
- OlympicArena math (#48), OMIBench (#47), IneqMath (#49)
- BeyondAIME, which is revised (#38)
- DeepMath (#76)
- Lean-Workbook statements (#94)
- All mirrors and translations: #5, #10, #11, #16, #17, #22-#24, #97, #101

### Tier C: synthetic or model output (ingest only as AI solutions or metadata, never as problems)

- Numina `synthetic_math`, `orca_math`, `metamath`, `synthetic_amc`
- PolyUniMath (#71)
- Model traces in OpenMathReasoning, OpenR1, OlymMATH-eval (#13), s1K, LIMO, OpenThoughts, dart-math, SCP-116K (#88)
- DeepInformal-Putnam (#33)
- DeepTheorem variants (#90)
- Putnam-AXIOM and PutnamGAP variations (#30, #31)
- mihailgribov synthetic (#91)
- AstralMath (#92)
- FineProofs, OProofs, Nemotron proofs (#66, #93)
- IMO-GradingBench and ProofBench model responses (#28, #53). These are good grader-training metadata.

Out of scope: #58-#60, #95, #103 and #104.

## 9. Dedup and provenance strategy

1. **Master order.** Build the canonical problem set from official sources first: MathNet, then Numina olympiads, then MathArena and IMO-Bench, then AoPS wiki pages with links. Every later source attaches to an existing problem, or creates a new one with a lower trust level.
2. **Natural keys.**
   - Where possible, key on (competition, year, round, label). The fields come from Numina `exam/year/problem_label`, MathNet `competition/country/id`, AoPS wiki `contest/name/link`, Pandores `year/part/index`, and MathArena `problem_idx` plus the dataset name.
   - Map all competition names to one controlled vocabulary. Example: Omni `usa_team_selection_test` = Numina `USA_TST`.
3. **Text fingerprints.**
   - Normalise the text: strip LaTeX spacing, unify `\dfrac`/`\frac`, lower-case, and drop "Problem 3." prefixes.
   - Compute exact hashes, then MinHash/LSH on 5-gram shingles, then embedding similarity for paraphrases.
   - LLM rewrites in OpenMathReasoning, AoPS-Instruct (`rewritten_question`), DAPO and BeyondAIME need the embedding pass. The MathNet-Retrieve pairs (#2) can tune the thresholds.
4. **Translated sources.** Numina `olympiads`, `cn_contest` and `cn_k12` are machine-translated from Chinese. Group them separately. Match them to Chinese originals (OlympiadBench zh, OlymMATH zh) with cross-lingual embeddings.
5. **Provenance record per problem.** Store a list of `(dataset_id, dataset_revision_sha, row_key, source_label, licence, retrieved_at)`.
   - `row_key` uses the dataset's own id column (`id`, `unique_id`, `problem_id`, `aops_problem_id`, `uuid`) or a content hash.
   - Keep the Numina `source` and `synthetic` values, the OMR `problem_source`, the Skywork `data_source` and the Nemotron `url`/`user_url`. These connect derived rows back to AoPS threads or exam PDFs.
6. **Validity flags.** Import Numina `problem_is_valid` and `solution_is_valid`, OMIBench `known_issue`, the IMO-AnswerBench-Verified fixes and the math-contests-2026 `reference_answer_audit` as quality signals. Never import them as truth.
7. **Solutions.**
   - Classify each solution as `official`, `human-forum`, `human-translated`, `model:<name>` or `rewritten-by-model`.
   - AoPS-Instruct `original_answers` and AI-MO/aops `candidates` are human-forum.
   - OMR `generated_solution` carries `generation_model`.
   - NuminaMath 1.0 CoT solutions are GPT-4o reformats. Mark them `rewritten-by-model`.
8. **Licences.**
   - Many key sets have no licence: project-numina/olympiads, aops, olympiads-ref-base and Big-Math cards.
   - AoPS content is under AoPS terms.
   - MathArena is NC-SA.
   - Plan to store statements under fair-use quotation with links, or to link only. Record the licence per item, as Crystal-Math and AstralMath do.
9. **Contamination and benchmarks.** Mark benchmark items (MathArena, IMO-Bench, Nemotron-IMO-Bench, AMO-Bench) as `benchmark=true`. Nemotron-IMO-Bench asks that its items not be used for training.

## 10. Estimate of distinct olympiad problems reachable via HF

My own synthesis. The numbers are order-of-magnitude, built from the row counts above.

| Bucket | Raw rows | Est. distinct after dedup |
|---|---|---|
| Official and curated (MathNet 27.8K + Numina olympiads 13.1K + Omni 4.4K + OlympiadBench COMP 2.1K + Putnam ~1K + AIME/AMC ~2.3K + benchmarks ~3K + regional ~6K) | ~60K | **~40 to 50K** (MathNet and Numina overlap on IMO, SL, Balkan and Baltic) |
| AoPS forum (AI-MO/aops 80.7K, OMR c6 olympiad ~125K rows, Numina aops_forum 67.8K, AoPS-Instruct, LiveAoPSBench 5.3K) | >1M | **~80 to 150K** olympiad-level problems, all overlapping each other |
| Chinese olympiad and contest, translated (Numina olympiads 197K + cn_contest 30K) | ~227K | **~60 to 120K** after removing duplicates and invalid rows |
| Other Numina olympiad-like (inequalities, number_theory, amc_aime) | ~17K | ~10K, mostly already counted |
| **Total distinct olympiad-level problems** | | **~150K-300K** |
| Of which source-traceable to exam, year and label | | **~40 to 60K** |

These estimates are UNVERIFIED until the dedup pipeline runs. The biggest uncertainty is how much of the AoPS forum and the Chinese-translated pools is truly olympiad level rather than school level.
