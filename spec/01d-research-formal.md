# 01d. Research: formal statements and proofs

Research date: 2026-10-31. Scope: formal datasets of olympiad problems, mostly Lean 4, plus Isabelle, Coq/Rocq, HOL Light and Metamath. It also covers the state of AI provers, how to keep statements faithful, and a design for olymmath and olymmath-agent.

Legend:
- **[P]** primary source: the repo, dataset card, paper or official page itself.
- **[S]** secondary source: a blog, news story, or another paper reporting the fact.
- **UNVERIFIED**: not confirmed against a primary source.

Counts come from repo trees, dataset cards or the Hugging Face datasets-server, as fetched in October 2026. Repos change, so pin a commit before you import anything.

Sibling pin: tamnd/math2 uses Lean v4.35.0 and Mathlib v4.35.0. Lean v4.35.0 was released on 2026-10-12. [P] https://github.com/leanprover/lean4/releases

---

## 1. Summary table: the datasets that matter most for olymmath

| Dataset | Size | Olympiad coverage | Proofs? | Lean pin | Licence | Id scheme | Src |
|---|---|---|---|---|---|---|---|
| Compfiles | 521 formalized, 515 solved | IMO 274 of 404, USAMO 217 of 306, others | Yes, nearly all | v4.35.0 | Apache-2.0 | `Imo2022P2`, `Usa1999P1` | [P] https://github.com/dwrensha/compfiles |
| Mathlib Archive/Imo | 55 files | IMO 1959 to 2026 (selected) | Yes, all | Mathlib master | Apache-2.0 | `Imo2024Q1` | [P] https://github.com/leanprover-community/mathlib4/tree/master/Archive/Imo |
| formal-imo (DeepMind) | 258 files | IMO 1959 to 2024 | Statements only | v4.27.0 | Apache-2.0 | `imo_2010_p1` | [P] https://github.com/google-deepmind/formal-imo |
| IMOSLLean4 | 171 problems | ISL 2006 to 2025 (mostly A and N) | Yes, all | v4.35.0 | Apache-2.0 | `IMO2019/A1.lean`, `final_solution` | [P] https://github.com/mortarsanjaya/IMOSLLean4 |
| PutnamBench | 672 Lean, 640 Isabelle, 412 Coq | Putnam 1962 to 2025 | Statements only | v4.33.1 | Apache-2.0 (Lean, Isabelle), MIT (Coq) | `putnam_1988_b1` | [P] https://github.com/trishullab/PutnamBench |
| miniF2F (several forks) | 488 (244 valid, 244 test) | AMC, AIME, IMO (40), MATH | Statements; some forks link proofs | varies | MIT or Apache | `imo_1959_p1`, `mathd_algebra_478` | [P] https://github.com/openai/miniF2F |
| miniF2F-v2 | 488 x 2 variants | same | Statements | Lean 4 | Apache-2.0 | same | [P] https://github.com/roozbeh-yz/miniF2F_v2 |
| MathOlympiadBench (Goedel) | 360 | IMO 158, ISL 131, regional 68 | Statements | Lean 4.9 | Apache-2.0 | `Usa2005P2` | [P] https://huggingface.co/datasets/Goedel-LM/MathOlympiadBench |
| Seed-Prover imo_all | 155 | past IMO | Proofs in imo.zip | v4.14.0 | Apache-2.0 | `imo1959_p1_numbertheory` | [P] https://github.com/ByteDance-Seed/Seed-Prover |
| NuminaMath-LEAN | 104,155 | IMO 813, HMMT 1,330, USAMO 72, others | 9,475 complete | Lean 4 | Apache-2.0 | `algebra_4013` (no problem number) | [P] https://huggingface.co/datasets/project-numina/NuminaMath-LEAN |
| CombiBench | 100 | 36 IMO combinatorics | Statements | Lean >= 4.15 | MIT | `imo_2000_p...` style | [P] https://huggingface.co/datasets/project-numina/CombiBench |
| FormalMATH | 5,560 (Lite 425) | olympiad-style (Omni, AIME, etc.) | Statements | Lean 4 | MIT | numeric ids | [P] https://huggingface.co/datasets/SphereLab/FormalMATH-All |
| IMO 2026 formal sets | 6 problems, several sets | IMO 2026 | Yes | v4.29 to v4.31 | MIT or Apache | `IMO2026/Q1` | [P] https://github.com/AxiomMath/IMO2026 |

---

## 2. Lean 4 olympiad datasets in detail

### 2.1 miniF2F family

miniF2F is the oldest benchmark that everyone uses. It has many forks, and each fork fixes a different set of errors.

| Variant | URL | Size | Language and Lean | Licence | Notes | Src |
|---|---|---|---|---|---|---|
| Original (OpenAI) | https://github.com/openai/miniF2F | 488: 244 valid, 244 test | Lean 3, Metamath, Isabelle, HOL Light (165 per split) | MIT (Metamath), Apache (Lean, Isabelle), FreeBSD (HOL Light) | Archived. Sources are AMC, AIME, IMO (20 per split) and MATH. | [P] repo; [P] https://arxiv.org/abs/2109.00110 |
| facebookresearch/miniF2F | https://github.com/facebookresearch/miniF2F | 488 | Lean, Isabelle, Metamath, HOL Light, plus `informal/` | MIT | Adds fixes. Adds informal statements and proofs from Draft, Sketch, and Prove. | [P] repo |
| xqyww123/miniF2F | https://github.com/xqyww123/miniF2F | 488 | Isabelle | MIT | Corrected Isabelle version. Work in progress. Updated 2026-08. | [P] repo |
| google-deepmind/miniF2F | https://github.com/google-deepmind/miniF2F | 488 minus removed false items | Lean 4 v4.27.0 | Apache-2.0 | Ported with mathport. Has docstrings. False statements removed. AlphaProof was evaluated on this version. | [P] repo |
| yangky11/miniF2F-lean4 | https://github.com/yangky11/miniF2F-lean4 | 488 | Lean 4, release v4.24.0 | MIT | Widely used. Not maintained. | [P] repo |
| justincasher/miniF2F (Harmonic) | https://github.com/justincasher/miniF2F | 488, original splits | Lean 4 | MIT | Harmonic's cleaned copy. Updated 2025-05. The old harmonic-ai/datasets URL now returns 404. Harmonic had also published a 392/48/48 re-split. | [P] repo; 404 checked on https://github.com/harmonic-ai/datasets |
| project-numina/minif2f_test | https://huggingface.co/datasets/project-numina/minif2f_test | 244 | Lean 4 | Apache-2.0 | Kimina's test split. Kimina reports at least 5 wrongly formalized problems. | [P] card |
| DeepSeek V1.5 minif2f.jsonl | https://github.com/deepseek-ai/DeepSeek-Prover-V1.5/tree/main/datasets | 488 | Lean 4.9.0 | MIT | Issue #26 (2026-09) reports statement defects. | [P] repo; [P] https://github.com/deepseek-ai/DeepSeek-Prover-V1.5/issues/26 |
| miniF2F-v2 (v2s, v2c) | https://github.com/roozbeh-yz/miniF2F_v2 | 488 each | Lean 4 | Apache-2.0 | v2c keeps all multiple-choice options. v2s is simplified. Over 300 statements corrected. | [P] repo; [P] https://arxiv.org/abs/2511.03108 |
| miniF2F_v2-fix | https://github.com/FinParker/miniF2F_v2-fix | 488 | Lean 4 | UNVERIFIED | Further fixes in `LOCAL_CORRECTIONS.json`. | [P] repo |

