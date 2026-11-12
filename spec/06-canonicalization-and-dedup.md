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
- **Licence and zone.** The statement keeps the zone and origin of the seed source in `registers/sources.yaml`. A source with no statement zone is not imported. Rows of a competition whose texts the project only links (document 11) are left out entirely, even as loose problems. A row whose text matches a problem of such a competition is dropped too, and is not added to the problem as a second statement: the import adds nothing to these competitions. (Before this rule the first five batches added eight such statements; the sixth batch takes them out.)
- **One clear problem.** A row is dropped when it is not one clear problem: too short, only a solution, a table of contents, a stub such as "prove the lemma", a request for a translation or for help, TeX that does not render, a figure the text depends on, a link to a web page for a part of the problem, or several problems in one row. HTML and BBCode that only style the text, folded blocks and notes of the poster are removed first. A row is dropped too when the cut of an answer or a solution leaves a text that asks nothing ("Given an odd number $m$." is what is left of "Given an odd number $m$. Proof: ...", where Proof stands for Prove), when it starts as a solution ("A quick solution:"), and when it is a post of news, with no formula, no question and a line of the poster about himself. The numina sets added more kinds, found in the fourth batch by hand, where about one new record in six was one of them:
  - a step of a solution given as the statement, as "Since the right side is divisible by 49, ..." or "By the condition, ...", and an answer under a heading, as "# 3. Answer: ...";
  - a template, whose parameters ("param1") a table fills in, so that one row stands for several problems;
  - a text whose displayed formulas the dataset lost, so that a blank line cuts a sentence ("Prove that if the equation", blank, "has three real roots, then"), or a note of the translator says the formula is missing;
  - a problem of physics or chemistry (out of scope, document 02), found by terms such as a resistor, a piston, a refractive index or a molar mass, and not by the bare word physics, which a puzzle about a club of physics uses too;
  - a solution in the place of the statement that does not start as one: a text with two or more distinct words of the steps of a solution, such as therefore, hence, indeed, it follows, we find or let us show. A problem may use one of them; in the fourth batch nearly every text with two was a solution;
  - a sentence that stops at a word that brings in a formula ("satisfy the condition", "the following equality holds") with a blank line and a new sentence after it, which is a lost formula too, and so is a text that ends at such a word, at "Prove that" or at a colon (79 records of the fourth batch, most of them Polish rows whose formulas the dataset dropped);
  - a problem of an economics or a financial literacy olympiad (a perfectly competitive market, a demand function, an income tax deduction, a mortgage, a cadastral value, the "Full score" line of such an olympiad), out of scope like the sciences. A plain problem about a price, a discount or a sales tax is mathematics and stays;
  - a task of an olympiad of informatics or of an applied contest, marked by a heading such as "Text of the task." or "Problem statement.", a request to write a program, or terms such as input data and time limit.
  The terms of physics include those of the school olympiads that the numina sets mix in: a density of water or wood, a heater, a body floating in a liquid, spring scales. On the placed problems of the content the new terms of physics, economics and informatics match 2 texts of more than 41000, both about densities.
  - a row of the IMO compendium, marked with the country that proposed it, as "(GBR 2)" or "N7 (GBR)", or with a note that the problem was set at the IMO. Its problems are those of the shortlists, the longlists and the IMO, which the content has by position, but the row does not give the year, so the import cannot place it and its other wording often misses the record (in the fourth batch about 290 such rows became new records, many of them copies).
  - a grading scheme ("Evaluation criteria", "points are awarded for") or a note of the translator about the text ("Note: The translation preserves the original text's line breaks"), which in the fifth batch came with a statement whose formulas were often lost;
  - a row with two marks of a version on lines of their own, as "## Variant 1" and "## Variant 2", which holds two versions of the problem.
  A request to translate that a dataset left at the end of the text ("Translate the text above into English, please keep ...") is cut, and the problem before it is kept. The fifth batch found more parts of a row that only frame the problem; each is removed and the problem is kept:
  - a line with the points of the problem, as "(10 points)", "## 16 points" or "## Maximum 15 points", the points before the text, as "(3 points) Find" or "1. [4 points] Find", and the points line at the end, "Points for the problem: 13.";
  - a heading of the statement, as "## Condition:" or "Condition of the problem", and one mark of a version, as "Variant 1" or "Option 2";
  - a paragraph on the form of the answer, as "The answer to the task should be an integer";
  - a lone "#" left from a heading of the next problem (188 records of the earlier batches end with one);
  - a section number with no point after it, as "10.2 Plot the set", which is cut like "10.2.";
  - the second copy of a text that the dataset gave twice, most often a translation and the text again in other words. The two halves, split at a blank line, must be close in length, have a Jaccard index of their word 5-grams of 0.6 or more, and have the same numbers, so the parts of a problem ("a) for $n=3$", "b) for $n=4$") stay.
  A heading line at the start that is the start of the problem, as "# 3. Solve the system of equations" or "# 3. In the equation" with the equation on the next line, is kept without its "#" and its number. Before, every heading line was removed, so such a problem lost its first words. A heading that is a title, as "# Task 6. Logic" or "## 26. Gold Coins", is still removed. An answer that is a placeholder with its spaces lost ("notfound", "seesolution"), or the end of a formula whose start was lost ("=2", ")=1"), is not kept: the fifth batch had 259 such answers, and main has 225 "notfound" and 94 broken answers from the earlier batches.
  The sixth batch reached the rows of problem books (geometry, probability and calculus books, and Russian problem collections), where about one new record in ten was not a clear problem and one in three kept the number of the book. Now:
  - the number of a problem of a book is cut like a problem number: up to four digits, with stars ("1037.", "18.45*.", "170***."), "Example 3.", "(!) 11.", and the mark of a grade or a round, as "3-3." or "3-ча 2.";
  - a text that names another problem or a section of the book is dropped as one that refers to another problem: "Solve problem 1.67 using ...", "(see problem II. 296)", "See also problems 248 a) - c) from § 1", "The same question for the numbers ...". A count such as "12 problems" is not a reference;
  - a bare formula with fewer than three words around it and no word that asks for something ("Example 4. $f(x)=...$.", "345. $\sin 35^{\circ}; \cos 167^{\circ}$") is dropped as not a statement: the task was in a heading that the dataset left out. A formula that ends in "=" or "=?", a list of choices, and a verb such as "Multiply:" or "Factorize" keep it;
  - a heading of the next section in capitals at the end ("TILINGS WITH POLYGONS"), a row of stars, a request to translate in Chinese or one that says "please retain", and a note that starts "The provided translation ..." are cut, and an answer with spaced letters ("A n s w e r:") is read as the start of an answer.
  The review of the sixth batch found more marks of the books. Now:
  - "Example." with no number, "Example 1 (to item 2).", the mark of a round as "3-rd 5." and a book number that starts with 0, as "0.16.", are cut like a problem number (about 210 of the 7800 records of the batch);
  - a sign that a book repeats at a line break, so that the dataset has it twice, as "$y=$ $=\sqrt{x}$" or "$a+$ $+b$", is written once (about 200 records). It is done for `=`, `+`, `-`, `<`, `>` and `\times`, and only where a formula ends with the sign and the next one starts with it;
  - the heading of an olympiad with its year, as "17th All-Union Olympiad, 1983 (Kishinev)", and the table of its problems by class and day after it ("| Class | 1st day | 2nd day |") are cut from the end of the last problem of that olympiad (11 records). A heading that has a point, a colon or a question mark is a sentence of the problem, so a table that a problem gives is kept.
  The review of the seventh batch cleaned about 450 texts by hand, most of them for marks of the books. The import now cuts the commonest ones:
  - the number of a book with the number of an older edition after it, as "112(981)." or "37(!).", and "Ex. 119.", like a problem number (160 records of the batch had the first form);
  - a page number of up to three digits in the last paragraph, after a text that ends with a point, a question mark, a bracket or a formula, as "... for every real $x$." and then "388" in a paragraph of its own. A text that ends with a colon keeps its number, since the problem may ask about it, and a number of four digits is most often a year, so it stays too;
  - the grade or the points of a problem in the last paragraph, as "(10-11 grades)", "$(8-9$ grade $)$" or "(7-8 pts)", and the grade before the text, as "(8th grade) Prove" (83 records of main start with one).
