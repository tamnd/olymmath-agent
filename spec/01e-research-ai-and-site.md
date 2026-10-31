# 01e. Research: AI solutions, grading and publishing

Research date: 2026-10-31. Scope: tamnd/olymmath (content) and tamnd/olymmath-agent (Go toolchain), site https://olymmath.tamnd.com.

Legend:
- **[P]** primary source (vendor docs, paper, official repo or blog).
- **[S]** secondary source (news, aggregator, third-party blog).
- **UNVERIFIED** means the claim was seen only in a search snippet or a weak source, and was not confirmed on a primary page.

Numbers were read on the cited pages on 2026-10-31. Re-check prices and limits before launch.

---

## Part 1. AI solutions and grading

### 1.1 Public corpora of model-written olympiad proofs with grades

| Corpus | What it contains | Grading | License | Source |
|---|---|---|---|---|
| **MathArena USAMO 2025 outputs** | 240 rows = 6 problems x 10 models x 4 runs. 23 columns: problem, model, full messages, answer, tokens, cost, `points_judge_1`, `points_judge_2`, per-rubric `grading_details`. Parquet, about 2.9 MB. | Two judges per solution, 0-7 per problem. In 2025 these were human experts (see "Proof or Bluff" below). | CC BY-NC-SA 4.0 (non-commercial, share-alike) | [P] https://huggingface.co/datasets/MathArena/usamo_2025_outputs |
| **MathArena other outputs** | `aime_2025_outputs`, `hmmt_feb_2025_outputs`, `cmimc_2025_outputs`, `apex_*_outputs`, `arxivmath-*_outputs`, `brokenarxiv-*`, `arxivlean-*`. Problem sets for AIME 2026, HMMT Feb 2026. | Final-answer sets are auto-checked. | Not shown on the org page; check each dataset card. | [P] https://huggingface.co/MathArena |
| **MathArena USAMO 2026 / IMO 2025 results** | Per-model raw outputs are browsable on the site (`/?comp=usamo--usamo_2026`). No separate HF dataset for these was confirmed. | See 1.3. | UNVERIFIED | [P] https://matharena.ai/usamo/ , https://matharena.ai/imo/ |
| **Open Proof Corpus (OPC)** | 1,010 problems from 20 competitions (IMO SL, USAMO, Putnam, MathArena, ...). 5,062 proofs from 6 LLMs. Splits: MathArena, PutnamBench, Best-of-n, Generic train/test. | Binary correct/incorrect. 13 expert graders (incl. ex-IMO). 1 or 2 judges per proof. Human-human agreement 90.4%. 43% of proofs correct. | Not stated on the site. UNVERIFIED. | [P] https://proofcorpus.ai/ ; [P] https://arxiv.org/abs/2506.21621 ; data https://huggingface.co/datasets/INSAIT-Institute/OPC |
| **IMO-Bench (Google DeepMind)** | IMO-AnswerBench 400 problems (`answerbench_v2.csv`). IMO-ProofBench 60 problems, 30 "Advanced" (`proofbench_v2.csv`). IMO-GradingBench 1,000 graded proofs (`gradingbench.csv`). IMO-LeanProofBench. | GradingBench uses 4 buckets mapped to IMO points: Correct 7, Almost 6, Partial 1, Incorrect 0. Balanced across buckets. ProofBench problems were altered by experts to reduce memorization. | Code Apache-2.0; other materials CC-BY 4.0 | [P] https://github.com/google-deepmind/superhuman/blob/main/imobench/README.md ; [P] https://imobench.github.io/ ; paper [P] https://arxiv.org/abs/2511.01846 |
| **ProofBench / ProofGrader** (Berkeley + GDM) | 145 problems from USAMO, IMO, Putnam, EGMO, APMO, TST (2022-2025). 435 proofs from Gemini-2.5-Pro, o3, DeepSeek-R1. | 0-7 scale. Problem-specific marking schemes (LLM-drafted, expert-refined). 5 experts. Mean score 2.38/7. | Check project page | [P] https://arxiv.org/abs/2510.13888 ; https://proofgrader.github.io/ |
| **OpenAI IMO 2025 proofs** | `problem_1.txt` .. `problem_5.txt` from the experimental model. | Graded by OpenAI's own team of three former IMO medalists; not certified by the IMO. 35/42. | No license shown | [P] https://github.com/aw31/openai-imo-2025-proofs ; [S] https://simonwillison.net/2025/Jul/19/openai-gold-medal-math-olympiad/ |
| **Google DeepMind IMO 2025 (Gemini Deep Think)** | PDF of all solutions. | Officially graded and certified by IMO coordinators: 35/42, 5 of 6 problems perfect, within 4.5 h. | Not stated | [P] https://deepmind.google/blog/advanced-version-of-gemini-with-deep-think-officially-achieves-gold-medal-standard-at-the-international-mathematical-olympiad/ ; PDF https://storage.googleapis.com/deepmind-media/gemini/IMO_2025.pdf |
| **DeepSeekMath-V2** | Open-weight 685B MoE model, self-verifying prover. IMO 2025 5/6 (P1-P5), CMO 2024 gold-level, Putnam 2024 118/120. | Self-reported. | License: sources disagree (MIT vs Apache 2.0). UNVERIFIED; check repo. | [P] https://arxiv.org/abs/2511.22570 ; [P] https://github.com/deepseek-ai/DeepSeek-Math-V2 |