Audits of miniF2F:
- miniF2F-v2 (NeurIPS 2025) found a formal/informal mismatch in more than half of the original problems. Exact percentage UNVERIFIED. [P] https://arxiv.org/abs/2511.03108
- On v2, the full pipeline (autoformalize, then prove) went from about 40% to about 70%. [P] https://arxiv.org/abs/2511.03108
- Human-judged autoformalization accuracy on v1, v2s and v2c: Kimina 90.6%, 91.0% and 75.4%; Herald 62.7%, 66.0% and 54.1%. Automated judges score higher than humans do. [P] https://arxiv.org/abs/2511.03108
- The "Faults" audit (section 6.1) counted findings in each fork. These are checker findings, not confirmed errors. Confirmed means proven defective. [P] https://arxiv.org/abs/2606.29493

| miniF2F variant | Findings | Proven defective | Src |
|---|---|---|---|
| v2c | 193 of 488 | 8 | [P] https://arxiv.org/abs/2606.29493 |
| v2s | 181 of 488 | UNVERIFIED | [P] same |
| yangky11 | 135 of 488 | 6 | [P] same |
| yangky11 (early) | 79 | UNVERIFIED | [P] same |
| justincasher | 74 | 0 | [P] same |
| harmonic | 74 | 0 | [P] same |
| ai-mo (Numina) | 37 of 244 | UNVERIFIED | [P] same |

Id mapping. miniF2F names encode the source. `imo_YYYY_pN` maps to `imo/YYYY/pN`. `aime_YYYY_pN` and `amc12a_YYYY_pN` map the same way. `mathd_*` and `algebra_*` are MATH-dataset items, not competition problems. [P] https://github.com/openai/miniF2F

### 2.2 PutnamBench

| Field | Value | Src |
|---|---|---|
| URL | https://github.com/trishullab/PutnamBench | [P] |
| Size | Lean 672, Isabelle 640, Coq 412 (1,724 total) | [P] repo README |
| Coverage | Putnam 1962 to 2025 | [P] repo |
| Proofs | Statements only | [P] repo |
| Lean pin | v4.33.1 | [P] `lean4/lean-toolchain` |
| Licence | Apache-2.0 (Lean, Isabelle), MIT (Coq) | [P] repo |
| Informal text | Included with permission from the MAA | [P] repo README |
| Answer problems | 346 of the 672 Lean problems use `abbrev putnam_YYYY_xN_solution : T := sorry` | [P] repo |
| Leaderboard tracks | with-answer and no-answer | [P] https://trishullab.github.io/PutnamBench/leaderboard.html |
| Id | `putnam_1988_b1` maps to `putnam/1988/b1` | [P] repo |

PutnamBench leaderboard (Lean, out of 672 unless noted). Source for all rows: [P] https://github.com/trishullab/PutnamBench/blob/main/docs/results.json

| System | Score | Date or notes |
|---|---|---|
| DeepSeek-Prover-V2 | 47 | 2025-06 |
| Goedel-Prover-V2 | 86 | 2025-07 |
| Ax-Prover | 91 | |
| C2C-Goedel | 110 | |
| Seed-Prover | 329 | |
| AxProverBase | 365 | Opus 4.5, v4.29 |
| Hilbert | 462 | |
| Aleph | 500, then 637, 668, then 672 | 637 in 2025-12; 668 on 2026-01-11; 672 on 2026-08-26 (v4.27, average $74 per problem) |
| Seed-Prover 1.5 | 581 | v4.22 |
| Leanstral 1.5 | 588 | open model |
| Goedel-Architect | 597, then 642 | v4.26 |
| Mistral Prover | 658 | |
| Humanfia | 672 | 2026-08-27, GPT-5.6 Sol, v4.27 |
| NEAR AI (DeepSeek V4 Flash) | 672 | mean $0.17; two problems were run on fixed formalizations |
| Midas Prover | 672 | no-answer mode, 2026-09-12, average $6.17 |
| Forall (Astrio) | 672 | Claude Opus 5, mean $3.52, 2026-09-24 |
| Varto | Isabelle 640 of 640 | GPT-5.5; 130 problems needed formalization fixes |
| Fable 5.1 Max | 8 (Rocq) | 2026-10-05; 3 of them on corrected formalizations |
| TIR Conjecturor | 15 | no-answer |
| Enumerate-Conjecture-Prove | 17 | no-answer |

Key point: the Lean track is saturated (several 672/672 results). Several top results needed fixed statements. That is direct evidence that the benchmark statements still have defects. [P] same file

### 2.3 Compfiles

