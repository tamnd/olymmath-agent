# 06. From captures to problems: parsing, matching, the human-first rule and deduplication

This document says how raw captures (04.5) become problem, statement, solution and appearance records. It is the core of the quality promise: **each problem on the site is one human-written problem, shown once, with its best human text first.**

## 6.1 The flow

```
capture ──parse──▶ candidate ──locate──▶ keyed candidate ──match──▶ problem
   (raw)      (adapter)   (text, meta)   (comp/ed/pos)     (06.5)    (new or existing)
                                                         └──▶ review queue (borderline)
```

1. **Parse.** The source adapter turns a capture into one or more candidates. A candidate has a kind (statement, solution, answer, appearance-hint), the text, the language, and every piece of metadata the source gives (competition name as written, year, problem number, author, date, thread id, HF columns).
2. **Locate.** The locator resolver maps the metadata to a locator (`imo/2024/p1`). It uses the alias table of competition names (`registers/competition-aliases.tsv`: "IMO", "International Mathematical Olympiad", "ММО", "Olimpiada Internacional de Matemática", AoPS category ids, NuminaMath `source` values, Omni-MATH `source` strings and so on). A candidate that cannot be located stays unlocated. It is still matched by content.
3. **Match.** The matcher decides whether the candidate is a new problem or a new statement or solution of an existing problem (6.5).
4. **Choose.** For each problem, the canonical statement is chosen again each time a statement is added (6.3).

Each step is a pure function of its inputs and versioned. A run of the pipeline over the same captures with the same versions gives the same records. This lets a curator fix a parser and reparse the whole raw store in hours, with no refetch.

## 6.2 Kinds of origin

Each statement and solution has an `origin_class`:

| Class | Meaning | Examples |
|---|---|---|
| H1 official | text published by the organizer | imo-official.org PDFs, MAA AMC booklets, UKMT papers, official national sites |
| H2 curated | a human transcription in a curated collection that names the official source | IMO Compendium, AoPS contest collections (moderator-posted problem threads), AoPS Wiki, problems.ru, Kedlaya's Putnam archive, Evan Chen's notes |
| H3 community | a human-written text posted by a user | AoPS forum replies, Math StackExchange, Diễn đàn Toán học, books |
| D derived | a human text taken from H1 to H3 by a dataset, with or without changes | NuminaMath rows whose `source` is `olympiads` or `aops_forum`, Omni-MATH, OlympiadBench |
| M model | written by a model | synthetic rewrites (NuminaMath `synthetic_math`, `synthetic_amc`), AI solutions |

The class of a dataset row is the class of its origin, when the origin is traceable, and D otherwise. A model-written row is M, whatever dataset holds it.

## 6.3 The human-first rule

**A problem record is created only from a candidate of class H1, H2, H3 or D.** A candidate of class M never creates a problem. M candidates can only attach as solutions (labelled AI) or as metadata (topic guesses, translations).

The canonical statement of a problem is chosen by this precedence. The first rule that applies wins:

1. `official-original`: an H1 text in the original language of the competition.
2. `official-translation`: an H1 text in English (or the official translation the organizer published).
3. `curated`: an H2 text, in order IMO Compendium, organizer-linked archives, problems.ru, Kedlaya archive, AoPS Wiki, AoPS contest collection post, other H2.
4. `community`: the earliest H3 text that the matcher links with high confidence to the appearance.
5. `derived`: a D text. The site marks it "text from dataset X; original source not yet found", and the problem goes to the queue "find the original".

A source that changed the wording of each problem on purpose, such as Kalva, has `reworded: true` in the source register. Its texts come after all other texts, whatever their class, so a text with the original wording wins. When a reworded text is not canonical, its key and its answer decide alone: a low text match is expected, so it is kept as a link unless its answer disagrees.

The site shows the canonical statement in the language of the reader when one exists at the same rank, and otherwise the canonical statement with a translation below it. A model translation is always labelled as such.

## 6.4 Normalization

Normalization makes texts comparable and renderable. It does not change mathematics. It is deterministic code, not a model, except in step 6 below.