### 1.2 Key papers on proof quality

| Finding | Detail | Source |
|---|---|---|
| **"Proof or Bluff" (USAMO 2025)** | Best model Gemini-2.5-Pro averaged 10.1/42 (24.4%). All others under 5% (R1 2.0, Grok 3 2.0, Claude 3.7 1.5, o1-pro 1.2, o3-mini 0.9). | [P] https://arxiv.org/html/2503.21934 |
| Grading setup | 4 expert judges (ex national IMO team or final TST). Each problem graded independently by 2 judges. 0-7 scale, schemes from AoPS, verified by judges. | [P] same |
| Failure modes | Logic (most common), Assumption, Creativity (wrong strategy), Algebra/Arithmetic. | [P] same |
| **LLM judges overrate** | o3-mini and Claude 3.7 as graders gave Gemini-2.5-Pro about 19.5/42 vs human 10.1. Inflation up to ~20x on weak models. | [P] same |
| **OPC judge accuracy** | Gemini-2.5-Pro as judge: 85.4% (pass@1), 88.1% (maj. vote) vs 90.4% human-human. Models judge their *own* proofs significantly worse (except Qwen3-235B). Correct final answer is a poor proxy for a valid proof. | [P] https://arxiv.org/abs/2506.21621 ; summary [S] https://www.emergentmind.com/topics/open-proof-corpus |
| **ProofAutoGrader (IMO-Bench)** | Pearson r vs humans: 0.96 (basic), 0.93 (advanced) on 14 public models; 0.87 on 170 internal systems. Best possible MAE on GradingBench is 3.9%, not 0. | [P] https://imobench.github.io/ |
| **Verification-and-refinement pipeline (Huang & Yang)** | Solve -> self-improve -> verifier writes bug report (critical error vs justification gap) -> optional bug-report review -> correct -> loop. Accept only if verifier passes **5 times in a row**; reject if a major issue persists (diagram says 10 iterations). Gemini 2.5 Pro, Grok-4 and GPT-5 each reached 5/6 on IMO 2025 with it (baseline best-of-32: 31.6%, 21.4%, 38.1%). | [P] https://arxiv.org/html/2507.15855 |
| **Aletheia (GDM, Feb 2026)** | Generator / Verifier / Reviser agent on Gemini 3 Deep Think. IMO-ProofBench Advanced 91.9% (leaderboard). Gemini 3 Deep Think alone 76.7%. 95.1% figure in some press is not on the leaderboard. | [P] https://imobench.github.io/ ; [S] https://www.infoq.com/news/2026/04/deepmind-aletheia-agentic-math/ |

### 1.3 How MathArena grades proofs (the most transparent public method)

| Contest | Method | Source |
|---|---|---|
| USAMO 2025 | Human only. 2 independent expert judges per solution, 0-7, with justification. 4 runs per model. | [P] https://arxiv.org/html/2503.21934 ; [P] https://matharena.ai/usamo/ |
| IMO 2025 | 4 human judges with IMO-level expertise; each anonymized answer graded by 2. Best-of-32 per model, selected by an LLM-judged bracket tournament; the winner goes to humans. Token limit 64k (128k for Grok-4 rerun). Best public model Gemini 2.5 Pro 13/42 (31%). Cost at least $3 per final answer, Grok-4 over $20; $400 for 24 Gemini answers. | [P] https://matharena.ai/imo/ |
| USAMO 2026 | Semi-automated. (1) Gemini-3.1-Pro drafts a rubric per problem from reference solutions (ProofBench style); humans heavily edited most rubrics. (2) Gemini-3.1-Pro rewrites each proof into a standard format; humans check faithfulness. (3) Jury of GPT-5.4, Gemini-3.1-Pro, Opus-4.6 with code execution. If all within 2 points, final = **minimum**; else judges see each other's verdicts, revise, and final = min of revised. (4) Humans review every solution with the jury's notes. Human review changed scores by at most 2 points, and only 3 solutions moved that much. Lesson: rubric wording drove most judge-human gaps. | [P] https://matharena.ai/usamo/ |
| USAMO 2026 results | GPT-5.4 95.2%, Gemini-3.1-Pro 74.4%, Opus-4.6 47.0% (unfinished outputs scored 0), Step-3.5-Flash 44.6%. Later paper: GPT-5.5 98%. | [P] https://matharena.ai/usamo/ ; [P] https://arxiv.org/abs/2605.00674 |
| Contamination policy | Only use problems released after the model; for ArXivMath, only papers from the last month. About 30% of ArXivMath questions could be answered from cited prior work. | [P] https://matharena.ai/arxivmath/ |