| Field | Value | Src |
|---|---|---|
| URL | https://github.com/dwrensha/compfiles | [P] |
| Dashboard | https://dwrensha.github.io/compfiles | [P] |
| Size | 521 formalized, 515 solved | [P] dashboard |
| IMO | 404 problems in total, 274 formalized, 270 solved | [P] dashboard |
| USAMO | 306 in total, 217 formalized, 216 solved | [P] dashboard |
| By topic | Geometry 19 formalized / 13 solved; Algebra 198; Number theory 146; Combinatorics 165; Inequality 40 | [P] dashboard |
| Lean pin | v4.35.0, the same as math2 | [P] `lean-toolchain` |
| Licence | Apache-2.0 | [P] repo |
| Answer style | `determine` marks an answer to be found; `problem` marks the theorem; `snip begin/end` hides proof-only material | [P] repo README |
| Extraction | `lake exe extractProblems` writes `_extracted/problems` and `_extracted/solutions` | [P] repo README |
| Id | `Imo2022P2` maps to `imo/2022/p2`; `Usa1999P1` maps to `usamo/1999/p1` | [P] repo |

Compfiles is the best single source to import. It is on the same toolchain as math2. It is human-written and has proofs. It also separates the problem statement from the solution.

### 2.4 Mathlib Archive/Imo

| Field | Value | Src |
|---|---|---|
| URL | https://github.com/leanprover-community/mathlib4/tree/master/Archive/Imo | [P] |
| Size | 55 problem files plus a README | [P] tree listing |
| Coverage | Imo1959Q1 through Imo2026Q5, including Imo2024Q1, Q2, Q3, Q5, Q6 and Imo2025Q3 | [P] tree listing |
| Proofs | Yes, all full proofs, checked in Mathlib CI | [P] |
| Licence | Apache-2.0 | [P] |
| Id | `Imo2024Q1` maps to `imo/2024/p1` | [P] |

### 2.5 formal-imo (Google DeepMind)

| Field | Value | Src |
|---|---|---|
| URL | https://github.com/google-deepmind/formal-imo | [P] |
| Size | 258 files, `Imo/YYYY/PN.lean` | [P] repo tree |
| Coverage | IMO 1959 to 2024 | [P] |
| Proofs | No. Statements end in `sorry`. | [P] |
| Lean pin | v4.27.0 | [P] `lean-toolchain` |
| Licence | Apache-2.0 | [P] |
| Names | `theorem imo_YYYY_pN`; parts as `imo_YYYY_pN.parts.i` | [P] |
| Answers | `answer(...)` elaborator | [P] |
| Topic tag | `@[imo_problem_subject algebra]` | [P] |
| Docstrings | From AoPS and Kalva | [P] |
| Provenance | Google's own formalizations, adapted miniF2F and adapted Mathlib Archive | [P] README |

Example [P] (repo, Imo/2010/P1.lean):

```lean
theorem imo_2010_p1 :
    {f : ℝ → ℝ | ∀ x y, f (⌊x⌋ * y) = f x * ⌊f y⌋} =
      answer({0} ∪ {fun x => c | c ∈ Set.Ico (1 : ℝ) (2 : ℝ)}) := by sorry
```

The sibling repo formal-conjectures (Apache-2.0, v4.33.1) deprecated `answer(...)` on 2026-10-15. The replacement is `@[question]` and `@[question value]`. The unknown becomes a separate `def <thm>.answer : T := sorry`. Their CONTRIBUTING.md says a trivial answer, such as the set itself, is out of scope. [P] https://github.com/google-deepmind/formal-conjectures/blob/main/CONTRIBUTING.md

### 2.6 IMO Shortlist: IMOSLLean4

| Field | Value | Src |
|---|---|---|
| URL | https://github.com/mortarsanjaya/IMOSLLean4 | [P] |
| Size | 171 problems | [P] repo tree |
| Coverage | ISL 2006 to 2025, mostly Algebra and Number theory, some Combinatorics | [P] |
| Proofs | Yes, all | [P] |
| Lean pin | Lean and Mathlib v4.35.0, the same as math2 | [P] `lean-toolchain` |
| Licence | Apache-2.0 | [P] |
| Id | `IMOSLLean4/Main/IMO2019/A1.lean`, theorem `final_solution`; maps to `isl/2019/a1` | [P] |

Note that the theorem name is always `final_solution`. The file path is the only id. ISL problems that became IMO problems need a cross-link.

### 2.7 Other olympiad statement sets

| Dataset | URL | Size | Coverage | Proofs | Lean | Licence | Id | Src |
|---|---|---|---|---|---|---|---|---|
| MathOlympiadBench (Goedel) | https://huggingface.co/datasets/Goedel-LM/MathOlympiadBench | 360 | IMO 158 (1959 to 2024), ISL 131 (2006 to 2023), regional 68, puzzles 3 | Statements; human-verified | Lean 4.9 | Apache-2.0 | `Usa2005P2` | [P] card |
| Seed-Prover imo_all.lean | https://github.com/ByteDance-Seed/Seed-Prover/tree/main/SeedProver | 155 | past IMO | Proofs in `imo.zip` | v4.14.0 | Apache-2.0 | `imo1959_p1_numbertheory`, `imo1959_p2a_algebra` | [P] repo |
| CombiBench | https://huggingface.co/datasets/project-numina/CombiBench | 100 (plus `test_with_solution`) | 42 Brualdi textbook, 36 IMO combinatorics (all since 2000 except 2004 P3), 12 other contests, 10 hackmath | Statements | Lean >= 4.15 | MIT | `hackmath_1` etc. | [P] card; [P] https://github.com/MoonshotAI/CombiBench |
| FIMO | https://arxiv.org/abs/2309.04295 | ISL problems | IMO Shortlist | Statements | Lean 3 | UNVERIFIED | UNVERIFIED | [P] paper |
| Lean dataset for IMO | https://arxiv.org/abs/2411.18872 | 20 | IMO | Yes | Lean 4 | UNVERIFIED | UNVERIFIED | [P] paper |
| FMC | https://arxiv.org/abs/2507.11275 | 6,980 | olympiad-level | Statements | Lean 4 | UNVERIFIED | UNVERIFIED | [P] paper |
| Lean-IMO-Bench (LEAP) | https://arxiv.org/abs/2606.03303 | 60 | IMO geometry and others | UNVERIFIED | Lean 4 | UNVERIFIED | UNVERIFIED | [P] paper |

CombiBench answers use `abbrev hackmath_1_solution : ℕ := sorry`. 45% of its problems need an answer. Kimina-Prover solved 7 of 100. DeepSeek-Prover-V2 solved 12 of 77. [P] https://huggingface.co/datasets/project-numina/CombiBench

---

## 3. Large training and benchmark sets (mostly autoformalized)

These are large. Most entries are machine-translated, and most have no competition id. They are useful as a pool of candidate statements, not as canonical statements.

