# 14. Coverage: how we know nothing is missing

"Collect all olympiad problems" is checkable only if "all" is written down. This document defines the two universes of 06.7, the coverage measures, and the report.

## 14.1 The closed universe: the register

`registers/competitions.yaml` lists each competition with the shape of each edition:

```yaml
- code: imo
  names: {en: International Mathematical Olympiad, fr: Olympiade internationale de mathématiques}
  level: international
  organizer: IMO Board
  url: https://www.imo-official.org/
  first: 1959
  gaps: [1980]
  editions:
    default: {papers: [d1, d2], positions: {d1: [p1, p2, p3], d2: [p4, p5, p6]}}
    1959: {positions: {d1: [p1, p2, p3], d2: [p4, p5, p6]}}
  sources: [imo-official, aops-collections, compendium, mathnet, imomath]
  official_solutions: false
  shortlist: isl
- code: isl
  embargo: "until the IMO of the next year"
  editions: {default: {positions: "from booklet"}}
```

The register starts with the ~210 competitions of 01b (157 table rows plus ~56 grouped countries), each with its first year and positions. Lower rounds of national olympiads are added as separate competitions when a source holds them. The register is the denominator of every closed-universe measure.

**How we find missing competitions.** The register is not trusted to be complete. Each source adapter reports every competition label it sees (AoPS category names, NuminaMath and Omni-MATH source strings, MathNet `competition` values, problems.ru sources, imomath and imogeometry directories). A label that does not resolve through `competition-aliases.tsv` goes to the queue "unknown competition". The coverage report lists these labels with their row counts. The register is complete when this queue is empty.

## 14.2 Measures

For each edition, each competition, each country and the whole corpus:

| Measure | Definition |
|---|---|
| C0 positions | share of expected positions with a problem record |
| C1 human text | share with a canonical statement of class H1 or H2 |
| C2 official | share with an H1 statement |
| C3 verified | share with status `verified` |
| C4 human solution | share with at least one human solution (shown or linked) |
| C5 graded | share with a solution graded 7/7 (any grader) |
| C6 answer | for answer problems, share with an answer checked against an official answer |
| C7 formal | share with a formal statement of status `two-human` |
| C8 proved | share with a checked proof of a `two-human` statement |

And for the open universe, by source: candidates seen, located, matched to an existing problem, new problems created, still in a queue.

## 14.3 The report

`om coverage` writes `reports/coverage/<release>.md` and `.json`, and the site page `/coverage/`:

- a table of competitions, sorted by importance, with C0 to C8;
- for each competition, the year grid of 10.3 with each cell coloured by the best measure reached;
- the queue sizes (unknown competitions, unlocated candidates, review pairs, find-the-original);
- the change since the last release.

## 14.4 Targets by phase

| Phase | C0 | C1 | C4 | C7 |
|---|---|---|---|---|
| Wave 1 competitions (international, ISL, USA, Putnam), end of phase 1 | 100% | 100% | 98% | none |
| Top 30 IMO countries' national olympiads + TSTs, end of phase 2 | 95% | 90% | 85% | none |
| All register competitions, end of phase 3 | 90% | 80% | 70% | 10% (wave 1) |
| Wave 1, end of phase 4 | 100% | 100% | 100% (AI fills gaps) | 50% |

C0 below 100% for an old edition is acceptable only when the coverage report links the evidence that the problems are not public anywhere (for example, a national olympiad whose papers were never published). That edition is marked `unavailable`, not missing.

## 14.5 The source checklist

The source register (05.3) is the open-universe checklist. A source is **done** when: its adapter has harvested all of it (cursor at the end or a full dump read), its candidates are all located or in a queue, and the last nightly found no new items, or new items are flowing daily. The coverage report lists sources by state: planned, harvesting, ingested, live, done, blocked (with reason, for example "ToS forbids" or "site down").