### 1.4 IMO 2026 and other 2026 AI results

Mostly secondary sources. The IMO website states no AI results or AI policy ([P] https://www.imo-official.org/ - only shows OpenAI as a Diamond Sponsor).

| Claim | Status | Source |
|---|---|---|
| Huawei Celia and Xiaohongshu dots-note 3.0 each 42/42, graded by IMO organisers (Shanghai, July 2026). | UNVERIFIED (no primary page found) | [S] https://tech-insider.org/ai-imo-2026-perfect-score-odds-hit-96-percent/ ; [S] https://www.digitalapplied.com/blog/imo-2026-perfect-scores-ai-benchmark-saturation |
| NVIDIA Nemotron 3 Ultra 30/42, "graded by the IMO team", no tools. | Primary is NVIDIA's own tweet. | [S] https://postcutoff.com/p/2026-07-21-nvidiaai-nemotron-3-ultra-imo-2026/ citing https://x.com/NVIDIAAI/status/2079642933058244704 |
| Claude Opus 5: 42/42 on IMO 2026, no tools, 4 answers per problem, all 24 accepted by a three-model panel, human experts gave full credit to one pre-specified solution per problem (system card section 8.6). | UNVERIFIED (system card PDF too large to fetch; seen in search snippet) | [P] https://assets.claude.com/ceaf5c7ff2783855203fde8208ec311252dced5b.pdf |
| Claude Opus 4.8 USAMO 2026 96.7% (model-judge grading). | UNVERIFIED | [S] https://benchmarklist.com/benchmarks/usamo_2026/ |
| Several frontier models claimed 42/42 via self-run harness graded by LLM agents. Human cutoffs 2026: gold 29, 7 perfect humans of 666. | UNVERIFIED | [S] https://hiwavemakers.com/blog/ai-perfect-score-imo-2026-what-it-proves/ |

Takeaway: by 2026, top models solve most olympiad problems. The hard part for olymmath is not generating proofs. It is **grading them reliably and labeling them honestly**.

### 1.5 Answer checking (final-answer problems)

| Tool | Notes | Source |
|---|---|---|
| **Math-Verify** (Hugging Face) | Extracts answer from LaTeX (`\boxed{}`, `$..$`, `\[..\]`), plain expressions, or strings; converts to SymPy; compares. Handles sets, intervals, matrices, equations, inequalities, percentages, units, complex numbers. Apache-2.0. `verify(gold, pred)` is **not symmetric**; gold must be clean; floats need precision setting; must pin ANTLR4 runtime. | [P] https://github.com/huggingface/Math-Verify |
| SymPy `simplify(a-b)==0` / `equals()` | Standard fallback. Needs careful parsing of LaTeX. Use Math-Verify as wrapper. | [P] https://docs.sympy.org/latest/modules/core.html (general docs) |

Recommendation: only answer-type problems (AIME, HMMT, many national rounds) get automatic correctness. Proof problems always need a judge plus human spot checks.

### 1.6 Recommended agent pipeline for olymmath-agent

Design based on Huang & Yang (5-pass verifier), MathArena USAMO 2026 (rubric + jury + min + human review) and OPC (no self-judging).

| Stage | What | Model tier | Why |
|---|---|---|---|
| 0. Ingest | Problem + all human solutions + official marking scheme if any. | none | Human-written first. |
| 1. Rubric | Draft a 0-7 marking scheme from human solutions (checkpoints, zero-credit items, deductions). Human reviewer edits. Cache per problem. | Opus | MathArena found rubric wording caused most errors. |
| 2. Generate | k=2-4 independent attempts. Ask for rigorous proof, mark gaps, state final answer in `\boxed{}`. | Sonnet (Opus for hard / MOHS >= 35) | Diversity; cheaper tier first. |
| 3. Self-refine | Verifier bug report (critical error vs gap) -> correction, up to N rounds. | same | Huang & Yang. |
| 4. Grade | Jury of >= 2 judges from **different model families** than the generator if possible, with rubric and reference solution. Final = **min** score. Disagreement > 1 point -> human queue. | Opus + one non-Anthropic judge | OPC: self-judging is worse. MathArena: min of jury. |
| 5. Answer check | For answer problems, Math-Verify against gold. Disagreement with judge -> human queue. | none | Cheap and objective. |
| 6. Human review | Required for publishing a "7/7 verified" badge. Sample-audit others. | human | GradingBench shows irreducible ~4% grader error. |
| 7. Publish | Show grade, judges, rubric, model id, date, cost, and an "AI-generated" label. | none | See 1.8. |

Grade labels to display (from IMO-GradingBench buckets): **Correct (7)**, **Almost (5-6)**, **Partial (1-4)**, **Incorrect (0)**, plus **Human-verified** flag. [P] https://imobench.github.io/

### 1.7 Cost per problem with current Claude models

Prices per million tokens (MTok), standard API, from Anthropic docs.

| Model | Input | Output | 5-min cache write | Cache hit | Batch input | Batch output | Source |
|---|---|---|---|---|---|---|---|
| Claude Opus 5.5 | $4 | $20 | $5 | $0.20 | $2 | $10 | [P] https://platform.claude.com/docs/en/about-claude/pricing |
| Claude Sonnet 5.5 | $2 | $10 | $2.50 | $0.10 | $1 | $5 | [P] same |
| Claude Haiku 5.5 (prompt <= 100k) | $0.10 | $0.50 | $0.125 | $0.01 | $0.05 | $0.25 | [P] same |
| Claude Haiku 5.5 (prompt > 100k) | $0.50 | $2.50 | $0.625 | $0.05 | $0.25 | $1.25 | [P] same |
| Claude Fable 5.1 (top tier) | $10 | $50 | $12.50 | $0.25 | n/a here | n/a here | [P] same; [P] https://claude.com/pricing |

Other pricing facts [P] same page:
- Batch API: **50% off** input and output. Stacks with prompt caching.
- Cache hits on Opus 5.5 / Sonnet 5.5 cost 0.05x base input.
- 1M context at standard price for Claude 4.6+ (except Haiku 5.5, which has a 100k price step).
- US-only inference (`inference_geo`) = 1.1x. Fast mode (Opus 5.5 $8/$40) is not available with Batch.

**Estimate per problem** (my assumptions, not vendor numbers): 4 generation attempts x (2k in + 30k out incl. thinking); 4 attempts x 3 verifier runs x (10k in + 8k out). Thinking tokens are billed as output (assumed; UNVERIFIED on this page).

| Tier (Batch API) | Generation | Grading | Total / problem | 100k problems |
|---|---|---|---|---|
| Haiku 5.5 | ~$0.03 | ~$0.03 | **~$0.06** | ~$6k |
| Sonnet 5.5 | ~$0.61 | ~$0.60 | **~$1.20** | ~$120k |
| Opus 5.5 | ~$1.22 | ~$1.20 | **~$2.40** | ~$240k |

Compare: MathArena paid at least $3 per final IMO 2025 answer with Gemini 2.5 Pro ([P] https://matharena.ai/imo/).

Recommendation: do not run all 300k-1M problems. Start with problems that have no human solution, or high-traffic contests (IMO, USAMO, ISL). Use Haiku/Sonnet for answer-type problems with Math-Verify. Use Opus for rubrics and judging. Always use Batch API and cache the rubric + problem prefix.

### 1.8 Disclosure and labeling of AI content

| Point | Detail | Source |
|---|---|---|
| EU AI Act Art. 50 in force since **2 Aug 2026** | Deployers publishing AI-generated text "to inform the public on matters of public interest" must disclose it, unless it had human review and editorial responsibility. Providers must machine-mark outputs (grace to 2 Dec 2026 for systems already on market). Applies when outputs are used in the EU. Math solutions are probably not "public interest" text, but labeling is cheap and safe. | [S] https://artificialintelligenceact.eu/transparency-rules-article-50/ ; [P] https://digital-strategy.ec.europa.eu/en/policies/guidelines-ai-transparency-obligations ; [S] https://www.bakerbotts.com/thought-leadership/publications/2026/september/eu-ai-act-article-50-transparency-obligations-go-live |
| Stack Overflow ban | Generative AI answers banned because they are often wrong but look right. Math.SE policy assumed similar. | [S] https://en.wikipedia.org/wiki/Stack_Overflow ; policy https://meta.stackoverflow.com/questions/421831 (Math.SE policy UNVERIFIED) |
| AoPS forum policy on AI solutions | No written policy found. UNVERIFIED. | - |
| Anthropic watermark for Art. 50 | Claimed only on Wikipedia. UNVERIFIED. | [S] https://en.wikipedia.org/wiki/Text_watermarking |

Recommended labeling:
- Separate tabs or sections: "Human solutions" first, "AI solutions" second, collapsed by default.
- Each AI solution shows: model id, date, pipeline version, grade (0-7), grader(s), human-verified yes/no, link to rubric.
- Machine-readable marker in JSON and HTML (`data-ai-generated="true"`, schema.org `creativeWorkStatus` or a custom field).
- Never present an AI solution as official or as an author's solution.

### 1.9 Contamination and memorization

| Finding | Detail | Source |
|---|---|---|
| MATH-Perturb | 18 LLMs drop 10-25% on hard perturbations of MATH level-5 problems. Models apply the original problem's steps to the changed problem. | [P] https://arxiv.org/abs/2502.06453 |
| LiveAoPSBench | All 17 LLMs did worse on 2024 AoPS problems than 2023 problems (2.4% to 23.6% drop). | [P] https://arxiv.org/abs/2501.14275 (summary [S] https://en.papernotes.org/ICML2025/llm_evaluation/leveraging_online_olympiad-level_math_problems_for_llms_training_and_contaminati/) |
| VAR-MATH | Up to -70% on symbolic variants of AIME24 for small math models. | [P] https://arxiv.org/abs/2507.12885 |
| MathConstruct | Counterpoint: rephrased problems scored similarly, little contamination seen. | [P] https://arxiv.org/abs/2502.10197 |
| IMO-ProofBench | Experts altered problems to avoid memorization. | [P] https://imobench.github.io/ |

Implications for olymmath:
- A correct AI proof of a famous old problem may be recall of the AoPS / official solution, not reasoning. Label it "may reflect training data".
- Show the problem's date and the model's release date. Flag "post-cutoff" solutions (problem published after model release) as stronger evidence.
- Detect near-copies: compare AI solution n-grams / embeddings against the human solutions on the same page. If similarity is high, show "closely matches human solution X" or drop it.
- Do not use AI grades on old problems as a measure of model skill.

---

## Part 2. Publishing a very large static math site

### 2.1 Hosting limits (official docs)

| Platform | Limit | Value | Source |
|---|---|---|---|
| **Cloudflare Pages** | Files per site | Free **20,000**; Pro/Business/Enterprise **100,000** (needs `PAGES_WRANGLER_MAJOR_VERSION=4`) | [P] https://developers.cloudflare.com/pages/platform/limits/ |
| | File size | 25 MiB | [P] same |
| | Builds / month | Free 500, Pro 5,000, Business 20,000 | [P] same |
| | Build timeout | 20 min | [P] same |
| | Concurrent builds | Free 1, Pro 5, Business 20 | [P] same |
| | Projects per account | 100 (not routinely raised) | [P] same |
| | Custom domains per project | Free 100, Pro 250, Business 500 | [P] same |
| | `_redirects` | 2,000 static + 100 dynamic | [P] same |
| | `_headers` | 100 rules | [P] same |
| | Status | Cloudflare now recommends **Workers** for new projects; Pages is being folded into Workers. | [P] https://developers.cloudflare.com/workers/static-assets/migration-guides/migrate-from-pages/ ; [S] https://flaviocopes.com/cloudflare-pages-vs-workers/ |
| **Workers static assets** | Files per Worker version | Free **20,000**; Paid **100,000** | [P] https://developers.cloudflare.com/workers/platform/limits/ |
| | File size | 25 MiB | [P] same |
| | Static asset requests | Free and unlimited; only Worker-script invocations are billed | [P] https://developers.cloudflare.com/workers/static-assets/billing-and-limitations/ |
| | Worker requests | Free 100,000/day; Paid $5/month incl. 10M requests, then $0.30/M; 30M CPU-ms incl. | [P] https://developers.cloudflare.com/workers/platform/limits/ ; [P] https://developers.cloudflare.com/workers/platform/pricing/ |
| **R2** | Storage | $0.015/GB-month; free 10 GB-month | [P] https://developers.cloudflare.com/r2/pricing/ |
| | Ops | Class A $4.50/M (free 1M/month); Class B (reads) $0.36/M (free 10M/month); egress free | [P] same |
| **D1** | DB size | Paid 10 GB, Free 500 MB per DB; Paid up to 1,000,000 DBs (first 100,000 included), Free 10 | [P] https://developers.cloudflare.com/d1/platform/limits/ |
| | Queries per Worker invocation | Paid 1,000, Free 50 | [P] same |
| | Pricing (Paid) | 25B rows read/month incl., then $0.001/M; 50M rows written incl., then $1/M; 5 GB incl., then $0.75/GB-month | [P] https://developers.cloudflare.com/workers/platform/pricing/ |
| | FTS5 | Supported (virtual tables); databases with virtual tables cannot be exported (drop, export, recreate). | [S] https://zander.wtf/blog/astro-cloudflare-d1-search/ citing Cloudflare docs https://developers.cloudflare.com/d1/best-practices/import-export-data/ |
| **GitHub Pages** | Published site | **1 GB** max | [P] https://docs.github.com/en/pages/getting-started-with-github-pages/github-pages-limits |
| | Source repo | 1 GB recommended | [P] same |
| | Bandwidth | 100 GB/month soft | [P] same |
| | Builds | 10/hour soft (not for custom Actions workflows) | [P] same |
| | Deploy timeout | 10 min | [P] same |
| **GitHub repos** | Size | 10 GB on-disk `.git` guideline; file hard cap 100 MB (1 MB recommended); push 2 GB; ~3,000 entries per directory; 300 files per diff view | [P] https://docs.github.com/en/repositories/creating-and-managing-repositories/repository-limits |

### 2.2 What the limits mean for 300k-1M pages

- One page per problem at 300k-1M HTML files **does not fit** a single Pages project or Worker (max 100k files).
- GitHub Pages: 1M pages x ~8 KB = ~8 GB. **Exceeds 1 GB.** GitHub Pages can only mirror a subset (e.g. top contests) or a JSON-shard SPA.
- The content repo must shard directories (~3,000 entries per directory guideline).

Sharding options:

| Option | How | Pros | Cons |
|---|---|---|---|
| A. Multiple Pages/Workers projects | Split by contest family into <= 100k-file projects; route `/c/<family>/...` via one Worker or subdomains. | Pure static, free asset requests. | 100 projects cap; 10+ deploys; cross-links span projects. |
| B. Pack many problems per file | One HTML page per contest year (e.g. "IMO 2019"), with anchors per problem. 1M problems / ~10 per page = ~100k files. | Fits 1 Worker on Paid. Fewer files, better for browsing. | Long pages; per-problem URLs are anchors. |
| C. Client-side render from JSON shards | Static shell + `shards/<prefix>.json` (e.g. 1,000 problems each = 1,000 files). | Tiny file count; fits GitHub Pages too. | Weak SEO unless prerendered; needs JS. |
| D. **Worker renders from R2** (recommended at full scale) | Store problem JSON (or prebuilt HTML) in R2; Worker fetches by key, renders template, caches at edge. Static assets (CSS, fonts, KaTeX) as Worker assets. | No file cap. 1M objects x 10 KB = 10 GB = ~$0.15/month storage. Reads $0.36/M. | Needs Workers Paid ($5/month). Worker requests billed. |
| E. Hybrid | Static prerender for the top ~90k pages (IMO, USAMO, ISL, national olympiads), Worker+R2 for the long tail; GitHub Pages mirror only the static part (<1 GB). | Best of both; mirror stays useful. | Two code paths. |

Recommendation: **E**. Use Workers static assets (not legacy Pages) for the main site on a Paid plan; R2 for the long tail; GitHub Pages mirrors a curated static subset. Keep math2/kvant style for the static part.

### 2.3 Search

| Option | Scale facts | Fit | Source |
|---|---|---|---|
| **Pagefind** | Chunked index; ~10k-page site downloads < 300 KB per search. Build memory roughly ~200 MB per 1,000 long pages (2 GB at 10k). Users report crashes at ~250k-300k pages; one gave up at 500k. **Multisite**: `pagefind.mergeIndex(url)` merges bundles in the browser, with `indexWeight` and `mergeFilter`. | Good per shard (e.g. one index per contest family, merged on query). | [P] https://cloudcannon.com/blog/introducing-pagefind/ ; [P] https://pagefind.app/docs/multisite/ ; [S] https://www.drupal.org/project/pagefind ; [S] https://discourse.gohugo.io/t/scaling-pagefind-to-over-1-million-pages/56109 |
| Orama | Full features (typos, facets, vectors). One comparison puts practical max ~50k docs in browser. | Small facet indexes only. | [S] https://npm-compare.com/@elastic/elasticsearch,@orama/orama,flexsearch,fuse.js,lunr,search-index |
| FlexSearch | Fast; v0.8 has persistent indexes in IndexedDB. Index JSON dominates size (~0.5-1 MB per 1,000 docs). | Title/metadata search over shards. | [S] https://www.pkgpulse.com/guides/fusejs-vs-flexsearch-vs-orama-client-side-search-2026 |
| **D1 + FTS5** | FTS5 virtual tables with bm25; 10 GB DB on Paid; rows read very cheap. FTS index ~20-30% of base table. | **Best server-side choice** on Cloudflare. Fits 1M problem statements. | [S] https://zander.wtf/blog/astro-cloudflare-d1-search/ ; [S] https://www.ai-analytics.org/writing/federal-fts5-search/ |
| Typesense | In-memory; RAM ~2-3x indexed field size. | Needs a server. | [P] https://typesense.org/docs/guide/system-requirements.html |
| Meilisearch | MIT self-host; Cloud from $20-30/month (100k docs, 50k searches base). | Needs a server; cost grows with docs. | [P] https://www.meilisearch.com/pricing |
| sql.js-httpvfs | SQLite over HTTP Range on static hosts; chunked DB files. No tests on VFS; "not production-ready"; no cache eviction. | Possible for GitHub Pages mirror only. | [P] https://github.com/phiresky/sql.js-httpvfs |
| DuckDB-wasm + Parquet | Range reads over HTTPS; FTS extension exists. Report: versions after 1.29.0 download whole Parquet instead of ranges. | Good for analytics/filter UIs, risky for FTS. | [P] https://duckdb.org/docs/clients/wasm/extensions ; [S] https://github.com/duckdb/duckdb-wasm/discussions/2247 |
| **Approach0** (formula search) | Math-aware search, text + formula queries. Public C code MIT but master inactive since 2019; newest engine closed. | Research reference; hard to host. | [P] https://github.com/approach0/search-engine ; https://approach0.xyz/search/ |
| **Tangent-CFT** (formula embeddings) | fastText embeddings over SLT/OPT tuples; ARQMath baseline (nDCG' 0.641 on ARQMath-3 task 2). | Could precompute vectors -> Cloudflare Vectorize or brute-force per shard. | [P] https://github.com/BehroozMansouri/TangentCFT ; [P] https://www.cs.rit.edu/~rlaz/files/Mansouri_ICTIR2019.pdf |

Recommendation: D1 FTS5 behind a Worker for the main site; Pagefind multisite (sharded per contest family) for the static mirror. Formula search is phase 2: normalize LaTeX to tokens and add them to FTS, then consider Tangent-CFT vectors.

### 2.4 Math rendering

| Tool | Key facts | Source |
|---|---|---|
| **KaTeX** | `katex.renderToString(tex, opts)` on the server. `output` default `htmlAndMathml` (visual HTML + MathML for a11y). `trust` default false (blocks `\includegraphics`, `\href` etc.). `strict` default "warn". `maxExpand` default 1000. Share one `macros` object only within trusted content. Client still needs `katex.css` and fonts. | [P] https://katex.org/docs/api ; [P] https://katex.org/docs/options |
| **MathJax 4.0** | Final release (docs dated 14 Dec 2025). 11 new fonts, new default `mathjax-newcm`; fonts split into dynamic chunks; **line breaking** for inline and display math; speech in a web worker; Node SSR via `@mathjax/src`. Known line-break bug fixed after 4.0.0. | [P] https://github.com/mathjax/MathJax/releases ; [P] https://docs.mathjax.org/_/downloads/en/v4.0/pdf/ ; [P] https://www.mathjax.org/ |

Recommendation: KaTeX server-side prerender at build (fast, no client JS, matches math2/kvant). Keep a MathJax 4 fallback in the client for formulas that KaTeX fails to parse (log failures at build time). Long display formulas on mobile are the main KaTeX weakness; MathJax 4 line breaking solves it.

### 2.5 Diagrams

| Source format | Build-time approach | Source |
|---|---|---|
| Asymptote `[asy]` from AoPS | `asy -f svg -o out.svg file.asy` (uses LaTeX + dvisvgm). Asy and dvisvgm versions must match. Many AoPS snippets use `olympiad.asy` and `cse5.asy` modules, which are not in standard Asymptote; vendor them into the asy path (UNVERIFIED location). Run in a sandbox with timeout: Asymptote is a full programming language. Cache SVG by content hash. | [P] https://github.com/vectorgraphics/asymptote/issues/33 ; [P] https://sourceforge.net/p/asymptote/discussion/409349/thread/1b7792a4/ |
| TikZ | LaTeX -> DVI -> `dvisvgm` -> SVG. | [P] https://www.tug.org/TUGboat/tb38-3/tb120gieseking.pdf |
| GeoGebra | Embed via applet only when interactivity matters; otherwise export static SVG. | UNVERIFIED (not researched in depth) |

### 2.6 UI references and difficulty rating

| Site | What to copy | Source |
|---|---|---|
| AoPS Contest Collections / Wiki | Tree browse by country -> contest -> year -> problem; many solutions per problem; `[asy]` diagrams. | https://artofproblemsolving.com/community/c13_contests (could not fetch; 403) |
| imo-official.org | Clean per-year problem PDFs and results. | [P] https://www.imo-official.org/ |
| MathArena | Model x problem score grid; click a cell to see the full output and judge notes. Good model for "AI solutions" view. | [P] https://matharena.ai/ |
| Codeforces problemset | Table with tags, rating range filter, solved count. Problem rating = rating at which a user solves it with p=0.5 in-contest (Elo-like; 200 below -> 0.75, 400 below -> 0.9). Range 800-3500. Exact algorithm unpublished ("weights, coefficients and heuristics"). | [P] https://codeforces.com/blog/entry/62865 ; [S] https://codeforces.com/blog/entry/107162 |
| Project Euler | Difficulty % per problem from solve data. | https://projecteuler.net/about (could not fetch; 403; UNVERIFIED) |

Difficulty scales:

| Scale | Detail | Source |
|---|---|---|
| **AoPS competition ratings 1-10** | Per contest/problem slot. AIME 1-5: 3, 6-9: 4, 10-12: 5, 13-15: 6. USAMO P1/4: 7, P2/5: 8, P3/6: 9. IMO P1/4: 6.5, P2/5: 7.5-8, P3/6: 9.5. ISL 1-2: 5.5-7, 3-4: 7-8, 5+: 8-10. China TST sets level 10. | [S] search snippets of https://artofproblemsolving.com/wiki/index.php/AoPS_Wiki:Competition_ratings and https://artofproblemsolving.com/wiki/index.php/International_Mathematical_Olympiad (page 403 to fetcher) |
| **MOHS (Evan Chen)** | Personal per-problem ratings, 0M-60M in steps of 5 (in practice 0-50). Covers IMO, USAMO, TSTST, TST. Latest PDF dated 22 July 2026. Chen's 2024 post "MOHS was a mistake" advises advanced students not to rely on it. No reuse license seen: link, do not copy. | [P] https://web.evanchen.cc/upload/MOHS-hardness.pdf ; [P] https://blog.evanchen.cc/2024/01/05/mohs-was-a-mistake/ |
| Epoch AI unified scale | Maps AoPS levels and MOHS together (MOHS 5-15 -> 7, 20-35 -> 8, 40-50 -> 9). | [P] https://epoch.ai/data-insights/math-competitions |

Recommendation for olymmath difficulty:
1. Prior: AoPS-style 1-10 level from contest + problem slot (rule table).
2. Where available, show MOHS as an external link, not as our own number.
3. Data-driven: official per-problem score statistics (IMO publishes per-problem averages) -> Elo/IRT difficulty. Later, AI solve rate across k attempts as a second signal (label it as such; it is not human difficulty).
4. Filters: contest, country, year, topic (A/C/G/N), level range, "has human solution", "has AI solution", "AI grade", "human-verified".

---

## Summary of recommendations

1. **Grading**: rubric per problem (LLM draft + human edit), jury of 2-3 judges from different families, final = min, auto human queue on disagreement > 1 point, Math-Verify for answers. Never let the generator grade itself.
2. **Cost**: Batch API (50% off) + caching. Rough per problem: Haiku 5.5 ~$0.06, Sonnet 5.5 ~$1.20, Opus 5.5 ~$2.40. Prioritize; do not blanket-run 1M problems.
3. **Labeling**: human first; AI clearly labeled with model, date, grade, verifier, human-verified flag; contamination note for pre-cutoff problems.
4. **Hosting**: 100k-file cap per Pages project / Worker version -> hybrid: static Workers assets for top ~90k pages, Worker + R2 for long tail, D1 FTS5 for search. GitHub Pages mirror only a < 1 GB subset.
5. **Rendering**: KaTeX SSR, MathJax 4 fallback; Asymptote -> SVG at build in a sandbox, cached by hash.
6. **Data reuse**: OPC, IMO-Bench (CC-BY 4.0) and ProofBench are usable as judge calibration sets. MathArena outputs are CC BY-NC-SA: OK to link, check before republishing.