1. Unicode NFC; typographic quotes and dashes to a fixed set.
2. LaTeX: expand the project macro set; `\[...\]` to `$$...$$`; `\(...\)` to `$...$`; remove `\displaystyle`, `\;`, and sizes that have no meaning; fix common AoPS forms (`[asy]` blocks become figure files; `[hide]` blocks are removed from statements and kept as notes).
3. Problem markers: remove leading "Problem 3.", "P3)", "3." when they repeat the locator.
4. Whitespace: one space; one blank line between paragraphs.
5. Numbers and variable names are kept as written.
6. OCR and PDF texts (document 05.6) go through a model **only** to fix extraction errors, under the "no rewrite" rule: the output must have a character-level edit distance to the input below a set share, and each change to a number, a variable or an operator is flagged for review. This follows the kvant textguard design.

A normalized statement keeps a pointer to its raw text and the normalizer version.

## 6.5 Matching and deduplication

The same problem arrives many times: IMO 2024 P1 is on imo-official, in the AoPS collection, on the AoPS wiki, in the IMO Compendium, in NuminaMath (more than once), in Omni-MATH, in OlympiadBench, in Compfiles and in miniF2F-style sets. Each must attach to one record.

The matcher combines four signals:

| Signal | How | Strength |
|---|---|---|
| **K, key** | both candidates locate to the same appearance (`imo/2024/p1`) | strong when both locators come from H1/H2 metadata; weak when from dataset free text |
| **T, text** | MinHash over word 5-grams of the normalized text with LaTeX tokens kept, then Jaccard on candidates from LSH buckets | strong at Jaccard ≥ 0.8 |
| **E, embedding** | cosine similarity of a math-aware text embedding of the statement, over an ANN index | used to find candidates that text misses: translations, rewordings |
| **A, answer** | both have answers and the answers are equal after symbolic normalization (math-verify / SymPy) | confirms; a mismatch is a strong signal of a variant |

The decision:

- **Same, automatic**: K strong, or T ≥ 0.8 with A not mismatching, or K weak plus E ≥ 0.92 plus A match.
- **Review**: E ≥ 0.85 without a strong second signal; any pair where T or E says "same" and A says "different" (likely a variant, 4.6); any candidate whose key points to an appearance that already has a problem with T < 0.5 (the key or the text is wrong).
- **New**: no candidate passes the review thresholds.

Thresholds are fixed per matcher version and tuned on a labelled set of 2,000 pairs (document 09.4). A merge that a person reverses becomes a labelled pair.

Translations: a statement in another language is matched by E across languages and by K. The project embeds a model translation into English for matching only. The translation is not stored as a statement unless a person keeps it.

### Known duplicate patterns

These patterns are handled by rules before the general matcher:

- ISL → IMO: the shortlist and the contest share a problem. The ISL booklet says which shortlist problems became contest problems. The rule merges them by key.
- TST reuse: many TSTs use ISL problems before the shortlist is public. AoPS threads of TSTs often link the ISL thread. The rule uses the link.
- AMC 10 / AMC 12 shared problems: the same problem appears as AMC 10A P20 and AMC 12A P15. The AoPS wiki marks the pair ("Duplicate" sections). The rule merges them.
- Grade levels: All-Russian and other national olympiads repeat a problem across grades. The rule merges by text within one edition.
- Dataset re-copies: NuminaMath, Omni-MATH, OlympiadBench and others copy the same AoPS texts. The rule uses their source columns to find the origin and the matcher does the rest.
- Parts: (a)/(b) parts stay in one problem when they were set together. Two problems that a dataset glued together are split (4.6).

## 6.6 Solutions from captures

A solution candidate is attached to the problem of its thread or row. Rules:

- An AoPS thread: the first post is the statement (H2 when posted in a contest collection). Replies are H3 solution candidates. A reply is kept as a solution when a classifier says it is a solution attempt (not a comment, not a hint request) and it is longer than a minimum. Very short replies ("answer is 5") become answer evidence, not solutions.
- The AoPS likes count and the poster rank are kept as metadata, not as a grade.
- A dataset solution whose origin is a human solution (for example NuminaMath rows that copy AoPS solutions and reformat them) is D. When the matcher finds the human original, the original wins and the dataset copy becomes an xref.
- A dataset solution written by a model is M and shown as AI, with the dataset as its source and no grade until graded (document 07).

## 6.7 What makes the corpus complete

Completeness has two meanings, and the coverage report (document 14) measures both:

- **Known-universe completeness.** For each competition in the register, the expected set of appearances is known (the number of problems per paper per edition). The report shows, for each edition, how many positions have a problem record, a canonical H1/H2 statement, a human solution and an answer. A missing position is a visible hole, not a silent one.
- **Open-universe completeness.** Problems from forums, magazines and books have no fixed list. The report counts them by source and tracks how many unlocated D candidates remain.
