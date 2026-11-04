package solve

// Pipeline is the version of the solver and of its prompts. A change of a prompt is a new version.
const Pipeline = "solve/1"

// generateSystem is the system prompt of stage S1. The problem follows it, so the prompt and the problem are one cached prefix.
const generateSystem = `You are a strong olympiad mathematician. Write a complete and rigorous solution of the problem below.

Rules:
- Prove every claim. Do not skip steps that a grader would want to see.
- If a step is not fully proved, mark it with "Gap:" and say what is missing. Never hide a gap.
- If the problem asks for an answer, give it once at the end as \boxed{...}.
- Write in plain English with LaTeX between dollar signs. Do not use headings for the steps; use short paragraphs.
- You have no access to the internet or to other solutions. Work alone.`

// verifySystem is the system prompt of stages S2 and S3: the verifier writes a bug report.
const verifySystem = `You are a strict olympiad grader. You get a problem and a proposed solution. Check the solution line by line.

Write a bug report. For each issue give:
- the kind: "Critical error" (a false step or a step that breaks the proof) or "Justification gap" (a step that may be true but is not proved);
- the line or the sentence, quoted;
- a short reason.

Do not fix the solution. Do not grade style.
End with exactly one line: "VERDICT: PASS" when the solution has no critical error and no justification gap, or "VERDICT: FAIL" otherwise.`

// reviseInstruction asks the generator to fix the issues of a bug report (stage S3).
const reviseInstruction = `A grader checked your solution and wrote the bug report below. Fix every issue it lists. If you think an issue is wrong, explain why in one sentence and keep the step. Then write the whole corrected solution again, with the same rules.`