| Dataset | URL | Size | Content | Proofs | Lean | Licence | Fidelity info | Src |
|---|---|---|---|---|---|---|---|---|
| NuminaMath-LEAN | https://huggingface.co/datasets/project-numina/NuminaMath-LEAN | 104,155 | olympiad and contest | `ground_truth_type`: complete 9,475; statement 8,726; with_sorry 1,905 | Lean 4 | Apache-2.0 | 84,003 by autoformalizer, 20,152 by humans | [P] card |
| Kimina-Prover-Promptset | https://huggingface.co/datasets/AI-MO/Kimina-Prover-Promptset | 24,418 | contest | No | Lean 4 | UNVERIFIED | | [P] card |
| Lean Workbook | https://huggingface.co/datasets/internlm/Lean-Workbook | 57,231 (Workbook), 82,893 (Plus) | contest-style | default config: 25,214 rows proved or disproved (18,985 proved, 6,229 disproved) | v4.8.0-rc1 | Apache-2.0 | 93.5% accuracy on a manually checked sample; possible leakage from miniF2F and ProofNet | [P] card; [P] https://github.com/InternLM/InternLM-Math |
| Goedel-Pset-v1 | https://huggingface.co/datasets/Goedel-LM/Goedel-Pset-v1 | 1,732,594 rows | autoformalized contest problems | No | Lean 4.9 | no licence tag | LLM fidelity checks. Paper says 1.64M autoformalized, 1.78M with Lean Workbook. | [P] card; [P] https://arxiv.org/abs/2502.07640 |
| Goedel Lean-workbook-proofs | https://huggingface.co/datasets/Goedel-LM/Lean-workbook-proofs | 29,750 | Lean Workbook | Yes | Lean 4.9 | UNVERIFIED | | [P] card |
| Goedel SFT_dataset_v2 / RL_dataset_V2 | https://huggingface.co/Goedel-LM | 1,745,010 / 98,632 | training | Yes / No | Lean 4 | Apache-2.0 (RL) | | [P] cards |
| FormalMATH | https://huggingface.co/datasets/SphereLab/FormalMATH-All | 5,560 | olympid 2,409; omni 1,210; blue_prove 703; blue_quant 396; aime 371; u-math 358; hardmath 67; DEMIMATH 46 | No | Lean 4 | MIT | 72.09% correct before manual review; 12 IMO medalists reviewed | [P] card; [P] https://github.com/Sphere-AI-Lab/FormalMATH-Bench |
| FormalMATH-Lite | https://huggingface.co/datasets/SphereLab/FormalMATH-Lite | 425 | 359 high school, 66 undergraduate | No | Lean 4 | MIT | | [P] card |
| DeepSeek-ProverBench | https://huggingface.co/datasets/deepseek-ai/DeepSeek-ProverBench | 325 | 15 AIME 2024/25, rest textbooks | No | Lean 4 | no licence tag | | [P] card |
| DeepSeek-Prover-V1 | https://huggingface.co/datasets/deepseek-ai/DeepSeek-Prover-V1 | 27,503 | synthetic | Yes | Lean 4 | DeepSeek licence | no informal text; names `thm_0` | [P] card |
| STP | https://huggingface.co/datasets/kfdong/STP_Lean_0320 | 3,262,558 | self-play conjectures | Yes | Lean 4 | MIT | | [P] card |
| project-numina/olympiads | https://huggingface.co/datasets/project-numina/olympiads | 13,088 | informal olympiad | n/a | n/a | UNVERIFIED | | [P] card |
| AdvancedMathBench | https://huggingface.co/datasets/internlm/AdvancedMathBench | 245 proverbench + 888 verifierbench | advanced | | Lean 4 | UNVERIFIED | 2026 | [P] card |

NuminaMath-LEAN mapping. It has fields uuid, problem, answer, source, exam, formal_statement, formal_ground_truth and ground_truth_type. The `exam` field gives the contest: IMO 813, HMMT 1,330, BalticWay 281, JBMO 185, USAMO 72, APMO 51, EGMO 20, and others. There is no year or problem-number field. Mapping to `imo/YYYY/pN` needs text matching against the informal corpus. [P] https://huggingface.co/datasets/project-numina/NuminaMath-LEAN

Proof bundles you can import as "known proofs" (check them again on our toolchain):

| Bundle | URL | Content | Src |
|---|---|---|---|
| DeepSeek-Prover-V2 | https://github.com/deepseek-ai/DeepSeek-Prover-V2 | `minif2f-solutions.zip`; 88.9% on miniF2F; 49 of 658 on PutnamBench | [P] repo |
| Kimina-Prover-Preview | https://github.com/MoonshotAI/Kimina-Prover-Preview | `minif2f_test_solved.zip`, `minif2f_test_solved_filtered_0710.zip` | [P] repo |
| Seed-Prover | https://github.com/ByteDance-Seed/Seed-Prover | `imo.zip` (past IMO), `MiniF2F.zip`, miniCTX-v2, `Putnam2025.zip` | [P] repo |
| Goedel | https://huggingface.co/datasets/Goedel-LM/Lean-workbook-proofs | 29,750 | [P] card |

---

## 4. Non-olympiad Lean benchmarks (for context)

| Dataset | URL | Size | Level | Lean | Licence | Src |
|---|---|---|---|---|---|---|
| ProofNet | https://huggingface.co/datasets/hoskinson-center/proofnet | 371 (185 valid, 186 test) | undergraduate | Lean 3 | MIT | [P] card |
| ProofNet# | https://huggingface.co/datasets/PAug/ProofNetSharp | 371 | undergraduate | Lean 4 | MIT | [P] card; ids like `Artin\|exercise_2_2_9` |
| miniCTX / v2 | https://huggingface.co/datasets/l3lab/miniCTX | 662 / 668 | real projects | Lean 4 | Apache-2.0 | [P] card |
| FATE (M, H, X) | https://github.com/frenzymath/FATE | 150 / 100 / 100 | abstract algebra | v4.16 (M, H), v4.19 (X) | MIT | [P] repo; [P] https://arxiv.org/abs/2511.02872 |
| Herald | https://huggingface.co/datasets/FrenzyMath/Herald_statements | 579,883 statements; 44,553 proofs | Mathlib | Lean 4 | Apache-2.0 | [P] card; [P] https://arxiv.org/abs/2410.10878 |

