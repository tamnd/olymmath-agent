# 04. The object model and identifiers

This document defines the records of olymmath, their identifiers and the files that hold them. The rest of the spec refers to the record kinds by the names in bold.

## 4.1 The record kinds

| Kind | One record is | Example |
|---|---|---|
| **competition** | a recurring contest, with an organizer | IMO, USAMO, All-Russian MO, Putnam |
| **edition** | one year (or one sitting) of a competition | IMO 2024 |
| **paper** | one round, day, level or grade of an edition | IMO 2024 Day 1; All-Russian 2019 Grade 10 Day 2 |
| **problem** | one mathematical task, independent of where it appeared | "Determine all real α such that ..." |
| **appearance** | the fact that a problem was set in a paper, at a position | IMO 2024 P1; ISL 2024 N1 |
| **statement** | one text of a problem, in one language, from one source | the official English text of IMO 2024 P1 |
| **solution** | one argument that solves a problem, with its author and source | the official solution; a post on AoPS; an AI proof |
| **answer** | the final answer of an answer-type problem, in a checkable form | `{α : α is an even integer}` |
| **grade** | a judgement of one solution: score, grader, rubric, notes | 7/7 by two human graders |
| **formal** | one formal statement of a problem, in Lean 4 (later other provers) | `theorem imo2024_p1 ...` |
| **proof** | one formal proof of one formal statement, with its check record | a Lean proof that compiles at the pinned Mathlib |
| **source** | one place the project harvests from | `aops-collections`, `hf:AI-MO/NuminaMath-1.5`, `imo-official` |
| **capture** | one raw item from one source, immutable, with its hash | one AoPS post; one HF row |
| **person** | a problem author, a solver, a grader | the proposer of ISL 2019 C9 |
| **topic** | a node of the topic tree | `N.diophantine.pell` |

The problem is the centre. A problem has one or more appearances, one or more statements, zero or more solutions, zero or one answer, zero or more formal statements. A capture is never shown as content by itself. It is evidence for a statement, a solution or an appearance.

### Why a problem is separate from its appearance

The same problem is often set more than once:

- An ISL problem becomes an IMO problem. It then appears as ISL 2024 N1 and IMO 2024 P1. The ISL text and the IMO text can differ.
- A national olympiad reuses a problem from another country, or a TST reuses a shortlist problem.
- A problem appears in a magazine (Kvant, KöMaL, Crux) and later in a contest.

If the appearance were the identity, the site would show the same problem many times with different solutions. With a separate problem record, the site shows one problem page with all its appearances, all its statements and all its solutions.

## 4.2 Identifiers

Each record of the kinds problem, solution, formal, proof and person has a **tag**. The format is the long tag of math2 (math2 corpus spec K3): ten characters of Crockford base32, nine random characters and one Luhn mod 32 check character. Example: `7K3QX2M9R2`. The resolver normalizes case, hyphens and the Crockford substitutions and rejects a tag with a wrong check character. The tag generator is the same Go package as in math2-agent, vendored or imported, so that a tag means the same thing in both projects.

A tag is permanent. It is never reused. When two problems are found to be the same, one tag becomes an alias of the other (4.6). It is not deleted.

Competitions, editions, papers and appearances have **locators** instead of tags. A locator is a readable path:

| Kind | Locator form | Example |
|---|---|---|
| competition | `<comp>` | `imo`, `usamo`, `ru-allrussian`, `vn-vmo`, `cn-mo` |
| edition | `<comp>/<edition>` | `imo/2024`, `putnam/1985`, `ru-allrussian/2019` |
| paper | `<comp>/<edition>/<paper>` | `ru-allrussian/2019/g10-d2`, `amc12/2021/a` |
| appearance | `<comp>/<edition>[/<paper>]/<position>` | `imo/2024/p1`, `isl/2024/n1`, `amc12/2021/a/p25`, `putnam/1985/b6` |

The competition codes are a controlled list in `registers/competitions.yaml` of the content repository (document 05 gives the catalogue). National competitions take an ISO 3166-1 alpha-2 prefix (`ru-`, `vn-`, `cn-`, `us-` is omitted for the AMC family by custom). An edition is usually the year. When a competition has two sittings a year, the edition is `2021a` or `2021-spring`; the register says which form each competition uses.

A locator can change when a curator fixes a mistake, for example a wrong problem number. The old locator stays in `registers/redirects.tsv`, and the site serves a redirect.

### URIs

| URI | Resolves to |
|---|---|
| `om://7K3QX2M9R2` | the problem, solution, formal or proof with that tag |
| `om://imo/2024/p1` | the appearance, and through it the problem |
| `om://imo/2024` | the edition |
| `om://src/aops-collections/post/12345678` | a capture |

