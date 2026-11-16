# 01f. Research: census of the Americas and the international school contests

Research date: 2026-11-16. Scope: the competitions of the United States, Canada, Mexico, Central America, the Caribbean and South America at elementary, junior and high-school level, and the international and multinational school contests that are not tied to one region. Phase 1 of the census: register and sources only, with no harvest of full texts.

## 0. How to read this note

- **[P]** means that the competition has a primary source in `registers/sources.yaml`: the organizer's own site or files.
- **[S]** means that the only known source is secondary: the AoPS collections, the AoPS wiki or a dataset.
- **none** means that no source of any kind was found. Such a competition has a gap with the reason no-source.
- All terms pages and robots.txt files were read on 2026-11-16 unless a row says otherwise.
- The level follows spec 05: elementary is grades 1 to 5, junior is grades 6 to 9, high-school is grades 10 to 12 and contests open to all secondary students. A contest with papers for several levels gets `levels`, and `level` is the highest one.

## 1. Results in numbers

- 107 new competitions: 40 in the United States, 13 in Canada, 10 international, 10 in Mexico, 7 in Brazil, 5 in Chile, 3 each in Argentina, Ecuador, Peru and Bolivia, 2 each in Paraguay and Colombia, and 1 each in Costa Rica, Honduras, El Salvador, Panama, Puerto Rico and Uruguay.
- By highest level: 77 high-school, 27 junior, 2 elementary, 1 undergraduate. By every level they cover: 14 have elementary papers, 38 have junior papers.
- Source marks of the new competitions: 44 [P], 61 [S] and 2 with no source. By region: international 7 [P], 2 [S], 1 none; United States 15 [P], 25 [S]; Canada 11 [P], 2 [S]; Latin America and the Caribbean 11 [P], 32 [S], 1 none.
- Updated rows: `kangaroo` (level, levels, Math Kangaroo USA source), `tot` (levels), `br-obmep` (level, levels, url, source), `pe-mo`, `co-mo` and `ve-mo` (organizer and primary source).
- 26 new sources: 4 international, 12 United States, 10 Latin America. By zone: 20 Z2 and 6 Z3. Updated sources: `mathcounts-official` (now Z3), `cemc`, `oma`, `aops-collections`, `aops-wiki`, `hmmt-official` and `cmimc-official`.
- 3 gap entries: 1 robots (`noetic`), 2 no-source (`pmwc`, `mx-omm-state`).
- Expected positions added: 0. No new row has a confirmed shape, so no row has `per_year` or a layout. The rule is "When the shape is unknown, use an era with no layout; do not guess positions." Phase 2 must read sample papers to set the shapes.

## 2. Source rules that this census followed

1. A site whose terms forbid copying or reuse is Z3, link only: MOEMS, Math Prize for Girls, SEAMO, MATHCOUNTS and the Colombian olympiads of the Universidad Antonio Nariño. Math League and SIMCC sell their papers, so they are Z3 by agent decision.
2. A site with a copyright line but no licence and no ban on reuse is Z2, the spec default for competition statements (spec 11).
3. A path that robots.txt disallows is not harvested. The Noetic Learning papers are PDF files, and the robots.txt disallows `/*.pdf$`, so `noetic` has a gap with the reason robots.
4. Some sites allow all agents but block named AI crawlers (ClaudeBot, anthropic-ai, claude-web, GPTBot, CCBot, Google-Extended, Bytespider, cohere-ai, meta-externalagent) and say `Content-Signal: search=yes,ai-train=no,use=reference`. This is the Cloudflare managed robots.txt. It applies to artofproblemsolving.com, hmmt.org, simcc.org, sasmo.simcc.org and holbrook.bcamathteam.org. The owner has not decided the rule for these sites yet. The notes of each source record the facts and say that the decision is pending. No gap was written for them.
5. No login, paywall, Cloudflare challenge or robots rule was bypassed, and no full text was harvested.

## 3. The competitions

### 3.1 International

