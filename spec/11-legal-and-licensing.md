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

Asking is the main way to move text from Z2 to Z1. Each request is tracked in `registers/permissions.yaml`, one row per rights holder (who, when, scope, answer, the letter hash). The rights holder is the organizer or publisher, not the site or dataset that copied the text: a request for MathNet or NuminaMath rows goes to the competitions they copy.

### 11.4.1 The permission register

`registers/permissions.yaml` is a list of rows:

| Field | Meaning |
|---|---|
| `id` | the row id, lower case letters, digits and hyphens, unique |
| `holder` | the official name of the rights holder |
| `country` | ISO 3166-1 alpha-2 code of the holder, or `international` |
| `competitions`, `sources` | the competition codes and source ids the request covers; at least one, each in its register |
| `languages` | ISO 639-1 codes of the letter; the English letter is always prepared, a translation is prepared per row when the holder works in another language |
| `contact` | `url` (the official page that shows the contact), `email` (a role address shown there), `address` (postal). Only from the holder's official site, never a personal or free-mail address |
| `scope` | what is asked: `statements`, `solutions` (booleans), `years` (`all`, `YYYY`, `YYYY-` or `YYYY-YYYY`), `uses` (`site`: full text on olymmath.tamnd.com with attribution and link; `export`: the text in the open data releases, 11.5; `licence`: an open licence such as CC BY 4.0 on the text) |
| `granted` | what was granted, in the shape of `scope`; can be less than asked |
| `priority` | `P0` to `P3` (below) |
| `state` | `draft`, `sent`, `granted`, `refused`, `no-answer` |
| `dates` | `drafted`, `sent`, `answered`, as `YYYY-MM-DD` |
| `letter` | `sha256:<hex>` of the letter as sent |
| `answer` | a short summary of the answer (and the hash of the answer mail when there is one) |
| `notes` | anything else: open terms already found, why the priority, a better contact |

`om check` reads the register and gives an error when:

- a competition code or source id is not in its register, or is in the row two times;
- a field is unknown (the register has no `zone` field), or the priority, state, country, years, use or a date is malformed, or the dates are out of order;
- the email is at a free-mail domain;
- a row past `draft` has no `dates.sent`, no `letter` and no contact; a `draft` has a sent date, an answer or a grant;
- a `granted` or `refused` row has no `dates.answered` or no `answer`; a `granted` row has no `granted` scope; a `refused` row has one.

**The register never changes a zone.** A grant is applied by its own commit to `registers/sources.yaml` (or to the statements it covers, when the grant is for some years only) that sets the zone to Z1 and the licence to `permission:<row id>`. `om check` gives an error when a source names a permission that is missing, not `granted`, or does not list the source, and a warning while a granted row's sources are still below Z1. A source in Z1 with no licence at all is an error: Z1 needs a basis, which is a licence on the source's page, the public-domain rules of 11.9, or a granted row.

Priority: a holder in the table below has at least its priority there. Otherwise, and when it gives a higher one: P0 when the holder's texts are behind 2,000 or more olymmath records; P1 for 300 or more, or for the national committee of a top-30 IMO country; P2 for 50 or more, or a holder whose terms already allow non-commercial reuse with credit; P3 for the rest. A holder whose terms already allow reuse is asked first within its priority, because the answer is likely yes.

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

The request letter template is in `docs/permission-letter.md` of this repository: what olymmath is, what it will show, the attribution, the link back, the removal promise, and the non-commercial status. It is short and plain English; a translation into the holder language is prepared per row when `languages` names one. The owner sends each letter; an agent prepares the row and the letter and never sends.

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

## 11.9 Public domain

A text is Z1 as public domain only when it is out of copyright **both in its country of first publication and in the United States**, where the site is served from Cloudflare and GitHub. The rules below are the ones olymmath applies; each is mechanical and the uncertainty of each is stated. A rule that needs a fact olymmath does not have (the death year of the author, a renewal search) leaves the text in its zone and marks it a candidate in `reports/z1-paths.md` of the content repository. As with grants, the zone moves by its own commit, on the statements of the years that the rule covers, with the licence `public-domain` and the rule in the commit message.

