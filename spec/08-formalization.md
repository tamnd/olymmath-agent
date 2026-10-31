# 08. Formalization in Lean 4

Evidence: 01d. This is phase 3 of the roadmap (document 13), but the data model is fixed now, so that phase 1 imports can link formal records from day one.

## 8.1 What exists

| Source | Size | Kind | Licence | Import rank |
|---|---|---|---|---|
| Compfiles (David Renshaw) | 521 formalized, 515 solved (IMO 274, USAMO 217) | human statements + proofs, Lean v4.35.0 | Apache-2.0 | 1 |
| IMOSLLean4 | 171 ISL problems 2006 to 2025, all proved | human, v4.35.0 | Apache-2.0 | 2 |
| Mathlib `Archive/Imo` | 55 proved files, 1959 to 2026 | human | Apache-2.0 | 3 |
| formal-imo (Google DeepMind) | 258 IMO statements 1959 to 2024 | human statements, no proofs | Apache-2.0 | 4 |
| PutnamBench | 672 Lean (640 Isabelle, 412 Coq) | human statements; answers as `abbrev` | Apache-2.0 | 5 |
| IMO 2025 / 2026 lab repos (Axiom, Pramaana, Seed-Prover, Aristotle, …) | 6 problems each, several independent statements | AI proofs; statements by humans or AI | varies; several have none | 6 |
| miniF2F v2 and corrected forks | 488 | statements (AMC/AIME/IMO/MATH) | MIT/Apache | 7 |
| CombiBench, MathOlympiadBench, Seed `imo_all` | hundreds | statements | varies | 8 |
| NuminaMath-LEAN (104k), FormalMATH (5,560), Lean Workbook (~140k), Goedel-Pset (1.7M) | very large | AI-autoformalized | varies | 9 |

The ICML 2026 audit (arXiv:2606.29493) flagged 4,833 likely defects in 10,318 benchmark problems and confirmed 398. Formal statements are therefore treated like forum text: evidence, ranked, reviewed, never trusted by compiling alone.

## 8.2 Data model

One problem has **many formal statements**. A formal record:

```yaml
tag: QX2M9R27K3
kind: formal
problem: 7K3QX2M9R2
part: null                  # a, b … for multi-part problems
system: lean4
toolchain: leanprover/lean4:v4.35.0
mathlib: v4.35.0
path: lean/Olymmath/Imo/Y2024/P1.lean
decl: Olymmath.Imo.Y2024.P1.statement
answer:
  style: separate-def       # separate-def | embedded | none (proof problems)
  decl: Olymmath.Imo.Y2024.P1.answer
origin:
  kind: human               # human | ai | ai+human
  source: compfiles
  url: https://github.com/dwrensha/compfiles/blob/<commit>/Compfiles/Imo2024P1.lean
  commit: <sha>
  native_name: Imo2024P1
licence: Apache-2.0
port_diff: null             # the diff when ported to the pinned toolchain
fidelity:
  status: two-human          # unreviewed | checker-clean | llm-triaged | one-human | two-human | known-wrong
  reviews: [rev/2027/03/…]
  defects: []
equivalent_to: [8TQ…]       # other formal records proved equivalent
rank: 1
proofs: [proof tags]
```

Answer problems use the formal-conjectures style: `def P1.answer : Set ℝ := …` and a theorem `P1.statement : {α | …} = P1.answer`. The answer value is also in the problem's `answer` field, so that the informal and formal answers can be compared. A trivial answer that makes the theorem vacuous is rejected by a check in the style of the RPE test (01d §9.4).

## 8.3 The Lean package

`tamnd/olymmath/lean/` is a Lake package `Olymmath`. It pins Lean v4.35.0 and Mathlib v4.35.0, the same as math2, so the two packages can share a build cache and a monthly bump. The module path follows the locator: `Olymmath/Imo/Y2024/P1.lean`, `Olymmath/Putnam/Y1985/B6.lean`.

Imported files keep their original text in `lean/vendor/<source>/<commit>/` and the package files import or adapt them. A port to the pinned toolchain is a separate commit with a diff.

## 8.4 Fidelity pipeline

In order, each step recorded:

1. Compile at the pinned toolchain with `autoImplicit false`.
2. Static checkers (a port of the atp-checkers: unused hypotheses, vacuous hypotheses, contradictory hypotheses, trivially true goals).
3. Counterexample search on small cases (`plausible`, `decide` on small instances), when the statement allows it.
4. Back-translation to English and comparison with the canonical informal statement by two model judges of different families. This only triages. It never sets a fidelity status above `llm-triaged`.
5. Equivalence: try to prove the statement equivalent to the other formal records of the same problem (a BEq-style check). A proven equivalence joins them in one class.
6. Human review. A statement is shown as **faithful** only after two human reviewers sign off (`two-human`). One human gives `one-human`, shown as "reviewed once".
7. A known-wrong statement stays in the corpus with its defect and reason, so that a later import does not bring it back.

## 8.5 Proof checking

1. Check each proof with `lake env lean` on the pinned toolchain, and with the comparator against the frozen statement file, so that a proof cannot change the statement it claims to prove. Canonical proofs also pass `lean4checker` (`--paranoid`). This runs on Linux in a container (`images/lean`).
2. Allowed axioms: `propext`, `Quot.sound`, `Classical.choice`. No `sorry`. `native_decide` is allowed only with a flag, and the site shows the flag.
3. Each proof record keeps the toolchain on which it last passed. The monthly bump rechecks all proofs; a failure opens an issue and the site shows "last checked at v…".

## 8.6 Autoformalization by agents

Later, `om formalize` drafts formal statements for problems that have none, with the solver infrastructure of document 07. The order: problems with a human solution and a clear answer first; geometry last (Mathlib's Euclidean geometry is still thin; LeanGeo and LeanEuclid are watched). An AI statement enters as `unreviewed` and moves up only through 8.4. An AI proof of a `two-human` statement is a fully trusted result once it checks, because the kernel checks the proof and people checked the statement.

## 8.7 The site

The problem page shows the formal section only when a formal record exists: the top-ranked statement with its fidelity badge, the answer, the proof status, the source and a "view in Lean" link (the file on GitHub, and live.lean-lang.org with the code). Each formal record has its own page with all statements of the problem, their equivalence classes and their proofs.