| Code | Competition | Country | Level | Levels | First year | Sources | Tag |
|---|---|---|---|---|---|---|---|
| `may-olympiad` | May Olympiad | intl | junior |  | 1995 | oma, hf-mathnet, aops-collections | [P] |
| `gqmo` | Global Quarantine Mathematical Olympiad | intl | high-school |  | 2020 | aops-collections | [S] |
| `mathlinks-contest` | MathLinks Contest | intl | high-school |  |  | aops-collections | [S] |
| `imas` | International Mathematics Assessments for Schools | intl | junior | elementary, junior |  | chiuchang | [P] |
| `emic` | Elementary Mathematics International Contest | intl | elementary |  |  | chiuchang | [P] |
| `iwymic` | Invitational World Youth Mathematics Intercity Competition | intl | junior |  | 1999 | chiuchang | [P] |
| `pmwc` | Primary Mathematics World Contest | intl | elementary |  | 1997 |  | none |
| `seamo` | Southeast Asian Mathematical Olympiad | intl | high-school | elementary, junior, high-school | 2016 | seamo-official | [P] |
| `sasmo` | Singapore and Asian Schools Math Olympiad | intl | high-school | elementary, junior, high-school | 2006 | simcc | [P] |
| `wmi` | World Mathematics Invitational | intl | high-school | elementary, junior, high-school |  | wmi-official | [P] |

### 3.2 United States

| Code | Competition | Country | Level | Levels | First year | Sources | Tag |
|---|---|---|---|---|---|---|---|
| `purple-comet` | Purple Comet! Math Meet | US | high-school | junior, high-school | 2003 | purple-comet | [P] |
| `moems` | Math Olympiads for Elementary and Middle Schools | US | junior | elementary, junior | 1979 | moems | [P] |
| `noetic` | Noetic Learning Math Contest | US | junior | elementary, junior | 2007 | noetic | [P] |
| `math-league` | Math League contests | US | high-school | elementary, junior, high-school | 1977 | mathleague | [P] |
| `usamts` | USA Mathematical Talent Search | US | high-school |  | 1989 | usamts | [P] |
| `imts` | International Mathematical Talent Search | US | high-school |  |  | aops-collections | [S] |
| `mpfg` | Math Prize for Girls | US | high-school |  | 2009 | mathprize | [P] |
| `mpfg-olympiad` | Math Prize for Girls Olympiad | US | high-school |  | 2010 | mathprize | [P] |
| `bamo` | Bay Area Mathematical Olympiad | US | high-school | junior, high-school | 1999 | bamo, hf-mathnet, aops-collections | [P] |
| `bmc-monthly` | Berkeley Math Circle Monthly Contest | US | high-school |  |  | hf-mathnet | [S] |
| `math-hour` | Math Hour Olympiad | US | high-school | junior, high-school |  | mathhour-uw, aops-collections | [P] |
| `umd-hsmc1` | University of Maryland High School Mathematics Competition, Part I | US | high-school |  |  | aops-collections | [S] |
| `umd-hsmc2` | University of Maryland High School Mathematics Competition, Part II | US | high-school |  |  | aops-collections | [S] |
| `kettering` | Kettering University High School Mathematics Olympiad | US | high-school |  |  | aops-collections | [S] |
| `utah-mo` | Utah Mathematical Olympiad | US | high-school |  |  | aops-collections | [S] |
| `bmmt` | Berkeley mini Math Tournament | US | junior |  | 2012 | berkeley-mt, aops-collections | [P] |
| `cmwmc` | Carnegie Mellon Winter Mathematics Competition | US | junior | junior, high-school | 2022 | cmimc-official, aops-collections | [P] |
| `chmmc` | Caltech Harvey Mudd Math Competition | US | high-school |  |  | aops-collections | [S] |
| `dmm` | Duke Math Meet | US | high-school |  |  | aops-collections | [S] |
| `rmt` | Rice Math Tournament | US | high-school |  |  | aops-collections | [S] |
| `asmt` | A-Star Math Tournament | US | high-school |  | 2015 | aops-collections | [S] |
| `lmt` | Lexington Math Tournament | US | junior |  | 2010 | lhsmath, aops-collections | [P] |
| `glmt` | Girls' Lexington Math Tournament | US | junior |  | 2025 | lhsmath, aops-collections | [P] |
| `sdmo-hs` | San Diego Math Olympiad, High School | US | high-school |  |  | aops-collections | [S] |
| `sdmo-ms` | San Diego Math Olympiad, Middle School | US | junior |  |  | aops-collections | [S] |
| `sdpc` | San Diego Power Contest | US | high-school |  |  | aops-collections | [S] |
| `harvest` | Harvest Math Invitational | US | high-school |  | 2020 | aops-collections | [S] |
| `mbmt` | Montgomery Blair Math Tournament | US | junior |  |  | aops-collections | [S] |
| `jhmt` | Johns Hopkins Math Tournament | US | high-school | junior, high-school |  | aops-collections | [S] |
| `jhmmc` | Joe Holbrook Memorial Math Competition | US | junior | elementary, junior |  | holbrook, aops-collections | [P] |
| `emcc` | Exeter Math Club Competition | US | junior |  |  | aops-collections | [S] |
| `mmaths` | Math Majors of America Tournament for High Schools | US | high-school |  |  | aops-collections | [S] |
| `moaa` | Math Open At Andover | US | junior |  |  | aops-collections | [S] |
| `cca-bonanza` | CCA Math Bonanza | US | high-school |  |  | aops-collections | [S] |
| `usmca` | US Math Competition Association | US | high-school |  | 2019 | aops-collections | [S] |
| `harker` | Harker Math Invitational | US | junior |  |  | aops-collections | [S] |
| `manhattan-mo` | Manhattan Mathematical Olympiad | US | high-school |  |  | aops-collections | [S] |
| `ommc` | Online Monmouth Math Competition | US | high-school |  |  | ommc-official, aops-collections | [P] |
| `nimo-summer` | NIMO Summer Contest | US | high-school |  |  | aops-collections | [S] |
| `itest` | iTest | US | high-school |  |  | aops-collections | [S] |