The same paths are the site URLs: `https://olymmath.tamnd.com/imo/2024/p1` and `https://olymmath.tamnd.com/p/7K3QX2M9R2`. Both are stable.

### External identifiers

A problem keeps every external identifier that names it, in the field `xref`. This is how links to and from other sites work and how deduplication starts (document 06).

| Key | Example value |
|---|---|
| `aops_topic` | `3107211` |
| `aops_wiki` | `2021_AMC_12A_Problems/Problem_25` |
| `problems_ru` | `108765` |
| `imo_official` | `2024-1` |
| `hf` | `AI-MO/NuminaMath-1.5#row=123456` |
| `omni_math` | the row index in Omni-MATH |
| `compfiles` | `Imo2024P1` |
| `minif2f` | `imo_1959_p1` |
| `putnambench` | `putnam_1985_b6` |
| `mathlib_archive` | `Archive/Imo/Imo2024Q1.lean` |
| `math2` | a math2 tag, when the problem is also a math2 node |
| `kvant` | `kvant_1975_1` problem id from the kvant corpus |

## 4.3 The problem file

The content repository holds one Markdown file for each problem. The file has YAML front matter and a body. The body holds the canonical statement and, after it, the other statements. Solutions are separate files (4.4), because a problem can have dozens of them and they have different authors and licences.

Path: the file lives at the locator of its **primary appearance**, so that the tree is browsable on GitHub:

```
problems/imo/2024/p1.md
problems/isl/2024/n1.md          -> a stub that points to the tag of imo/2024/p1
problems/_loose/7K3/7K3QX2M9R2.md  a problem with no known appearance
```

The primary appearance is the earliest official appearance in a competition. If the problem has no known appearance (a forum problem, a book problem), the file is in `_loose/`, sharded by the first three characters of the tag. A path is not an identity. The tag is the identity.

```markdown
---
tag: 7K3QX2M9R2
kind: problem
appearances:
  - at: imo/2024/p1
    primary: true
  - at: isl/2024/n1
    note: "shortlisted as N1; same text"
title: null                      # only for problems with a well-known name
type: proof                      # proof | answer | answer+proof | construction | multiple-choice
topics: [N.floor, N.divisibility]
difficulty:
  mohs: 5                        # when a source gives it
  model_estimate: {value: 0.31, method: "solve-rate:v1", at: 2026-11-02}
authors:
  - person: 9QRT4M2XKC          # proposer, when known
    role: proposer
    country: CA
languages: [en, fr, ru, vi]
canonical:
  statement: S1
  rule: official-original       # the precedence rule that chose it (06.3)
statements:
  - id: S1
    lang: en
    source: imo-official
    capture: imo-official/2024/eng.pdf#p1
    kind: official
    zone: Z2                    # statement + link (document 11)
  - id: S2
    lang: en
    source: aops-collections
    capture: aops-collections/post/32007342
    kind: transcription
    diff_from: S1               # the normalized diff is stored, and small
answer:
  form: set
  value: "\\{\\alpha \\in \\mathbb{Z} : 2 \\mid \\alpha\\}"
  sympy: "ImageSet(Lambda(k, 2*k), Integers)"
  source: official
xref:
  aops_topic: "3107211"
  compfiles: Imo2024P1
status:
  statement: verified           # captured | normalized | verified | disputed
  solutions: {human: 4, ai: 2, official: 1}
  formal: {statements: 1, proved: true}
---

<!-- S1 -->
Determine all real numbers $\alpha$ such that, for every positive integer $n$, the integer
$$\lfloor \alpha \rfloor + \lfloor 2\alpha \rfloor + \dots + \lfloor n\alpha \rfloor$$
is a multiple of $n$.

<!-- S3 lang=fr -->
...
```

Rules for the body:

- The canonical statement is first. It is the human text chosen by the rule in 06.3, after normalization only (06.4). A model may normalize markup. It may not rewrite the mathematics.
- Each other statement follows, with its id in an HTML comment. A transcription that differs from the canonical text only in markup is not stored again. Its diff is stored in the front matter.
- Figures are files next to the problem: `p1.fig1.svg`, with the original source (`p1.fig1.asy`, a scan crop, a GeoGebra file) kept beside the SVG.
- Mathematics is LaTeX between `$...$` and `$$...$$`, with the macro set of `macros/olymmath.sty` (the same subset as KaTeX supports).

## 4.4 The solution file

Solutions live at `solutions/<path of the problem without .md>/<tag>.md`, for example `solutions/imo/2024/p1/4M2XKC9QRT.md`.

