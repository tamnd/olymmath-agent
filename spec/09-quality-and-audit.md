# 09. Quality, review and the audit

The goal is the highest quality, not the most rows. This document says what "quality" means for each record, how it is checked, and which rules `om audit` runs on each pull request.

## 9.1 Status of a statement

| Status | Meaning | Who can set it |
|---|---|---|
| `captured` | a candidate is attached; no human text of class H1/H2 yet | the pipeline |
| `normalized` | the canonical text passed normalization and the render check | the pipeline |
| `sourced` | the canonical text is H1 or H2 and its capture is in the raw store | the pipeline |
| `verified` | a person compared the canonical text with the official source (PDF page, booklet) and it matches | a reviewer |
| `disputed` | two sources disagree on the mathematics (a constant, a condition), and no reviewer has decided | the pipeline or a reviewer |

The site shows the status on each problem page. A `disputed` problem shows both texts and the difference.

## 9.2 Status of a solution

| Field | Values |
|---|---|
| `origin` | official, human, ai, human+ai |
| `complete` | the author claims a full solution, or a sketch, or partial progress |
| `grade` | none; one or more grade records (07.4) |
| `answer_check` | match, mismatch, none, n/a |

The badge on the site is derived: "official", "human, graded 7/7", "human, not graded", "AI, graded 7/7 by 2 humans", "AI, model-graded 7/7", "AI, not graded". An AI solution with no human grade never shows a green badge.

## 9.3 The audit rules

`om audit` runs on each pull request of `tamnd/olymmath` and on the full corpus nightly. A rule has a family letter and a number, as in math2. A failing rule of level error blocks the merge. A warning is shown in the pull request and in the coverage report.

| Rule | Level | Checks |
|---|---|---|
| **T, tags and identity** |||
| T01 | error | each tag is valid (check character) and unique |
| T02 | error | no tag is reused; each removed tag is in `aliases.tsv` |
| T03 | error | each locator is in the competition register and each old locator is in `redirects.tsv` |
| T04 | error | each appearance belongs to exactly one problem |
| **S, statements** |||
| S01 | error | each problem has a canonical statement and its rule name is valid |
| S02 | error | the canonical statement is not of class M |
| S03 | error | each statement has a source, a capture hash that exists in the raw store, and a licence zone |
| S04 | error | the canonical text renders with KaTeX without error |
| S05 | warning | the canonical text is of class D (find the original) |
| S06 | warning | two statements of one problem differ in a number or an operator (possible `disputed`) |
| S07 | error | a statement in zone Z3 (link only) has no text in the published tree; its text is only in the private raw store |
| S08 | warning | a figure referenced in the text has no file |
| **D, duplicates** |||
| D01 | error | no two problems have T-signal Jaccard ≥ 0.95 without a `variant_of` edge or a reviewed "not same" record |
| D02 | warning | pairs above the review threshold that no one reviewed |
| D03 | error | each merge of the last commit has a review or two strict signals (6.5). The merges register has the signal E in an eighth column when the merge used it |
| **L, solutions** |||
| L01 | error | each solution points to an existing problem |
| L02 | error | each AI solution has a run block (model, prompt version, date, run id, transcript hash) |
| L03 | error | no AI solution has `origin: human`; no translated solution hides its translator |
| L04 | error | each solution has a zone; a Z3 solution is a link only; a Z2 solution is published as author, link and an excerpt of at most two lines; only Z1 solutions are published in full |
| L05 | warning | an answer-type problem has a solution whose answer mismatches the problem answer |
| L06 | warning | a solution is a near-duplicate of another solution of the same problem |
| **G, grades** |||
| G01 | error | each grade names its grader, rubric version and date |
| G02 | error | a "human" grade is by a person in `registers/people.yaml` with the role reviewer |
| G03 | warning | model grades disagree by more than 2 points out of 7 and no human grade exists |
| **F, formal** |||
| F01 | error | each formal statement compiles at the pinned toolchain (08.3) |
| F02 | error | each proof compiles, uses no `sorry`, and uses only the allowed axioms |
| F03 | error | each formal statement has a fidelity record (08.4) before it is shown as "faithful" |
| **C, coverage** |||
| C01 | warning | an edition in the register has positions without a problem |
| C02 | warning | a competition in the register has editions without any problem |
| C03 | info | coverage numbers changed by more than 1% since the last nightly |
| **X, legal** |||
| X01 | error | each source in use has a licence zone and a ToS note in `registers/sources.yaml` |
| X02 | error | no capture from a source with a `do-not-republish` flag is in a published export, and no text from a source with `ai-train=no` (AoPS, ProofWiki) is in the Hugging Face export |

## 9.4 Measured quality

The audit checks rules. It does not measure accuracy. Accuracy is measured by sampling, and the numbers are in the quarterly quality report:

| Measure | Method | Target |
|---|---|---|
| statement accuracy | a reviewer checks 200 random `sourced` statements against the official source | ≥ 99.5% exact in mathematics |
| locator accuracy | 200 random appearances: is the competition, year and number right | ≥ 99.5% |
| duplicate rate | 500 random problems, nearest 5 neighbours by E: is any a missed duplicate | ≤ 0.5% |
| false-merge rate | 200 random merges: is each a true duplicate | ≤ 0.2% |
| OCR error rate | 100 random OCR statements, character error rate in mathematics | ≤ 0.1% |
| AI grade agreement | 100 AI solutions graded by people and by the grader models | report Cohen's κ; target ≥ 0.8 |
| human-first share | share of problems whose canonical text is H1/H2 | grows each release; target 95% for register competitions |

The labelled sets used for these measures (`bench/` in `olymmath-agent`) are versioned. A changed threshold must show its effect on the labelled sets in its pull request.

## 9.5 Review

Reviewers are people in `registers/people.yaml`. A review is a file in `reviews/<yyyy>/<mm>/<id>.yaml` with the reviewer, the records, the decision and a note. The review tool `om review` shows the queue in the terminal (a side-by-side view of the two texts, the PDF crop and the signals), and the site has a read-only queue page. The queue is ordered by value: register competitions before loose problems, H1/H2 conflicts before D, popular problems (by AoPS views) before rare ones.