### 3.3 Canada

| Code | Competition | Country | Level | Levels | First year | Sources | Tag |
|---|---|---|---|---|---|---|---|
| `ca-gauss` | Gauss Mathematics Contests | CA | junior |  |  | cemc | [P] |
| `ca-pascal` | Pascal Mathematics Contest | CA | junior |  |  | cemc | [P] |
| `ca-cayley` | Cayley Mathematics Contest | CA | high-school |  |  | cemc | [P] |
| `ca-fermat` | Fermat Mathematics Contest | CA | high-school |  |  | cemc | [P] |
| `ca-fryer` | Fryer Mathematics Contest | CA | junior |  |  | cemc | [P] |
| `ca-galois` | Galois Mathematics Contest | CA | high-school |  |  | cemc | [P] |
| `ca-hypatia` | Hypatia Mathematics Contest | CA | high-school |  |  | cemc | [P] |
| `ca-cimc` | Canadian Intermediate Mathematics Contest | CA | high-school | junior, high-school |  | cemc | [P] |
| `ca-csmc` | Canadian Senior Mathematics Contest | CA | high-school |  |  | cemc | [P] |
| `ca-ctmc` | Canadian Team Mathematics Contest | CA | high-school |  |  | cemc | [P] |
| `ca-comc` | Canadian Open Mathematics Challenge | CA | high-school |  |  | cms | [P] |
| `ca-ahsmc` | Alberta High School Mathematics Competition | CA | high-school |  | 1957 | aops-collections | [S] |
| `ca-csmo` | Canadian Students Math Olympiad | CA | high-school |  |  | aops-collections | [S] |

### 3.4 Latin America and the Caribbean