```markdown
---
tag: 4M2XKC9QRT
kind: solution
problem: 7K3QX2M9R2
origin: human                    # official | human | ai | human+ai
author:
  person: null
  display: "user 'TheUltimate123' on AoPS"
source: aops-collections
capture: aops-collections/post/32007561
zone: Z2                         # excerpt + link unless the author opts in (document 11)
lang: en
method: [floor-sum, induction]  # short tags of the idea
complete: true                   # the author claims a full solution
grades:
  - by: human:reviewer/tamnd
    score: 7
    max: 7
    at: 2026-12-01
answer_claimed: "even integers"
answer_check: match              # match | mismatch | none | n/a
---

Write $\alpha = k + \beta$ ...
```

An AI solution has `origin: ai` and a `run` block: the model id, the prompt version, the date, the token cost, the run id and the transcript hash. Document 07 gives the full form. An AI solution never has `origin: human`. A human solution that a model has translated has `origin: human`, and a `translated_by` block with the model and date.

## 4.5 Captures and the raw store

A capture is the raw bytes that a source returned, plus a small record:

```json
{"source":"aops-collections","key":"post/32007342","url":"https://artofproblemsolving.com/community/c6h3107211p32007342",
 "fetched":"2026-11-03T10:22:01Z","sha256":"…","bytes":4211,"type":"text/html","adapter":"aops@v0.3.1"}
```

Captures are not in git. They are in the raw store: a content-addressed object store (Cloudflare R2, bucket `olymmath-raw`) with a Parquet index per source. A capture is never changed. A refetch that returns other bytes is a new capture with the same key and a newer date. This follows the kvant rule that the raw layer is ground truth, so that a parsing bug is fixed by a reparse and not by a refetch.

The Parquet index of each source is also published as a Hugging Face dataset, `tamnd/olymmath-captures`, when the licence zone of the source allows it (document 11). Otherwise only the index (key, URL, hash, date) is published.

## 4.6 Merges, splits and aliases

- **Merge.** Two problem records are the same problem. The record with the earlier appearance keeps its tag. The other tag goes to `registers/aliases.tsv` as `old -> new`. Its statements, solutions and xrefs move to the kept record. A merge needs one human review, or two independent automatic signals at the strict threshold of 06.5.
- **Split.** One record holds two problems (often a forum thread that posts parts (a) and (b) of different contests). The record keeps its tag for the part that matches its primary appearance. The other part gets a new tag.
- **Variant.** Two problems differ in a constant, a condition or a strengthening. They stay two problems, with a `variant_of` edge. A variant is not a duplicate.

## 4.7 Edges

Problems link to each other and to the outside with typed edges, in the front matter field `links`:

| Edge | Meaning |
|---|---|
| `variant_of` | the same problem with a changed constant or condition |
| `generalizes` / `special_case_of` | one problem implies the other |
| `uses` | a solution of this problem uses the result of that problem |
| `same_idea` | a curator says the key idea is the same |
| `answers_at` | a math2 node that the problem is about |
| `formalized_as` | a formal record |

These edges, together with appearances, topics, authors and the competition tree, are the browsing graph of the site (document 10).

## 4.8 Topics

The topic tree has four roots: `A` algebra, `C` combinatorics, `G` geometry, `N` number theory, as in the IMO shortlist. Below each root there are two more levels, for example `G.triangle.incircle`, `N.diophantine.pell`, `C.graph.extremal`, `A.inequality.schur`. The tree is in `registers/topics.yaml`. It starts from the AoPS wiki topic pages and the subject classification of Omni-MATH, and curators extend it. A problem has one to four topics. A model may propose topics. A proposal is stored with `by: ai` until a person confirms it, and the site shows it in another style.

## 4.9 Difficulty

Difficulty has no single true value. The record keeps each estimate with its method:

| Method | Range | Source |
|---|---|---|
| `mohs` | 0 to 60 in steps of 5 | Evan Chen's MOHS ratings for IMO/USAMO/ISL problems |
| `aops-rating` | 1 to 10 | the AoPS wiki "Competition ratings" scale, applied to the competition and position |
| `position` | | the problem number within its paper (P1/P4 easy, P3/P6 hard for IMO) |
| `omni-math` | 1 to 10 | the difficulty column of Omni-MATH |
| `solve-rate:v1` | 0 to 1 | the share of a fixed panel of models that solve it (07.6) |
| `contestant` | 0 to 1 | the share of contestants with full marks, from imo-official statistics |

The site shows one difficulty band, computed from the available estimates by a published rule (document 10), and shows the estimates themselves on the problem page.