| Rule | Covers | Legal basis | Certainty |
|---|---|---|---|
| PD-US-1931 | any work first published before 1931, as seen from the US | 17 U.S.C. §304 (95 years from publication for works of 1923-1977) and §305 (terms run to the end of the year): works of 1930 entered the US public domain on 1 January 2026 | certain in the US; the country of origin must also pass PD-LIFE or PD-ANON |
| PD-LIFE | a work with a named author who died more than N years before 1 January of this year | life + 70: EU Directive 2006/116/EC Art. 1(1) (Hungary Act LXXVI of 1999 §31, Romania Law 8/1996, Poland Act of 4 February 1994 Art. 36, Czech Act 121/2000 §27, Germany UrhG §64), UK CDPA 1988 s.12, Russia Civil Code Art. 1281(1) (+ 4 years for authors who worked or fought in the Great Patriotic War, Art. 1281(4)), Canada Copyright Act s.6 (since 30 December 2022, not reviving works already free), Australia Copyright Act s.33, Japan Copyright Act Art. 51 (since 30 December 2018); life + 50: China Copyright Law Art. 23, Vietnam IP Law Art. 27(2); life + 60: India Copyright Act s.22; life + 100: Mexico LFDA Art. 29 | the term is certain; the death year of each problem's author is usually unknown, so this rule rarely applies to olympiad problems, which have no named author on the paper |
| PD-ANON | a work published with no named author (a committee paper, an anonymous journal problem) more than 70 years ago | EU Directive 2006/116/EC Art. 1(3) and 1(4) (70 years from lawful publication; collective works), UK CDPA s.12(3), Russia Civil Code Art. 1281(2), Hungary Act LXXVI of 1999 §31 | uncertain: if the author is later named, the term becomes life + 70 again; olympiad committees often name the problem authors in later books |
| PD-US-RENEWAL | a US work first published 1931-1963 whose copyright was not renewed | 17 U.S.C. §304(a) as in force before the Copyright Renewal Act of 1992 | applies to US works only (foreign works fall under PD-URAA); a candidate only, until a search of the Copyright Office renewal records (Catalog of Copyright Entries, 1950-1977, and the online records from 1978) shows no renewal |
| PD-URAA | the US side for foreign works of 1931 and later | 17 U.S.C. §104A restored the copyright of foreign works that were not in the public domain in their country on 1 January 1996 | a foreign work of 1931 or later is PD in the US only if it was PD at home on 1 January 1996 (for example, an anonymous work published before 1926 in the EU); otherwise it waits for its US term (95 years from publication) |
| PD-OFFICIAL | official texts excluded from copyright | Russia Civil Code Art. 1259(6), Germany UrhG §5, China Copyright Law Art. 5, Vietnam IP Law Art. 15, US 17 U.S.C. §105 (works of the US federal government only) | not applied: these provisions cover laws, decisions and official documents of a legislative, administrative or judicial kind; an olympiad paper set by a state body is very unlikely to count, and no court has held it does. Recorded so that the question is not asked again |

What this means for olymmath, as of 2026: US competitions and journals published before 1931 (the American Mathematical Monthly problems, 1894-1930) are public domain; Hungarian papers before 1931 (Eötvös/Kürschák 1894-1930, KöMaL 1894-1930) and Romanian Gazeta Matematică 1895-1930 are public domain in the US and are public domain at home when the problems are anonymous or their authors died in 1955 or earlier. Soviet olympiads of the 1930s to 1950s are left in their zone: the Russian transitional rules of 1993 and 2004, and so their status in the US under the URAA, are disputed. Post-war Eastern European olympiads (Romania from 1949, Poland from 1950, Czechoslovakia from 1952) are not public domain in the US, whatever their status at home. A source's own statement that its texts are public domain or under an open licence is not a public-domain rule: it is a licence on the page and is recorded as one.

This section is an engineering policy, not legal advice (see the top of this document).
