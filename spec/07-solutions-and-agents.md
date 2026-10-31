# 07. Solutions: human first, then AI agents, and how each is graded

Evidence: 01e Part 1. The finding that shapes this document: by 2026 frontier models solve most olympiad problems (MathArena USAMO 2026: GPT-5.4 95.2%; several labs claim 42/42 at IMO 2026, mostly UNVERIFIED). **Writing proofs is no longer the hard part. Grading them reliably and labelling them honestly is.**

## 7.1 Order of solutions on a problem page

1. **Official** solutions (H1): organizer booklets, IMO shortlist booklets, MAA solution pamphlets, UKMT solutions, official national solutions.
2. **Curated human** solutions (H2): IMO Compendium, Evan Chen's notes (link-only unless licensed, document 11), Kedlaya's Putnam solutions, problems.ru solutions, AoPS Wiki solutions.
3. **Community human** solutions (H3): AoPS thread replies, Math StackExchange answers, forum posts, ranked by grade, then by length-normalized community signal (AoPS likes), then by date. Each is shown as its zone allows (document 11): in full for Z1 (Stack Exchange, opted-in authors), as author, excerpt and link for Z2 (AoPS by default). The grade and the ranking are computed from the private copy either way, so a Z2 solution still gets its place and its badge.
4. **AI** solutions (M), in a separate section that is collapsed by default, ranked by grade.

The page always shows a human solution first when one exists. An AI solution is never placed above a human solution, even if the human solution is graded lower. It is shown as "graded higher" in its own section.

## 7.2 Which problems get AI solutions

Running agents on every problem is not worth it (7.6). The selection order:

| Priority | Set | Why |
|---|---|---|
| P1 | problems with **no human solution at all** after full harvest | the AI solution fills a real gap |
| P2 | problems whose only human solutions are sketches or "answer only" | a full proof adds value |
| P3 | answer-type problems with no official answer, or with conflicting answers | the answer check settles the record |
| P4 | high-traffic problems (IMO, ISL, USAMO, national olympiads of top countries) | users ask for alternative solutions |
| P5 | recent problems (published after the model release) | the solution is not recall; useful evidence for the community |

## 7.3 The solver agent

The agent loop follows the published designs that work: the verification-and-refinement loop of Huang and Yang (arXiv:2507.15855), MathArena's USAMO 2026 grading jury, and the kvant-solver loop of "solve, then grade against the printed solution".

| Stage | What | Model (default) |
|---|---|---|
| S0 context | the canonical statement; the answer if known; no human solutions (so the attempt is independent) | none |
| S1 generate | k independent attempts (k = 2 by default, 4 for hard problems); rigorous proof; answer in `\boxed{}`; gaps marked explicitly | Sonnet 5.5; Opus 5.5 for difficulty band ≥ 8 |
| S2 self-check | a verifier pass writes a bug report: critical error or justification gap, with the line | same family |
| S3 revise | the generator fixes the reported issues; up to 5 rounds; accept when the verifier passes 3 times in a row | same |
| S4 answer check | answer problems: compare the boxed answer with the gold answer (`answer` package, Math-Verify in a container) | none |
| S5 dedupe | compare with the human solutions of the problem (embedding and n-gram); a near copy is flagged "closely follows solution X" and is not published as a separate solution | none |

All model calls use the Batch API and cache the shared prefix (the system prompt, the problem). The agent never sees the internet. Its tools are a sandboxed Python with SymPy (for computations and small searches) and, in phase 3, the Lean checker (document 08).

## 7.4 Grading

A grade is a record:

```yaml
grade:
  solution: 4M2XKC9QRT
  scale: imo-7
  score: 7
  bucket: correct          # correct (7) | almost (5 to 6) | partial (1 to 4) | incorrect (0)
  rubric: imo/2024/p1@r2   # the rubric file and its version
  graders:
    - {by: model:claude-opus-5-5, score: 7, notes_hash: …}
    - {by: model:<other-family>, score: 7, notes_hash: …}
    - {by: human:reviewer/<id>, score: 7, at: 2026-12-01}
  rule: min-of-jury-v1
```

The rules:

- **Rubric first.** Each problem that gets AI grading has a rubric file `rubrics/<locator>.md`: the checkpoints with points, the zero-credit items, the common deductions. An official marking scheme is used when it is public. Otherwise a model drafts the rubric from the human solutions and a person edits it. MathArena found that rubric wording caused most grading errors, so a rubric is reviewed before it is used at scale.
- **A jury, not one judge.** At least two judges, at least one of another model family than the generator. A model does not grade its own output (OPC found self-judging worse).
- **Minimum.** The jury score is the minimum of the judges' scores. When judges differ by more than 1 point, they see each other's notes and revise once; then the minimum. A difference that remains goes to the human queue.
- **Humans for the badge.** "Verified 7/7" needs a human grade. Model-only grades show as "model-graded".
- **Human solutions are graded too,** with the same rubric, when a forum solution is used as the top solution. This finds wrong forum solutions, which are common.
- **Calibration.** The grader is measured on the public labelled sets before it is used: IMO-GradingBench (1,000 human-graded proofs, CC BY 4.0), the Open Proof Corpus (5,062 judged proofs), ProofBench. The target is agreement with humans close to human-human agreement (OPC: 90.4%). The numbers are in the quality report (09.4).

## 7.5 Labelling

Each AI solution on the site shows: "AI-written", the model id, the date, the pipeline version, the grade and who graded it, whether a human verified it, the rubric link, and the cost. The HTML carries `data-origin="ai"` and the JSON a field `origin: ai`. A contamination note is shown when the problem was public before the model's training cutoff: "This problem was public before the model was trained. The solution may reflect solutions the model has seen."

## 7.6 Cost

Prices (Anthropic, October 2026, per million tokens, Batch API 50% off): Opus 5.5 $4 / $20, Sonnet 5.5 $2 / $10, Haiku 5.5 $0.10 / $0.50. With the assumptions of 01e §1.7 (k = 4 attempts, 3 verifier runs each), a problem costs about $0.06 on Haiku, $1.20 on Sonnet and $2.40 on Opus with Batch. With k = 2 and early stop on easy problems, the Sonnet figure drops to about $0.50.

| Plan | Problems | Budget (estimate) |
|---|---|---|
| P1 + P2 gap fill, Sonnet, k = 2 | ~50,000 | ~$25,000 |
| P3 answer settling, Haiku, k = 4 | ~30,000 | ~$2,000 |
| P4 alternative solutions, Opus, k = 2 | ~5,000 | ~$6,000 |
| Grading jury on everything published | none | about equal to generation |

`om solve` takes `--budget` and stops at it. Each run writes its token counts and cost to `runs/`, using the `pricing` package of kvant-solver. These numbers are estimates; the first batch of 500 problems measures the real cost per difficulty band, and the plan is revised from it.

## 7.7 Human contributions

A person can add a solution by pull request (a Markdown file in `solutions/`), or later by a form on the site that opens the pull request. The author chooses CC BY-SA 4.0 or CC BY 4.0. People can also add grades (as reviewers) and rubric edits. Contributors are credited on the solution page and in `people.yaml`.