Herald claims 93.2% on miniF2F. Human evaluation in miniF2F-v2 gives 62.7%. [P] https://arxiv.org/abs/2511.03108

---

## 5. Autoformalization and equivalence datasets

| Dataset | URL | Size | Use | Licence | Src |
|---|---|---|---|---|---|
| MMA | https://arxiv.org/abs/2311.03755 ; mirror https://huggingface.co/datasets/casey-martin/multilingual-mathematical-autoformalization | 332,774 pairs | Isabelle AFP and Mathlib, informalized by GPT-4 | UNVERIFIED | [P] paper; [S] mirror |
| FormL4 | https://arxiv.org/abs/2406.01940 | UNVERIFIED | Lean 4 autoformalization | UNVERIFIED | [P] paper (details UNVERIFIED) |
| StepFun-Formalizer training | https://huggingface.co/datasets/stepfun-ai/StepFun-Formalizer-Training | 100K to 1M rows | autoformalizer training, BEq as RL reward | Apache-2.0 | [P] card; [P] https://arxiv.org/abs/2508.04440 |
| ProofNetVerif | https://github.com/augustepoiroux/LeanInteract (UNVERIFIED that this is the paper repo) | 3,752 annotated pairs | judging formal/informal match | MIT | [P] Poiroux et al., EMNLP 2025 |
| RLM25 | same paper | 619 pairs | research-level matching | MIT | [P] same |
| FormalRx test set | https://arxiv.org/abs/2607.04655 | 7,030 items | error classification | UNVERIFIED | [P] paper |
| CriticLean | https://arxiv.org/abs/2507.06181 | UNVERIFIED | critic for semantic fidelity | UNVERIFIED | [P] paper |

CriticLean reports that Kimina-Autoformalizer-7B was right 38% of the time in one pass on 50 Omni-MATH problems. [P] https://arxiv.org/abs/2507.06181

---

## 6. Inequalities and geometry

### 6.1 Inequalities

| Dataset | URL | Size | Formal? | Lean | Licence | Src |
|---|---|---|---|---|---|---|
| Ineq-Comp | https://github.com/haoyuzhao123/LeanIneqComp | 75 seed problems, comp2_100, real 50 | Yes | 4.9 and 4.18 | Apache-2.0 | [P] repo; [P] https://arxiv.org/abs/2505.12680 (NeurIPS 2025) |
| IneqMath | https://huggingface.co/datasets/AI4Math/IneqMath | 1,252 train / 200 test | No, informal only | n/a | UNVERIFIED | [P] https://arxiv.org/abs/2506.07927 |

Under step-wise judging, o1 scores only 8% on IneqMath. [P] https://arxiv.org/abs/2506.07927

### 6.2 Geometry

Geometry is the weakest area for Lean. Compfiles has only 19 geometry problems formalized and 13 solved. [P] https://dwrensha.github.io/compfiles

| System or dataset | URL | Size | Language | Licence | Notes | Src |
|---|---|---|---|---|---|---|
| AlphaGeometry IMO-AG-30 | https://github.com/google-deepmind/alphageometry | 30 (`imo_ag_30.txt`) | AG DSL | Apache-2.0 | ids like `translated_imo_2000_p1`. AG1 solved 25; Wu plus AG solved 27. | [P] repo |
| AlphaGeometry JGEX-AG-231 | same | 231 (`jgex_ag_231.txt`) | AG DSL | Apache-2.0 | | [P] repo |
| AlphaGeometry 2 IMO-AG-50 | https://arxiv.org/abs/2502.03544 | 50 (all 45 IMO geometry problems 2000 to 2024) | AG DSL | n/a | AG2 solved 42 of 50 | [P] paper |
| Newclid | https://github.com/LMCRC/Newclid | UNVERIFIED | Python DSL | no licence field | | [P] repo |
| GenesisGeo | UNVERIFIED | IMO-95, HAGeo-409 | DSL | UNVERIFIED | | [S] UNVERIFIED |
| Yuclid (Harmonic) | https://github.com/harmonic-ai (UNVERIFIED path) | solves 17 of AG-30 | | Apache-2.0 | Used by Aristotle for IMO 2025 P2 | [P] https://arxiv.org/abs/2510.01346 |
| LeanGeo | https://github.com/project-numina/LeanGeo | 122-problem bench (GeometryLeanBench); 43 solutions, 2 IMO | Lean 4 v4.15.0 | MIT | Built on SystemE and LeanEuclid | [P] repo; [P] https://arxiv.org/abs/2508.14644 |
| LeanEuclid | https://github.com/loganrjmurphy/LeanEuclid | Euclid's Elements, UniGeo | Lean 4 v4.19.0 | MIT | | [P] repo |
| MechGeo | https://arxiv.org/abs/2608.02295 | UNVERIFIED | | UNVERIFIED | | [P] paper |

Geometry ids map well: `translated_imo_2000_p1` maps to `imo/2000/p1`.

---

## 7. Other proof assistants

| System | Resource | URL | Content | Licence | Src |
|---|---|---|---|---|---|
| Isabelle | miniF2F Isabelle | https://github.com/openai/miniF2F ; corrected https://github.com/xqyww123/miniF2F | 488 statements | Apache / MIT | [P] |
| Isabelle | PutnamBench Isabelle | https://github.com/trishullab/PutnamBench | 640 | Apache-2.0 | [P] |
| Isabelle | AFP entry IMO2019 (Eberl) | https://www.isa-afp.org/entries/IMO2019.html | IMO 2019 Q1, Q4, Q5 with proofs | BSD (AFP default) | [P] |
| Isabelle | filipmaric/IMO | https://github.com/filipmaric/IMO | 37 `.thy` files, ISL solutions | MIT | [P] repo; [P] https://arxiv.org/abs/2010.16015 |
| Coq/Rocq | PutnamBench Coq | https://github.com/trishullab/PutnamBench | 412 | MIT | [P] |
| Coq/Rocq | dedicated olympiad repo | none found | | | search, Oct 2026 |
| HOL Light | miniF2F HOL Light | https://github.com/openai/miniF2F | partial, 165 per split | FreeBSD | [P] |
| Metamath | set.mm | https://us.metamath.org/mpeuni/imo72b2.html | only `imo72b2` (IMO 1972 B2, Polu, 2020) | CC0 | [P] |
| Metamath | miniF2F Metamath | https://github.com/openai/miniF2F | 488 statements | MIT | [P] |