- **Dedup.** Each row is matched against every record and against the rows before it, with T, E and A of 6.5, an exact match of the normalized text, and the keyword match of the statement (`keymatch`). Texts are compared in a match form: the spacing inside formulas, TeX commands with the same meaning (`\geq` and `\ge`, `\dfrac` and `\frac`, `\left`), a mark around the letters of a segment or an angle (`\overline{BC}` and `BC`, `\angle{ABC}` and `\angle ABC`), a number in a formula or not, and the line of the author are ignored. The stored text does not change. In the first batch an IMO Shortlist problem that wrote every segment as `\overline{BC}` did not find the official text, because the grams of the marks outweighed the problem. A problem number glued to the text, as "3-There is" or "1.Let", is cut like "3. ". Two multiple choice problems of one contest often have the same choices, such as 3, 4, 5, 6 and 7, and they made different problems look alike. So when both texts have choices, T and the keyword match are taken from the texts without the choices if both are low there (T below 0.3 and a keyword match below 0.5). The other way, the choices of one problem are written in many ways, in a formula or as text, so when the texts without the choices have T ≥ 0.8, that T counts, if the keyword match of the whole texts is 0.5 or more. In the first batch this left 39 AMC problems as new records next to the record they copy. The keyword bar is there because a short stem such as "Find the largest number among the following numbers" is shared by problems whose choices differ. E is taken on the texts without the choices for the same reason: a row with its choices in a formula has many grams of the formula that the text of the content does not have, so the other problems with such choices came nearer than the problem itself. A row is the same as a record when A does not mismatch and one of these holds: the normalized texts are equal; T ≥ 0.8; or E ≥ 0.5, a keyword match of 0.7 or more, E + F + T ≥ 1.6, and T ≥ 0.3 or E ≥ 0.8, with the shared-token floor of `keymatch` or E ≥ 0.95; or by the rule of the words below. The bar of T ≥ 0.3 or E ≥ 0.8 came from the review of the sixth batch: of the 59 links with T below 0.3 and E below 0.8, 12 were two problems in the same words with another claim or other data ("does not exceed 1" and "is at least 1", centres of mass and a regular tetrahedron), and a wrong link loses the problem of the row. Most of the other 47 were translations of a record, which now wait in the held list and lose only a link. The keyword bar is above the 0.55 of the content match, because a loose row has no competition and year to agree with: on the AoPS rows, pairs from 0.55 to 0.7, and pairs below the sum, were often two inequalities or two equations with the same letters. A row that is the same as a record becomes another statement of that record, one per source. Rows that are the same as each other become one record with several statements. A row is joined to a row before it only when the two texts have the same numbers, each as many times, and the same as the row that made the record. In a derived dataset two rows with the same words and other numbers are most often two versions of one problem, such as the years 2021 and 2022, 33 and 123 points, or a sum of 840 and 2520. The exact key and T read past one number, and a join to a join can carry the change on. In the first NuminaMath batch 50 of the 585 joins were two problems; the rule holds 25 of them, and also 23 right joins, which lose nothing since a joined row leaves no trace. Such a row is held with the reason "other numbers". The numbers count each time they appear: x^4+x^3-x^2 and x^4+3x^3+x^2 have the same set of digits and other coefficients, and so do "= 2" and "= 1" in an equation that has the digits 1 and 2 elsewhere. A join needs T ≥ 0.3 too, and a join of two rows of one file at most 10 rows apart needs T ≥ 0.8 (reasons "two rows, T" and "neighbour rows, T"): a book sets a problem and then the same problem with one change, a circle for a sphere or triangles for all polygons. In the sixth batch 39 of the 449 joins were two problems, all with T below 0.8; the three bars hold 31 of them, and 38 right joins, which lose nothing. A link to a placed record does not need the rule, because the labels and the text of the content add numbers. A link to a loose record that a derived dataset gave (origin D), or through a row that links, is a join of two rows in all but name, so it needs the same numbers too, and is held with the same reason when the numbers differ, and the bar of neighbour rows holds it too. In the fifth batch the review found such links from a row to a loose record of the same dataset with other data. A row in the review band (E ≥ 0.7, T ≥ 0.5, a keyword match of 0.35 or more with E ≥ 0.5, or an answer mismatch) is held back and listed in `reports/review/loose-held.tsv`, not imported. Two rows that name the same edition, or a row and a loose problem whose source names the same edition, are joined only on the same normalized text or T ≥ 0.8. A contest often sets one problem in two grades with one change, such as a square and a cube. In the first batch 4 of the joins of rows of one edition below T 0.8 were two problems. Such a row is held. A row that names an edition whose papers are all in the content, and that matches none of its problems, is held too, with the nearest problem of the edition: it is a copy in other words, or its label is wrong. In the first batch the 10 such IMO rows were all problems of the content, whose old text is in other words with no TeX. Two sources often write each formula of one problem in another way: `$A,B,C$` and `$A$, $B$, $C$`, a system in a bare `align` or in `$$`, `\sin x` and `\sin{x}`. E, T and the keyword match all read the formulas, so such a pair was not even compared: in the first batch about 25 AMC, AIME and Shortlist problems became new records next to the record they copy. So a row is also compared with the texts nearest by the words outside the formulas (TF-IDF cosine of 0.7 or more), and it is the same by the rule of the words when the Jaccard index of those words is 0.8 or more, each text has 10 words or more, the Jaccard index of the letters, digits and commands of the formulas (with no commands of layout, such as `\left` or `\frac`) is 0.75 or more, and the two texts have the same numbers. The numbers keep apart two problems in the same words with other data, and the floor of words keeps apart short texts such as "Find all functions $f$ such that ... for all reals $x, y$". A text with fewer than 20 words needs its formulas in nearly the same order too: a Jaccard index of 0.7 or more on the pairs of adjacent tokens of the formulas, with the signs. Without it two functional equations in the same 10 words, with the letters f, x and y in another order, were the same. On the first batch the rule linked 34 rows that were new, held or joined before, all right on a check by hand.
- **Edition.** A row that names an edition the content has (the competition and the year) and is not the same as a record by the rule above is compared with each problem of that edition. The bar is the bar of the content match, since the competition and the year agree: a keyword match of 0.55 or more with the floors and a gap of 0.15 to the next problem of the edition, or E ≥ 0.5 with a gap of 0.2 to the next problem, and then T ≥ 0.3, a keyword match of 0.5 or E ≥ 0.8. The content may not have the problem of the row, and then the best problem of the edition is often another one on the same topic: in the first run two of 14 links with no runner-up were such pairs, with T below 0.3. E catches a row that says the problem in other words or names the variables in another way, where the keyword match is low. A pair with a keyword match of 0.35 or more, or E ≥ 0.35, that does not pass is held: IMO 2008 problem 4 on AoPS uses other letters and has a keyword match of 0.19 and E of 0.42 with the text of the content, so it waits for a review and does not become a new problem. The link is a second statement of the record, as above, never a new appearance.
- **Marks in the text.** A row of the Polish olympiad starts with its mark, as "XXXVIII OM - I - Problem 11": the olympiad in Roman numerals, the round and the number. The olympiad of 1949/50 is the first, so the year of olympiad N is 1949 + N. The content has the final, round III, by position, so a row of the final whose number is a position of that year is left to the import by position. A row of round I or II has no position: the mark is cut from the text, and the row becomes a loose problem with an `unplaced` entry for pl-om and the year, with the mark as its label. The mark gives no position the content uses, so no number is made up.
- **Unplaced.** The competition and year that the rows of a record give become `unplaced` entries. A label that names a shortlist or a longlist gives no entry when the competition is not a shortlist or a longlist: the Iberoamerican shortlist is not the Iberoamerican olympiad. A label with a range of years, as in "Shortlist 1992 - 2010", gives no year.
- **Capture.** Each statement keeps `capture: <source>/<file>#row=<n>`, so that a second run knows the rows it has used. A review that removes a loose problem because it is not one clear problem leaves no capture, so the next batch would bring the row back: in the seventh batch all 31 records that the review of the sixth batch removed came back as new records. So the review adds the row to `registers/loose-removed.tsv`, one line per row with the capture, a tab and the reason, and the import drops the rows of that file with the reason "removed by a review". A row that a review moved into another record keeps its capture there and needs no line.
- **Batches.** `-limit` stops after a number of new records, so that the import lands in reviewable pull requests. Each batch is sampled by hand, and the samples are kept as reviews.