| Code | Competition | Country | Level | Levels | First year | Sources | Tag |
|---|---|---|---|---|---|---|---|
| `mx-omm-state` | Mexican state mathematical olympiads | MX | high-school |  |  |  | none |
| `mx-ommeb` | Mexican Mathematical Olympiad for Basic Education | MX | junior | elementary, junior | 2017 | ommeb-cimat | [P] |
| `mx-femenil` | Mexican Girls' Contest | MX | high-school |  |  | aops-collections | [S] |
| `mx-regional-center` | Regional Mathematical Olympiad of the Center Zone of Mexico | MX | high-school |  |  | aops-collections | [S] |
| `mx-regional-west` | Regional Mathematical Olympiad of Western Mexico | MX | high-school |  |  | aops-collections | [S] |
| `mx-regional-southeast` | Regional Mathematical Olympiad of Southeast Mexico | MX | high-school |  |  | aops-collections | [S] |
| `mx-regional-northeast` | Regional Mathematical Olympiad of Northeast Mexico | MX | high-school |  |  | aops-collections | [S] |
| `mx-regional-northwest` | Regional Mathematical Olympiad of Northwest Mexico | MX | high-school |  |  | aops-collections | [S] |
| `mx-quarantine` | Mexican Quarantine Mathematical Olympiad | MX | high-school |  | 2020 | aops-collections | [S] |
| `mx-ommu` | Mexican University Mathematical Olympiad | MX | undergraduate |  |  | aops-collections | [S] |
| `cr-tst-oim` | Costa Rica Iberoamerican Team Selection Test | CR | high-school |  |  | hf-ai-mo-aops | [S] |
| `hn-mo` | Honduran Mathematical Olympiad | HN | high-school |  |  | aops-collections | [S] |
| `sv-correspondence` | El Salvador Correspondence Olympiad | SV | high-school |  |  | aops-collections | [S] |
| `pa-mo` | Panamanian Mathematical Olympiad | PA | high-school | elementary, junior, high-school |  | opm | [P] |
| `pr-tst` | Puerto Rico Team Selection Test | PR | high-school |  |  | hf-ai-mo-aops | [S] |
| `br-obm-n1` | Brazilian Mathematical Olympiad, level 1 | BR | junior |  |  | obm | [P] |
| `br-obm-n2` | Brazilian Mathematical Olympiad, level 2 | BR | junior |  |  | obm, aops-collections | [P] |
| `br-tm2` | Girls in Mathematics Tournament | BR | high-school | junior, high-school | 2019 | tm2, aops-collections | [P] |
| `br-rj-mo` | Rio de Janeiro Mathematical Olympiad | BR | high-school |  |  | aops-collections | [S] |
| `ar-nandu` | Ñandú | AR | junior | elementary, junior |  | oma | [P] |
| `cl-junior` | Chilean Junior Mathematical Olympiad | CL | junior |  |  | aops-collections | [S] |
| `cl-qualifier` | Chilean Mathematical Olympiad, qualifying round | CL | high-school |  |  | aops-collections | [S] |
| `cl-qualifier-junior` | Chilean Junior Mathematical Olympiad, qualifying round | CL | junior |  |  | aops-collections | [S] |
| `ec-mo` | Ecuadorian Mathematical Olympiad | EC | high-school |  |  | omec, aops-collections | [P] |
| `ec-junior` | Ecuadorian Junior Mathematical Olympiad | EC | junior |  |  | omec, aops-collections | [P] |
| `ec-sedem` | SEdeM Mathematical Olympiad | EC | high-school |  |  | sedem | [P] |
| `py-mo` | Paraguayan Mathematical Olympiad | PY | high-school |  |  | omapa, aops-collections | [P] |
| `py-junior` | Paraguayan Junior Mathematical Olympiad | PY | junior | elementary, junior |  | omapa, aops-collections | [P] |
| `co-junior` | Colombian Junior Mathematical Olympiad | CO | junior |  |  | aops-collections | [S] |
| `br-tst` | Brazilian IMO Team Selection Test | BR | high-school |  |  | hf-ai-mo-aops | [S] |
| `br-conosur-tst` | Brazilian Cono Sur Team Selection Test | BR | high-school |  |  | hf-ai-mo-aops | [S] |
| `br-egmo-tst` | Brazilian EGMO Team Selection Test | BR | high-school |  |  | hf-ai-mo-aops | [S] |
| `ar-conosur-tst` | Argentine Cono Sur Team Selection Test | AR | high-school |  |  | hf-ai-mo-aops, hf-aops-wiki | [S] |
| `ar-ibero-tst` | Argentine Iberoamerican Team Selection Test | AR | high-school |  |  | hf-ai-mo-aops, hf-aops-wiki | [S] |
| `cl-tst` | Chilean IMO Team Selection Test | CL | high-school |  |  | hf-ai-mo-aops | [S] |
| `cl-ibero-tst` | Chilean Iberoamerican Team Selection Test | CL | high-school |  |  | hf-ai-mo-aops | [S] |
| `pe-tst` | Peruvian IMO Team Selection Test | PE | high-school |  |  | hf-ai-mo-aops | [S] |
| `pe-egmo-tst` | Peruvian EGMO Team Selection Test | PE | high-school |  |  | hf-ai-mo-aops, hf-aops-wiki | [S] |
| `pe-pagmo-tst` | Peruvian PAGMO Team Selection Test | PE | high-school |  |  | hf-ai-mo-aops | [S] |
| `co-tst` | Colombian Team Selection Test | CO | high-school |  |  | hf-ai-mo-aops, hf-aops-wiki | [S] |
| `bo-tst` | Bolivian IMO Team Selection Test | BO | high-school |  |  | hf-ai-mo-aops, hf-aops-wiki | [S] |
| `bo-conosur-tst` | Bolivian Cono Sur Team Selection Test | BO | high-school |  |  | hf-ai-mo-aops, hf-aops-wiki | [S] |
| `bo-ibero-tst` | Bolivian Iberoamerican Team Selection Test | BO | high-school |  |  | hf-ai-mo-aops, hf-aops-wiki | [S] |
| `uy-tst` | Uruguayan IMO Team Selection Test | UY | high-school |  |  | hf-ai-mo-aops | [S] |