Conclusion: outside Lean, the only sizeable olympiad sets are PutnamBench (Isabelle, Coq) and miniF2F (Isabelle, Metamath, HOL Light). Lean 4 should be the only first-class target. Others can be stored as linked external statements.

---

## 8. AI provers and contest results

### 8.1 IMO 2024: AlphaProof

| Fact | Src |
|---|---|
| Published in Nature 651, 607 to 613 (2026), DOI 10.1038/s41586-025-09833-y, online 2025-11-12 | [P] https://www.nature.com/articles/s41586-025-09833-y |
| Score 28 of 42: P1, P2, P6 by AlphaProof; P4 by AlphaGeometry 2 | [P] https://deepmind.google/discover/blog/ai-solves-imo-problems-at-silver-medal-level/ |
| Statements were formalized by humans | [P] same |
| Graded by Timothy Gowers and Joseph Myers | [P] same |
| Proofs | [P] https://storage.googleapis.com/deepmind-media/DeepMind.com/Blog/imo-2024-solutions/index.html ; mirror [S] https://github.com/James-Oswald/alphaproof-outputs-mirror |
| Trained on about 1M informal problems, giving about 80M formal statements | [P] Nature paper |
| Related results repo, Apache-2.0, v4.27.0 | [P] https://github.com/google-deepmind/alphaproof-nexus-results |

### 8.2 IMO 2025

| System | Result | Formal artefacts | Lean | Licence | Src |
|---|---|---|---|---|---|
| Seed-Prover (ByteDance) | P1 (after the contest), P3, P4, P5 in Lean; P2 by Seed-Geometry; officially scored 2/7/7/7/7/0 = 30, silver | https://github.com/ByteDance-Seed/Seed-Prover | v4.14.0 | Apache-2.0 | [P] repo |
| Seed-Prover 1.5 | P1 to P5 (hours: 16.5 / 0.01 / 5 / 8 / 1) | same | v4.22.0 | Apache-2.0 | [P] https://arxiv.org/abs/2512.17260 |
| Harmonic Aristotle | P1, P3, P4, P5 in Lean, gold-level; P2 in text (`P2.txt`) | https://github.com/harmonic-ai/IMO2025 | v4.20.0-rc5 | no licence file | [P] repo; [P] https://arxiv.org/abs/2510.01346 |

Aristotle details [P] https://arxiv.org/abs/2510.01346:
- Humans translated the statements. This was the only human formalization step.
- An LLM proposed the answers, such as c = 4 for P3.
- The repo has `StatementOnly_` files for P1 and P3 to P6, and a second P4 proof (`P4_solve2`).

### 8.3 Putnam 2025

Seed-Prover 1.5 proved 11 of 12 Putnam 2025 problems on v4.22.0 (`Putnam2025.zip`). It also reports 88% on PutnamBench, FATE-H 80% and FATE-X 33%. [P] https://arxiv.org/abs/2512.17260

Seed-Prover 1.0 claims: miniF2F valid 100% and test 99%, past IMO 79%, CombiBench 30%, MiniCTX-v2 82%. [P] https://github.com/ByteDance-Seed/Seed-Prover

### 8.4 IMO 2026 (Shanghai, 15 and 16 July 2026)

| Repo | Who wrote the statements | Proofs | Lean | Licence | Src |
|---|---|---|---|---|---|
| https://github.com/AxiomMath/IMO2026 | AxiomProver (autoformalized) | 6 of 6, 42/42. Lines and minutes: Q1 521 / 24; Q2 1224 / 360; Q3 4229 / 869; Q4 520 / 39; Q5 457 / 65; Q6 771 / 139. Checked with Axle `verify_proof`. | v4.31.0 | MIT | [P] repo |
| https://github.com/pramaana-labs/imo2026-lean | Panini autoformalizer, then human-checked | Hardy proved all 6 for $148.64 total. It also proved Axiom's statements. Checked with Comparator. | v4.31.0 | MIT | [P] repo |
| https://github.com/math-ai-org/IMO2026 | UNVERIFIED | 6 of 6 | v4.30.0-rc2 | Apache-2.0 | [P] repo |
| https://github.com/midas-intelligence/IMO2026 | UNVERIFIED | Midas Prover 1.2, 6 of 6, 590 minutes | UNVERIFIED | UNVERIFIED | [P] repo |
| https://github.com/MechMath/IMO2026 (EonMath) | Axiom's statements | 6 of 6 | v4.29.0 | UNVERIFIED | [P] repo |
| https://github.com/humanfia/hoa-qed/tree/main/imo2026 (NVIDIA Research) | UNVERIFIED | UNVERIFIED | UNVERIFIED | UNVERIFIED | [P] repo |
| https://github.com/studio-dots-ai/dots-IMO2026 | UNVERIFIED | UNVERIFIED | UNVERIFIED | UNVERIFIED | [P] repo |
| Mathlib Archive `Imo2026Q5.lean` | human | full proof | Mathlib master | Apache-2.0 | [P] https://github.com/leanprover-community/mathlib4/tree/master/Archive/Imo |

Natural-language AI results at IMO 2026 [S] https://www.digitalapplied.com (IMO 2026 AI roundup):
- Celia (Huawei) and dots-note-3.0 (Xiaohongshu) scored 42 under official grading.
- Nemotron 3 Ultra scored 30, graded by the IMO team. It has open weights.
- Fable 5, GPT-5.6 Sol and Kimi K3 ran self-administered tests graded by Claude agents. Their scores are not official. UNVERIFIED.
- The gold cutoff was 29.

Takeaway for olymmath: in 2026, several groups published independent Lean statements of the same IMO problems. Some statements were human-checked and some were not. Several provers proved other groups' statements. This is exactly the "many formal statements per problem" situation that the data model must support.

### 8.5 IMO Grand Challenge

| Fact | Src |
|---|---|
| Format is formal-to-formal: the AI gets a Lean statement and must output a Lean proof | [P] https://imo-grand-challenge.github.io |
| The kernel must check the proof within 10 minutes. No partial credit. The AI must be open source. | [P] same |
| Who formalizes the statements was never decided. Last activity in 2022. Apache-2.0. | [P] same; [P] https://github.com/IMO-grand-challenge |

