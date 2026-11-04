---
tag: RG3MNR0R50
kind: solution
problem: 0HVHSDNNMY
origin: ai
author:
  display: claude-opus-5-5
zone: Z1
lang: en
complete: true
answer_claimed: All even integers $\alpha$.
answer_check: match
grade:
  scale: imo-7
  score: 6
  bucket: almost
  rubric: imo/2024/p1@r2
  at: 2026-11-03
  graders:
    - {by: 'model:claude-sonnet-5-5', score: 7}
    - {by: 'model:claude-haiku-5-5', score: 6}
  rule: min-of-jury-v1
  run_id: grade-20261103-0001
run:
  model: claude-opus-5-5
  prompt: solve/1
  pipeline: solve/1
  date: 2026-11-03
  run_id: solve-20261103-0001
  transcript: 4f1c0e1b2a3d4c5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d
  tokens: {input: 12000, output: 30000}
  cost: 0.66
  cutoff: 2026-01
---
Write $\alpha = 2m + \beta$ with $m$ an integer and $0 \leq \beta < 2$. The sum changes by a multiple of $n$, so we may take $\alpha = \beta$.