## 4. The new sources

Each source below is primary. The robots.txt column gives the facts read on 2026-11-16. The terms column quotes or sums up the terms page.

| Source | URL | Zone | AI training | robots.txt (2026-11-16) | Terms | Tag |
|---|---|---|---|---|---|---|
| `chiuchang` | https://chiuchang.org/imas/ | Z2 | unknown | 404, no rules | The site has no page of terms and gives no licence (read on 2026-11-16). The footer says "copyright 九章數學出版社、九章數學基金會版權所有" | [P] |
| `wmi-official` | https://www.wminv.org/ | Z2 | unknown | 404, no rules | The site has no page of terms and gives no licence (read on 2026-11-16). The footer says "Copyright © World Mathematics Invitational Organizing Committee" | [P] |
| `seamo-official` | https://www.seamo-official.org/ | Z3 | unknown | 404, no rules; terms forbid automatic gathering | The terms at https://seamo-official.org/page/terms_and_conditions (read on 2026-11-16) forbid copying or printing the content without permission, and forbid ... | [P] |
| `simcc` | https://sasmo.simcc.org/ | Z3 | no | all agents allowed; ClaudeBot and 8 other AI crawlers disallowed; ai-train=no (pending) | The home page of sasmo.simcc.org gave 403 on 2026-11-16, so its terms could not be read | [P] |
| `purple-comet` | https://purplecomet.org/ | Z2 | unknown | no blocking rules | The site has no page of terms and gives no licence (read on 2026-11-16). The footer says "All content copyright © 2003-2026 by Jonathan Kane, Titu Andreescu,... | [P] |
| `moems` | https://www.moems.org/ | Z3 | unknown | Shopify default (shop paths only) | The terms at https://www.moems.org/pages/terms-of-service (read on 2026-11-16) say "You agree not to reproduce, duplicate, copy ... without express written p... | [P] |
| `noetic` | https://www.noetic-learning.com/mathcontest/ | Z2 | unknown | disallows /mathcontest/documents/ and /*.pdf$ (gap: robots) | The site has no page of terms that covers the papers (read on 2026-11-16) | [P] |
| `mathleague` | https://www.mathleague.com/ | Z3 | unknown | Joomla default | No page of terms was found (read on 2026-11-16). Math League sells the past contests in books and as files | [P] |
| `mathkangaroo-usa` | https://mathkangaroo.org/mks/ | Z2 | unknown | 404, no rules | The site has no page of terms for the papers (read on 2026-11-16). The footer says "Copyright © 2003-2026 Math Kangaroo USA, NFP, Inc. All rights reserved" | [P] |
| `usamts` | https://www.usamts.org/ | Z2 | unknown | allows all | The site has no page of terms of its own (read on 2026-11-16). The footer says "© Copyright 2026 Art of Problem Solving Initiative. All rights reserved." Age... | [P] |
| `bamo` | https://www.bamo.org/ | Z2 | unknown | allows all | The site has no page of terms and no copyright line on the home page (read on 2026-11-16) | [P] |
| `mathprize` | https://mathprize.atfoundation.org/ | Z3 | unknown | disallows /assets/, /forms and data paths | The terms at https://mathprize.atfoundation.org/misc/terms (read on 2026-11-16) say "You shall not duplicate, modify, copy, or post any part of the Site or t... | [P] |
| `mathhour-uw` | https://sites.math.washington.edu/~mathcircle/olympiad/ | Z2 | unknown | disallows /cgi-bin/, /Secure/ and other private paths | The site has no page of terms (read on 2026-11-16). The footer says "Copyright © 2009-2025 UW Math Enrichment" | [P] |
| `lhsmath` | https://www.lhsmath.org/LMT/Problems_and_Solutions | Z2 | unknown | disallows /Account/, /Admin/, /res/, /Download | The site has no page of terms and gives no licence (read on 2026-11-16) | [P] |
| `holbrook` | https://holbrook.bcamathteam.org/ | Z2 | no | all agents allowed; ClaudeBot and 8 other AI crawlers disallowed; ai-train=no (pending) | The site has no page of terms and gives no licence (read on 2026-11-16) | [P] |
| `ommc-official` | https://www.ommcofficial.org/ | Z2 | unknown | allows all | The site has no page of terms and gives no licence (read on 2026-11-16) | [P] |
| `obmep` | https://www.obmep.org.br/ | Z2 | unknown | 404, no rules | The site has a privacy notice but no page of terms for the papers, and gives no licence (read on 2026-11-16) | [P] |
| `tm2` | https://www.tm2.org.br/ | Z2 | unknown | disallows /wp/wp-admin/ only | The site has no page of terms and gives no licence (read on 2026-11-16) | [P] |
| `olimpiadas-uan` | https://oc.uan.edu.co/ | Z3 | unknown | Joomla default | The legal notice at https://oc.uan.edu.co/aviso-legal (read on 2026-11-16) says that the reader "no podrá copiar, reproducir, transmitir, distribuir o crear ... | [P] |
| `omec` | https://omec-mat.org/ | Z2 | unknown | one read: Crawl-delay 120; later no answer | The site did not answer on most tries on 2026-11-16, so no terms were read | [P] |
| `sedem` | https://sedem.org.ec/olimpiadas-2/ | Z2 | unknown | allows all | The site has no page of terms and gives no licence (read on 2026-11-16). The footer says "Copyright © 2026 SEdeM" | [P] |
| `omapa` | https://www.omapa.org/ | Z2 | unknown | disallows /wp-admin/ only | The site has no page of terms and gives no licence (read on 2026-11-16). The footer says "Copyright © 2024. All Rights reserved." Agent decision, not a perso... | [P] |
| `acm-ve` | https://www.acm.ciens.ucv.ve/ | Z2 | unknown | acmfiles S3 robots gives 403; site did not answer | The site www.acm.ciens.ucv.ve did not answer on 2026-11-16, so no terms were read | [P] |
| `minedu-onem` | https://www.minedu.gob.pe/onem/ | Z2 | unknown | two transparency paths only | The site has no page of terms for the ONEM papers and gives no licence (read on 2026-11-16) | [P] |
| `opm` | https://opm.org.pa/ | Z2 | unknown | empty file, allows all | The site has no page of terms and gives no licence (read on 2026-11-16). The footer says "© Olimpiada Panameña de Matemática 2025 / Todos los derechos reserv... | [P] |
| `ommeb-cimat` | https://olimpiadasbasicas.cimat.mx/ | Z2 | unknown | Drupal default | The site has no page of terms and gives no licence (read on 2026-11-16) | [P] |