---

## 9. Best practice for statement fidelity

### 9.1 How bad is the problem?

| Finding | Number | Src |
|---|---|---|
| Faults audit: findings across 10 benchmarks | 4,833 findings over 10,318 problems; 398 certified (proven) defects | [P] https://arxiv.org/abs/2606.29493 |
| FormalMATH | 3,250 findings of 5,560; 141 proven | [P] same |
| FormalMATH-Lite | 213 of 425 | [P] same |
| ProverBench | 370 findings over 325 problems; 208 proven | [P] same |
| ProofNet | 82 of 371 | [P] same |
| ProofNet# | 67 | [P] same |
| CombiBench | 78 of 100 | [P] same |
| miniF2F | over half had a mismatch | [P] https://arxiv.org/abs/2511.03108 |
| FormalMATH before human review | 72.09% correct | [P] https://huggingface.co/datasets/SphereLab/FormalMATH-All |
| FormalMATH compiled statements rejected on semantics | 60.7% (as cited by Lean-GAP) | [S] Lean-GAP, citing FormalMATH (UNVERIFIED primary) |
| Compile success vs. semantic success | nearly half of the compile gain fails the semantic check | [P] https://arxiv.org/abs/2606.31002 |
| Lean Workbook sample | 93.5% accurate | [P] https://huggingface.co/datasets/internlm/Lean-Workbook |
| PutnamBench top runs | several needed fixed statements; Varto needed 130 Isabelle fixes | [P] https://github.com/trishullab/PutnamBench/blob/main/docs/results.json |

### 9.2 Misformalization taxonomy

The Faults paper (Ammanamanchi, Bhat and Biderman, ICML 2026) is the most complete taxonomy. [P] https://arxiv.org/abs/2606.29493

| Class | Sub-class | Example |
|---|---|---|
| Fidelity | Specification error | Missing hypothesis; only part of the problem translated |
| Fidelity | Formalization error | Wrong domain or type (ℕ instead of ℤ); a definition that does not match; vacuous hypotheses |
| Fidelity | Lean encoding hazard | Natural-number subtraction truncates; `x / 0 = 0`; integer division |
| Evaluation loophole | | Extra axioms; `native_decide`; unsafe tactics; the `apply?` bug before Lean 4.20 |
| Maintenance | | Version drift; too many forks; defects in the natural-language statement itself |

FormalRx uses four classes: logical structure, mathematical object, constraint violation and Lean-specific errors. [P] https://arxiv.org/abs/2607.04655

