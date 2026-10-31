# 02. The goal

## 2.1 One sentence

**olymmath is the complete, deduplicated, source-traced corpus of olympiad mathematics: every problem ever set at an olympiad-level competition, with its best human text, every human solution we may show or link, honestly labelled AI solutions, and, over time, faithful Lean statements and checked proofs, published as a fast, linkable website at https://olymmath.tamnd.com and as open data.**

## 2.2 What counts as an olympiad problem

A problem is in scope when it meets one of these:

1. **Register competitions.** It was set in a competition of the register (`registers/competitions.yaml`, document 05.2): international and regional olympiads, national olympiads and their final rounds, team selection tests, shortlists and longlists, the strong short-answer contests (AMC 10/12, AIME, HMMT, PUMaC, ARML, …), undergraduate competitions (Putnam, IMC, Schweitzer, SEEMOUS, …), and the strong regional and invitational contests (Tournament of Towns, Sharygin, ELMO, USEMO, Tuymaada, RMM, …).
2. **Olympiad problem sections** of journals: Kvant Задачник, KöMaL (A and B problems; the C and K problems are school level and are kept but marked `level: school`), Crux Mathematicorum, Mathematical Reflections, Mathematical Excalibur, AMM problems, Gazeta Matematică, Tzaloa, Eureka!.
3. **Olympiad-level forum and book problems** with a human author: the AoPS olympiad forums (not the contest collections only), Diễn đàn Toán học, Math StackExchange `contest-math`, and well-known problem books (metadata and links only, document 11).

Out of scope: school textbook exercises, gaokao and K-12 exam items (kept only as a negative filter for dedup), MATHCOUNTS below the national round (kept in a separate `level: school` band, not in the default views), Kangaroo multiple choice below the top level, programming olympiads, physics and other sciences, research problems (they belong to math2), and every synthetic problem.

## 2.3 The quality promise

1. **Human-written first.** Each problem record comes from a human text. No synthetic problem enters the corpus. Models may parse, translate, classify, solve and grade, and every model output is labelled (06.3, 07.5).
2. **One problem, one page.** The same problem from 10 sources is one record with 10 sources (06.5).
3. **Traced.** Each statement and solution names its source, its capture and its licence. The official source is found whenever it exists.
4. **Complete and honest about gaps.** For each competition in the register, the coverage report shows every expected position, filled or empty (document 14).
5. **Checked.** The audit (09.3) runs on each change, and the quality report measures accuracy by sampling (09.4).

## 2.4 The reference scale

From the research (01a, 01b, 01c):

| Part | Estimated distinct problems | Basis |
|---|---|---|
| register competitions | ~120,000 (90k to 160k), ~50,000 of them proof problems | bottom-up count over ~210 competitions (01b §9) |
| journal problem sections | ~20,000 to 40,000 at olympiad level | Crux ~5k, Kvant ~3k, KöMaL 20 to 30k (part olympiad level), AMM ~12k, Reflections ~3k (01b §5) |
| forum, translated Chinese and other open-universe sources | ~60,000 to 150,000 after dedup and level filter | AoPS olympiad forums (~125k problems in OpenMathReasoning's HS olympiad category), NuminaMath `olympiads` (197k rows, partly school level), Diễn đàn Toán học, MSE (01a, 01c) |
| **total target** | **~200,000 to 300,000 problems** | |
| human solutions | ~1 to 2 million, most of them AoPS replies | |

Cross-checks: MathNet has 30,676 problems from official booklets; AI-MO/aops has 80,661 AoPS problems; NuminaMath-1.5 has about 316k non-synthetic olympiad-like rows before dedup. The total is planned at 400,000 problems for capacity (storage, site, search), with margin.

## 2.5 Who it is for

- **Students and coaches**: browse by competition, topic and difficulty; practice with spoilers hidden; read several solutions.
- **Problem writers and historians**: find where a problem first appeared, its variants, its author.
- **AI researchers**: clean, deduplicated, provenance-labelled, licence-split data; contamination dates; formal statements with fidelity status; a public record of model solutions with graded results.
- **Formalizers**: the list of problems without faithful Lean statements, ranked by value.

## 2.6 Non-goals

- olymmath is not a forum. Discussion stays on AoPS and elsewhere; the site links to it.
- olymmath does not republish content whose licence forbids it. It links (document 11).
- olymmath does not invent problems or "improve" human statements.
- olymmath is not a benchmark. It can export benchmark splits (post-cutoff problems, for example), but it does not keep secret test sets.

## 2.7 Relation to other tamnd projects

- **math2**: a problem that is a named theorem or that a math2 node is about gets an `answers_at` edge. math2's `problem` kind covers open problems; olymmath covers solved competition problems. The two share tags, the Lean pin and the KaTeX renderer.
- **kvant**: the Kvant Задачник problems come from the kvant corpus, with their kvant ids, not from a new harvest.
- **The \*-cli tools**: each source adapter imports the CLI library for its site (03.3).