## 5. Gaps

| Competition | Years | Reason | Note |
|---|---|---|---|
| `pmwc` | all | no-source | No official archive. The domain pmwc.org was parked on 2026-11-16. |
| `mx-omm-state` | all | no-source | The 32 state rounds of the OMM have no common archive. |
| `noetic` | all | robots | The robots.txt of noetic-learning.com disallows `/mathcontest/documents/` and `/*.pdf$`, which hold the papers. |

## 6. Label mapping

The labels of `reports/coverage/unknown-competitions.tsv` that belong to this region now have a line in `registers/competition-aliases.tsv`. A label goes to `-` when it is a mock contest, a forum contest, a collection of problems by topic or a contest for another subject. This census maps 278 labels: 241 new lines and 37 existing lines that now point to a new code. They cover 249 lines of the file and 3,716 rows: 2,557 rows go to a code and 1,159 rows go to `-`. Some new codes have no layout yet. Agent decision, not a person's review: such a code must get a layout before `om import` takes its numbered rows, because the problem numbers restart in each paper.

## 7. Phase 2: ranked harvest list

The counts are estimates from the archive pages that this census read. They are not harvest results. The list puts the Z2 sources with many problems first, because their text can go into the dataset.

| Rank | Source | Competitions | Estimated problems | Access | Licence and zone |
|---|---|---|---|---|---|
| 1 | `obmep` | OBMEP levels 1 to 3 and Mirim | 3,000 | public PDF archive, 2005 onward | no licence; Z2 |
| 2 | `oma` | May Olympiad, Ñandú, OMA levels | 2,000 | public HTML and PDF pages | no licence; Z2 |
| 3 | `lhsmath` | LMT | 1,500 | public PDF archive, 2010 onward | no licence; Z2 |
| 4 | `chiuchang` | IMAS, EMIC, IWYMIC | 1,500 | public PDF files | no licence; Z2 |
| 5 | `purple-comet` | Purple Comet | 1,400 | public PDF archive, 2003 onward | no licence; Z2 |
| 6 | `acm-ve` | Venezuela OJM | 1,000 | public PDF books | no licence; Z2 |
| 7 | `omapa` | Paraguay olympiads | 800 | public PDF files | no licence; Z2 |
| 8 | `usamts` | USAMTS | 700 | public PDF archive, 1989 onward | no licence; Z2 |
| 9 | `cemc` | Gauss, Pascal, Cayley, Fermat, Euclid, Fryer, Galois, Hypatia, CIMC, CSMC, CTMC | 3,000 | public PDF archive (the past-contests page lists 2016 to 2026) | CC BY-NC 4.0; Z3, so phase 2 gives links and match keys, not text |
| 10 | `mathkangaroo-usa` | Kangaroo (US edition) | 300 | sample papers only | no licence; Z2 |
| 11 | `mathhour-uw` | Math Hour Olympiad | 150 | public PDF archive | no licence; Z2 |
| 12 | `bamo` | BAMO | 120 | public PDF archive, 1999 onward | no licence; Z2 |

Each harvest must first read a few sample papers to set the layout of each competition. The rows of a paper whose numbering restarts must not be placed by number alone.