Static checkers from Faults (code at https://github.com/Shashi456/atp-checkers, MIT, Lean v4.30.0-rc2) [P]:

| Checker | Catches |
|---|---|
| Counterexample | statement is false for a small case |
| Vacuous | hypotheses cannot all hold |
| Unsound Axiom | file declares an axiom |
| DivZero | division where the divisor can be 0 |
| NatSub | subtraction on ℕ that can truncate |
| Analytic Domain | `Real.log`, `Real.sqrt`, etc. outside their domain |
| Unused Binder | a variable that is never used (often a lost hypothesis) |
| Integer Division | `/` on ℕ or ℤ where ℚ or ℝ was meant |
| Zero-Indexed Range | `Finset.range n` used for 1..n |

Release standards proposed by Faults [P]:
- Use `proof_wanted` instead of `sorry` for open statements.
- Set `autoImplicit` to false.
- Run the checkers before release.
- Allow no extra axioms.
- Pin the Lean version.
- Use an LLM judge only for triage, not as the final decision.

### 9.3 Equivalence checking

| Method | What it does | Numbers | Src |
|---|---|---|---|
| BEq (Liu et al., ICLR 2025) | Proves A implies B and B implies A, with restricted automation | Plain definitional equality: 100% precision, 11.43% recall | [P] https://openreview.net/forum?id=hUb2At2DsQ (UNVERIFIED exact URL) |
| BEq+ (Poiroux et al., EMNLP 2025) | Improved BEq; evaluated on ProofNetVerif and RLM25 | | [P] Poiroux et al., EMNLP 2025; repo URL UNVERIFIED |
| ITPEval | Fixed tactic cascade, deterministic | | [P] https://arxiv.org/abs/2607.19407 |
| Back-translation / round trip | Formal to informal, then compare with the original text | details UNVERIFIED | [P] https://arxiv.org/abs/2604.25031 |
| FormalAlign | Learned alignment score | | [P] ICLR 2025, UNVERIFIED URL |
| CriticLean | Critic model for semantic fidelity | | [P] https://arxiv.org/abs/2507.06181 |
| Two LLM judges | Use two judges from different model families | | [P] https://arxiv.org/abs/2606.31002 |

### 9.4 How answer-type problems are formalized

| Style | Used by | Form | Src |
|---|---|---|---|
| `abbrev ..._solution := sorry` | PutnamBench, CombiBench | `abbrev putnam_YYYY_xN_solution : T := sorry` with the answer in a comment | [P] https://github.com/trishullab/PutnamBench |
| `determine` | Compfiles | `determine solution_set : Set ℕ := ...` | [P] https://github.com/dwrensha/compfiles |
| `answer(...)` elaborator | formal-imo | inline in the theorem | [P] https://github.com/google-deepmind/formal-imo |
| `@[question value]` with `def <thm>.answer` | formal-conjectures, from 2026-10-15 | separate def for the unknown | [P] https://github.com/google-deepmind/formal-conjectures/blob/main/CONTRIBUTING.md |
| Answer in the statement | miniF2F, many training sets | the answer is baked in; the prover only proves it | [P] https://github.com/openai/miniF2F |
| FPS / D-FPS | FormalMath500, MiniF2F-Solving, PutnamBench-Solving | the prover must find the answer; graded with RPE | [P] https://arxiv.org/abs/2505.04528 ; https://github.com/Purewhite2019/formal_problem_solving_main |

The main risk is a trivial answer. An example is the answer "the set of all f that satisfy the equation", which is the problem restated. formal-conjectures says this is out of scope. FPS handles it with RPE (restricted propositional equivalence). RPE agrees with xVerify with kappa 0.9732. FPS reports that wrong answers cause most failures. [P] https://arxiv.org/abs/2505.04528

### 9.5 Proof checking in Lean 4.35.0

| Fact | Src |
|---|---|
| Lean v4.35.0 includes `lake check` and `lake comparator`. They replace the standalone leanprover/comparator. | [P] https://github.com/leanprover/lean4/releases |
| `lake comparator` checks that each solution theorem has the same statement as the challenge theorem, uses only permitted axioms, and is accepted by the kernel | [P] same |
| `--paranoid` adds external kernels: lean4lean, nanoda, con-leche, con-ron and leanchecker-paranoid | [P] same |
| Builds run in a bubblewrap sandbox, which is Linux only. Default config file is `comparator.json`. | [P] same |
| The axioms `Lean.reduceBool`, `Lean.reduceNat` and `Lean.trustCompiler` were removed. So `native_decide` can no longer slip in through them. | [P] same |
| Old comparator config: `{"challenge_module":"Challenge","solution_module":"Solution","theorem_names":[...],"permitted_axioms":["propext","Quot.sound","Classical.choice"]}` | [P] https://github.com/leanprover/comparator |

---

## 10. Mapping existing datasets to olymmath ids

Proposed canonical id: `<competition>/<year>/<problem>`, for example `imo/2024/p1`, `isl/2019/a1`, `putnam/1988/b1`, `usamo/1999/p1`.

| Source | Native id | Rule | Confidence |
|---|---|---|---|
| Compfiles | `Imo2022P2`, `Usa1999P1` | regex `([A-Z][a-z]+)(\d{4})P(\d+)`; table of competition codes | high |
| Mathlib Archive | `Imo2024Q1` | `Imo(\d{4})Q(\d+)` | high |
| formal-imo | `imo_2010_p1`, `.parts.i` | `imo_(\d{4})_p(\d+)`; parts become sub-statements | high |
| miniF2F | `imo_1959_p1`, `aime_1983_p1`, `amc12a_2019_p21` | prefix table; `mathd_*` is not a contest | high for contests |
| PutnamBench | `putnam_1988_b1` | `putnam_(\d{4})_([ab]\d)` | high |
| IMOSLLean4 | path `IMO2019/A1.lean` | from path | high |
| Seed imo_all | `imo1959_p2a_algebra` | `imo(\d{4})_p(\d+)([a-z]?)_` | high |
| MathOlympiadBench | `Usa2005P2` | Compfiles-style | high |
| AlphaGeometry | `translated_imo_2000_p1` | strip `translated_` | high |
| CombiBench | `imo_...`, `brualdi_...`, `hackmath_...` | prefix table | medium |
| NuminaMath-LEAN | `exam` field only | match the informal text against the olymmath corpus | low; needs review |
| FormalMATH, Lean Workbook, Goedel-Pset | numeric ids | text match only | low |
| IMO 2026 repos | `IMO2026/Q1` | from path | high |

---

## 11. Recommendation for olymmath and olymmath-agent

### 11.1 Data model

1. One informal problem has many formal statements. Never treat one statement as "the" formalization by default.
2. Each formal statement record has:
   - `problem_id` (for example `imo/2024/p1`) and an optional part (`a`, `b`).
   - `source` (dataset name, URL, commit hash, file path, native name).
   - `licence` (copied from the source).
   - `author_kind`: human, AI, or AI then human-reviewed.
   - `lean_toolchain` and `mathlib_rev` it was written for.
   - `answer_style` and `answer_value` (the answer kept as a separate field).
   - `review_status`: unreviewed, checker-clean, LLM-triaged, one-human, two-human canonical, or known-wrong.
   - `rank`, used to pick the statement shown on the site.
   - `equivalent_to`: links to other statements proven equivalent.
   - `proofs`: a list of proof records, each with its own check result.
3. Write answers in the formal-conjectures style: a `def <thm>.answer : T` plus the theorem that uses it. Store the answer value in a field. Reject trivial answers with an RPE-style check.

### 11.2 Fidelity pipeline (in order)

1. Compile on the pinned toolchain, with `autoImplicit` set to false.
2. Run the static checkers (atp-checkers, or a port of them).
3. Run a counterexample search on small cases.
4. Back-translate to English and compare with the informal statement using two LLM judges from different model families. Use this only for triage.
5. Try a BEq-style equivalence proof against the other statements for the same problem. A proven equivalence merges them into one class.
6. A statement becomes canonical only after two human reviewers sign off.
7. Keep defect reports. A statement marked known-wrong stays in the database with its reason, so it is not imported again.

### 11.3 Proof checking

1. Pin Lean v4.35.0 and Mathlib v4.35.0, the same as math2.
2. Check every proof with `lake comparator` against the frozen challenge statement. Use `lake check --paranoid` for proofs marked canonical. Run these on Linux, because the sandbox needs bubblewrap.
3. Allow only the axioms `propext`, `Quot.sound` and `Classical.choice`.
4. Allow no `sorry`. Reject `native_decide` unless the proof is flagged and stored apart.
5. When the toolchain is bumped, check every proof again. Record the toolchain each proof last passed on.

### 11.4 Import order

| Priority | Source | Why |
|---|---|---|
| 1 | Compfiles | human-written, has proofs, v4.35.0, Apache-2.0, clean ids |
| 2 | IMOSLLean4 | ISL with proofs, v4.35.0, Apache-2.0 |
| 3 | Mathlib Archive/Imo | human-written proofs, Apache-2.0 |
| 4 | formal-imo | full IMO statement coverage 1959 to 2024, Apache-2.0; port to v4.35.0 |
| 5 | PutnamBench | full Putnam coverage, Apache-2.0; answers as `abbrev` |
| 6 | IMO 2025 and 2026 repos | several independent statements for the same problems; good for equivalence tests |
| 7 | miniF2F v2 and the DeepMind fork | AMC and AIME coverage; use the corrected forks |
| 8 | MathOlympiadBench, CombiBench, Seed imo_all | extra variants, ranked low until reviewed |
| 9 | NuminaMath-LEAN, FormalMATH, Lean Workbook, Goedel-Pset | large pools of AI statements; import only after a text match to a known problem; never canonical without review |

Import rules:
- Pin each import to a commit hash.
- Copy the source licence onto each record. Skip sources without a licence (harmonic-ai/IMO2025, DeepSeek-ProverBench, Goedel-Pset-v1) or link to them only, until the licence is clear.
- Keep full provenance: URL, commit, path and native name.
- Port to v4.35.0 in a separate step. Record port changes as a diff against the original.
