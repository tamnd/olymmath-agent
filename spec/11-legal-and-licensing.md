# 11. Legal and licensing

Evidence: 01b §6 (competition copyright), 01c Part 3 (site terms, case law), 01a (dataset licences), 01e (AI labelling). This is an engineering policy, not legal advice. Before launch a lawyer reads this document once.

## 11.1 The facts that set the policy

- **Almost no olympiad problem is under an open licence.** Organizers hold the copyright on their papers and official solutions (MAA, UKMT, CEMC, the IMO Board, national committees). ARML is the most permissive. The IMO Shortlist is confidential until the next IMO.
- **Facts are free; expression is not.** The competition, year, number, answer, topic and difficulty of a problem are facts. The wording of a statement and of a solution is expression. ETS v. Katzman (3d Cir. 1986) held test questions protected. A short mathematical statement may have thin protection, but olymmath does not rely on that.
- **Forum posts belong to their authors** (AoPS terms; Stack Exchange posts are CC BY-SA, with attribution).
- **Sites state their wishes.** AoPS robots.txt carries `Content-Signal: ai-train=no, search=yes, use=reference`-style signals. problems.ru allows reuse with a link and the problem number. Kedlaya asks for links. Evan Chen's notes are "all rights reserved" with free personal use. cut-the-knot and problems.ru set crawl delays. CMS disallows the CMO directory.
- **Dataset licences do not clean their contents.** An Apache-2.0 HF dataset of AoPS posts does not make the posts Apache-2.0. olymmath takes from a dataset only what the original source allows.

## 11.2 Licence zones

Each source in `registers/sources.yaml` has a zone for statements and a zone for solutions. A record can be in a higher zone than its source when the rights holder grants it.

| Zone | What the site publishes | What the exports carry | Default for |
|---|---|---|---|
| **Z1 full text** | the full text, with attribution and licence line | full text (HF export only if `ai_train` allows) | CC BY / CC BY-SA sources (Math SE, MathNet paper text, ProofWiki CC BY-SA on site only), public-domain texts (old papers past term), organizers and authors who granted permission, contributors' own solutions |
| **Z2 statement + metadata + link** | the statement (a short problem text published with source and link, as all problem archives do), facts, answer, a ≤ 2-line excerpt of each solution, a link to the full solution | facts, statement where the source allows, links | competition statements in general; AoPS solutions; problems.ru; MathNet official-booklet items; Kalva; imomath |
| **Z3 link only** | locator, title, facts, link | facts and link | paywalled journals (AMM, Math Magazine), Evan Chen, MOHS, MathArena (CC BY-NC-SA), sites that say no |

The statement of a competition problem is published in Z2 as the reference record of the problem, with the organizer's name and a link to the official source. When an organizer objects, the problem moves to Z3 in one commit (11.6). Official solutions are Z2 (excerpt and link) unless the organizer grants Z1.

Everything captured, whatever its zone, is kept in the private raw store for dedup, grading and audit. Nothing from it is published beyond its zone (09.3 S07, L04).

## 11.3 Licences of olymmath's own work

Per-path licensing with REUSE (`REUSE.toml` and `LICENSES/`), so that each file says what it is:

| Path | Licence |
|---|---|
| `olymmath-agent` code | Apache-2.0 |
| `olymmath` metadata: registers, appearances, edges, topics, difficulty, xrefs, tags, coverage | CC0 1.0 (facts; maximal reuse) |
| contributor solutions, reviews, rubrics, olymmath-written translations | CC BY-SA 4.0 (or CC BY 4.0 at the contributor's choice) |
| AI solutions and grades | CC BY 4.0 with the AI label (11.7) |
| Lean files written for olymmath | Apache-2.0 (as Mathlib); imported Lean keeps its licence (Compfiles Apache-2.0, formal-imo, PutnamBench MIT) |
| third-party text in Z1/Z2 | its own licence or "all rights reserved, quoted with permission/as reference", stated per record |

Contributions use the DCO (`Signed-off-by`), not a CLA.

## 11.4 Permission requests

Asking is the main way to move text from Z2 to Z1. Each request is tracked in `registers/permissions.yaml` (who, when, scope, answer, the email or letter hash).

| Holder | Ask for | Priority |
|---|---|---|
| IMO Board / imo-official.org | statements and official solutions of IMO and ISL, all languages | P0 |
| MAA AMC | AMC/AIME/USAMO/USAJMO statements and solution pamphlets (their non-commercial posting notice already covers statements) | P0 |
| AoPS | a data agreement: wiki and contest-collection export or allowlisted crawler; links back (05.6) | P0 |
| MCCME (problems.ru) | Z1 for statements and solutions with links to problems.ru numbers | P0 |
| Evan Chen | Z1 or Z2 for his IMO/USAMO notes | P1 |
| UKMT, CEMC, AMT, CMS, national committees (top 30 IMO countries) | official papers and solutions | P1 |
| KöMaL | English statements and solutions | P1 |
| AwesomeMath (Mathematical Reflections), CMS (Crux) | journal problems and solutions | P2 |
| imomath, VMF (Diễn đàn Toán học), imojp | crawling and text | P2 |
| Kedlaya | Putnam solutions as Z1 | P2 |

The request letter template is in `docs/permission-letter.md`: what olymmath is, what it will show, the attribution, the link back, the removal promise, and the non-commercial status.

## 11.5 `ai-train=no` and the Hugging Face export

The site is for reading. The Hugging Face export (`om publish hf`) is training data in practice, whatever its stated purpose. So:

- Text from a source with `ai_train: no` (AoPS, ProofWiki where flagged, any source whose holder says so) is never in the HF export. The export has the problem's facts and the link instead.
- The export is split into configs by zone (`z1-text`, `z2-facts`, `z3-links`), and each row carries `source`, `zone`, `licence`.
- The export card says what is left out and why.

Audit rule X02 enforces this.

## 11.6 Takedown and objection

- `/about/takedown/` and `takedown@olymmath.tamnd.com` (and the GitHub repo issues).
- A request from a rights holder moves the records to Z3 (or removes the text from history if they ask, by rewriting the published branch and purging the R2 objects) within 7 days. The site's acknowledgement is sent in 2 working days.
- Each action is a commit with a `takedown/<id>.yaml` record (the request, the scope, the date). The scope also goes into `registers/sources.yaml` so that a re-harvest does not bring the text back.

## 11.7 AI labelling

- Each AI solution and AI grade is labelled on the page, in the HTML (`data-origin="ai"`), in the JSON (`origin: ai`), and in the HF rows (07.5). This follows the EU AI Act Article 50 transparency duties (applying from 2 August 2026) for machine-generated text published to inform the public, and the plain promise of human first.
- An AI text that a person edited is `origin: human+ai`, with both named.
- AI translations of statements are labelled and are never canonical (06.3).
- No AI text is presented as an official or human solution; no model is credited as a problem author.

## 11.8 Privacy

olymmath stores public user names of solution authors (AoPS, MSE) as attribution. It does not store e-mail addresses, IP addresses or private messages. An author can ask to be anonymized ("an AoPS user") or removed. Contestants' results (names, scores) are not harvested beyond what the organizer publishes as hall-of-fame data, and are linked rather than copied.
