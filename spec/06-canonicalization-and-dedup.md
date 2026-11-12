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

Strict signals (matcher `match/2`): a merge with two strict signals needs no review (rule D03). The strict signals are a strong key, a text match and an answer match. A text match is T ≥ 0.8. For a pair with a strong key, E ≥ 0.7 is also a text match, when one of the two problems is the nearest problem of the other by E. Here E is the cosine of the TF-IDF weights of the words and of the runs of four characters of the TeX. The reason is that two transcriptions of one problem, for example the shortlist booklet and an AoPS post, often have T below 0.3, because the TeX and the words differ in form. `om merge -pattern` uses this rule to join the records of a known duplicate pattern that two imports made apart. On the corpus of October 2026 it joined 40 pairs of a contest and its shortlist, and a check of the texts found no wrong merge. A pair below the rule waits for a review, and `om merge -review` joins it when the review accepts it.

Translations: a statement in another language is matched by E across languages and by K. The project embeds a model translation into English for matching only. The translation is not stored as a statement unless a person keeps it.

### Known duplicate patterns

These patterns are handled by rules before the general matcher:

- ISL → IMO: the shortlist and the contest share a problem. The ISL booklet says which shortlist problems became contest problems. The rule merges them by key.
- ILL → ISL or IMO: the jury picks the shortlist and the paper from the longlist of the same year, so a longlist problem can be a shortlist or a contest problem too. The rule puts the ill record in the year group of the IMO and joins the pair with two strict signals, like ISL → IMO.
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

## 6.8 Loose import

Most rows of the forum and dataset sources (hf-ai-mo-aops, hf-numina-olympiads, the olympiad parts of hf-numinamath-1-5, hf-mathnet) are human problems that cannot be placed: the row names a competition and a year, or nothing, but not a position. `om import -loose` brings them in as loose problems (4.3) and never invents a position. The rules:

- **Which rows.** Rows that the alias register locates to a competition but that have no usable number, and rows of olympiad-type sources. For hf-numinamath-1-5 only the rows whose source column is olympiads, aops_forum or amc_aime are taken. The school and synthetic sets are left out.
- **Licence and zone.** The statement keeps the zone and origin of the seed source in `registers/sources.yaml`. A source with no statement zone is not imported. Rows of a competition whose texts the project only links (document 11) are left out entirely, even as loose problems.
- **One clear problem.** A row is dropped when it is not one clear problem: too short, only a solution, a table of contents, a stub such as "prove the lemma", a request for a translation or for help, TeX that does not render, a figure the text depends on, a link to a web page for a part of the problem, or several problems in one row. HTML and BBCode that only style the text, folded blocks and notes of the poster are removed first. A row is dropped too when the cut of an answer or a solution leaves a text that asks nothing ("Given an odd number $m$." is what is left of "Given an odd number $m$. Proof: ...", where Proof stands for Prove), when it starts as a solution ("A quick solution:"), and when it is a post of news, with no formula, no question and a line of the poster about himself.
- **Dedup.** Each row is matched against every record and against the rows before it, with T, E and A of 6.5, an exact match of the normalized text, and the keyword match of the statement (`keymatch`). Texts are compared in a match form: the spacing inside formulas, TeX commands with the same meaning (`\geq` and `\ge`, `\dfrac` and `\frac`, `\left`), a mark around the letters of a segment or an angle (`\overline{BC}` and `BC`, `\angle{ABC}` and `\angle ABC`), a number in a formula or not, and the line of the author are ignored. The stored text does not change. In the first batch an IMO Shortlist problem that wrote every segment as `\overline{BC}` did not find the official text, because the grams of the marks outweighed the problem. A problem number glued to the text, as "3-There is" or "1.Let", is cut like "3. ". Two multiple choice problems of one contest often have the same choices, such as 3, 4, 5, 6 and 7, and they made different problems look alike. So when both texts have choices, T and the keyword match are taken from the texts without the choices if both are low there (T below 0.3 and a keyword match below 0.5). The other way, the choices of one problem are written in many ways, in a formula or as text, so when the texts without the choices have T ≥ 0.8, that T counts, if the keyword match of the whole texts is 0.5 or more. In the first batch this left 39 AMC problems as new records next to the record they copy. The keyword bar is there because a short stem such as "Find the largest number among the following numbers" is shared by problems whose choices differ. E is taken on the texts without the choices for the same reason: a row with its choices in a formula has many grams of the formula that the text of the content does not have, so the other problems with such choices came nearer than the problem itself. A row is the same as a record when A does not mismatch and one of these holds: the normalized texts are equal; T ≥ 0.8; or E ≥ 0.5, a keyword match of 0.7 or more, and E + F + T ≥ 1.6, with the shared-token floor of `keymatch` or E ≥ 0.95. The keyword bar is above the 0.55 of the content match, because a loose row has no competition and year to agree with: on the AoPS rows, pairs from 0.55 to 0.7, and pairs below the sum, were often two inequalities or two equations with the same letters. A row that is the same as a record becomes another statement of that record, one per source. Rows that are the same as each other become one record with several statements. A row in the review band (E ≥ 0.7, T ≥ 0.5, a keyword match of 0.35 or more with E ≥ 0.5, or an answer mismatch) is held back and listed in `reports/review/loose-held.tsv`, not imported. Two rows that name the same edition, or a row and a loose problem whose source names the same edition, are joined only on the same normalized text or T ≥ 0.8. A contest often sets one problem in two grades with one change, such as a square and a cube. In the first batch 4 of the joins of rows of one edition below T 0.8 were two problems. Such a row is held. A row that names an edition whose papers are all in the content, and that matches none of its problems, is held too, with the nearest problem of the edition: it is a copy in other words, or its label is wrong. In the first batch the 10 such IMO rows were all problems of the content, whose old text is in other words with no TeX.
- **Edition.** A row that names an edition the content has (the competition and the year) and is not the same as a record by the rule above is compared with each problem of that edition. The bar is the bar of the content match, since the competition and the year agree: a keyword match of 0.55 or more with the floors and a gap of 0.15 to the next problem of the edition, or E ≥ 0.5 with a gap of 0.2 to the next problem, and then T ≥ 0.3, a keyword match of 0.5 or E ≥ 0.8. The content may not have the problem of the row, and then the best problem of the edition is often another one on the same topic: in the first run two of 14 links with no runner-up were such pairs, with T below 0.3. E catches a row that says the problem in other words or names the variables in another way, where the keyword match is low. A pair with a keyword match of 0.35 or more, or E ≥ 0.35, that does not pass is held: IMO 2008 problem 4 on AoPS uses other letters and has a keyword match of 0.19 and E of 0.42 with the text of the content, so it waits for a review and does not become a new problem. The link is a second statement of the record, as above, never a new appearance.
- **Unplaced.** The competition and year that the rows of a record give become `unplaced` entries. A label that names a shortlist or a longlist gives no entry when the competition is not a shortlist or a longlist: the Iberoamerican shortlist is not the Iberoamerican olympiad. A label with a range of years, as in "Shortlist 1992 - 2010", gives no year.
- **Capture.** Each statement keeps `capture: <source>/<file>#row=<n>`, so that a second run knows the rows it has used.
- **Batches.** `-limit` stops after a number of new records, so that the import lands in reviewable pull requests. Each batch is sampled by hand, and the samples are kept as reviews.
