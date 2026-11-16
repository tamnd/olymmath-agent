# 01f. Census: Western, Northern and Central Europe

Census date: 2026-11-16. Scope: the competitions of the United Kingdom, Ireland, France, the Benelux, Germany, Austria, Switzerland, Italy, Spain, Portugal, the Nordic and Baltic countries, Poland, the Czech Republic, Slovakia, Hungary, Slovenia and Croatia, and the regional international contests (MEMO, the Czech-Polish-Slovak matches, Náboj, the Grand Duchy of Lithuania olympiad). Kangaroo belongs to the census of international contests. The register changes are in tamnd/olymmath (registers/competitions.yaml, sources.yaml, gaps.yaml and competition-aliases.tsv), one pull request per group of countries: tamnd/olymmath#249 (the United Kingdom, Ireland, France and the Benelux), #253 (Germany, Austria, Switzerland, Italy, Spain and Portugal), #254 (the Nordic and Baltic countries) and #255 (Poland, Czechia, Slovakia, Hungary, Slovenia and Croatia).

Legend:
- **[P]** read on the primary page (the organizer's site, the paper or its robots.txt) on the census date.
- **[S]** from a secondary page (an archive of a third party, a list, a forum).
- **UNVERIFIED** seen only in a search result, not confirmed on a page.

Nothing was harvested in this phase. A few pages were read to confirm the shape of an archive, and robots.txt was read for each host that answered.

---

## 1. What the register now holds

The census added 80 competitions (59 high-school, 20 junior, 1 elementary; 75 national, 5 international). 34 of them have a shape read on a primary page (the papers or the archive of the organizer), and two existing rows (ch-smo and pl-omj) got one; the shapes add 11,725 expected positions (section 2). The other 46 new codes have no shape, because no paper of the round was read year by year or the site may not be fetched, and they count no positions.

| Country | New codes |
|---|---|
| GB | uk-pmc, uk-jmc, uk-imc, uk-smc, uk-jmo, uk-imoc, uk-mog, uk-tmc, uk-stmc (uk-ukmt keeps the Kangaroos) |
| IE | ie-prism |
| DE | de-mo-r1, de-mo-r2, de-mo-r3, de-mo-r4 (lower grades of the federal round), de-lwm-by, de-lwm-bw, de-mw-he, de-qedmo |
| FR | fr-cg, fr-tfjm, fr-coupe-animath |
| IT | it-archimede, it-febbraio, it-squadre, it-classi-prime, it-oliforum |
| ES | es-ome-local, es-fespm, es-bcntech |
| PT | pt-opm-a, pt-opm-junior, pt-opm-elim |
| NL | nl-nwo-r1, nl-jwo, nl-tst |
| BE | be-jwo |
| CH | ch-smo-r1, ch-smo-r2, ch-tst |
| DK, NO, SE | dk-georgmohr-r1, no-abel-r1, no-abel-r2, se-skolornas-kval |
| FI, IS | fi-mo-r1, fi-peruskoulu, is-mo-pre |
| EE, LV, LT | ee-mo-reg, ee-open, ee-tst, lv-amo, lv-tst, lt-tst, lt-vumif, lt-gdl |
| PL | pl-om-r1, pl-om-r2, pl-omj-r1, pl-omj-r2 |
| CZ, SK | cz-sk-mo-r1, cz-sk-mo-r2, cz-sk-mo-z, cz-pythagoriada, naboj, naboj-junior |
| HU | hu-arany, hu-durer, hu-zrinyi |
| SI | si-mo-qual, si-tst, si-tech, si-tech-qual, si-vega |
| HR | hr-jmo, hr-girls, hr-tst, hr-state, hr-county, hr-city |
| International | cps-junior, hungary-israel (and naboj, naboj-junior, lt-gdl above) |

Existing rows that changed: uk-ukmt, de-mo, fr-academiques, it-cesenatico, pt-opm, be-omb, lu-mo, ch-smo (now the final round), fi-mo, is-mo, lv-mo, pl-omj (now the final), cz-sk-mo (source `mo-cz` instead of the missing id `matematickaolympiada`, plus `skmo`) and hr-mo. Each description now names its round and points to the codes of the other rounds.

Each row has a `level`, the highest level of its papers (spec 05.2, olymmath-agent#260). The census also filled the two rows of the region that had none: de-mo is high-school (its edition is the final paper of the highest grade), and uk-ukmt is high-school with `levels: [junior, high-school]` (the Junior and Grey Kangaroos are for years 7 to 9). 16 new rows have papers at two or more levels and carry `levels`: the rounds of the German MO for grades 3 to 13 (de-mo-r1 to r3; de-mo-r4 is junior and high-school), the Croatian school rounds (hr-city, hr-county, hr-state), Dürer (grades 3 to 12) and the open olympiad of Latvia (grades 5 to 12) with all three school levels; uk-imoc, ie-prism and the Estonian regional round with junior and high-school; categories Z5 to Z9 of the Czech and Slovak MO, Pythagoriáda, Zrínyi and the Slovenian Vega with elementary and junior. Where a layout mixes levels, each paper carries its own `level`: the junior papers of uk-imoc and ie-prism, and the Z5 papers of cz-sk-mo-z (elementary). it-classi-prime (Gara a squadre delle classi prime, grade 9) is junior.

uk-pmc is the Primary Mathematics Challenge of the Mathematical Association, not a UKMT challenge; its organizer, URL and source (pmc-ma) say so.

## 2. Shapes and expected positions

Expected positions are counted as om counts them: the positions of the shape of each year from `first_year` to 2026, less the years in the row's own `gaps` (years with no edition). The gap register does not lower them; it only marks positions as link-only, robots or no-source. An era with no layout counts nothing. Each row's `# Agent decision` note gives the papers read and the years that have no layout.

| Code | first_year | Shape (positions a year) | Years with no layout | Expected |
|---|---|---|---|---|
| uk-pmc | 2018 | November paper n1-25 and bonus paper b1-25 (50) | none | 450 |
| uk-jmc | 1987 | 25 from 2004 | 1987-2003 | 575 |
| uk-imc | 1987 | 25 from 2004 | 1987-2003 | 575 |
| uk-smc | 1961 | 25 from 2005 | 1961-2004 | 550 |
| uk-jmo | 2015 | a1-10 and b1-6 (16) in 2015-19 and 2021-24; 6 from 2025 | 2020 | 156 |
| uk-imoc | 2015 | Cayley c1-6 and Hamilton h1-6 (junior), Maclaurin m1-6 (18) | none | 216 |
| uk-mog | 2011 | 5 | none | 80 |
| ie-prism | 2006 | junior j1-15 and senior s1-15 (30) from 2016 | 2006-2015 | 330 |
| fr-coupe-animath | 2009 | a1-15 and s1-15 (30) from 2022 | 2009-2021 | 150 |
| nl-nwo-r1 | 2000 | a1-5 and b1-4 (9) to 2005; a1-8 and b1-4 (12) from 2006 | none | 306 |
| nl-jwo | 2008 | a1-15 and b1-10 (25) in 2010-12; a1-8 and b1-8 (16) in 2013-19 and from 2021 | 2008-2009, 2020 | 283 |
| be-jwo | 2019 | a1-30 and b1-30 (60) in 2020; with the final p1-4 (64) from 2021 | 2019 | 444 |
| ch-smo (existing) | 2004 | 10 to 2018, 8 from 2019 | none | 214 |
| ch-smo-r2 | 2004 | 5 to 2016, 6 from 2017 | none | 125 |
| ch-tst | 2004 | 12 | none | 276 |
| de-lwm-by | 1999 | a1-6 and b1-4 (10) from 2010 | 1999-2009 | 170 |
| it-archimede | 1996 | biennio and triennio: 45 (1996-2011), 36 (2012-19), 24 (2020-21), 32 (from 2022) | none | 1,216 |
| it-febbraio | 1997 | 17 (18 in 2007) | 2002-2005 | 443 |
| it-classi-prime | 2013 | 20 in 2013, 18 from 2014; gaps 2021-2023 | none | 200 |
| pt-opm-elim | 1994 | a1-4 and b1-4 (8) | none | 264 |
| pt-opm-a | 1994 | a1-4, b1-4 and the final p1-6 (14); final of 4 in 2020-21 | none | 458 |
| pt-opm-junior | 2011 | as pt-opm-a | none | 220 |
| dk-georgmohr-r1 | 2006 | 20 from 2008 | 2006-2007 | 380 |
| no-abel-r1 | 2005 | 20 | none | 440 |
| no-abel-r2 | 2006 | 10 | none | 210 |
| is-mo-pre | 1984 | lower n1-22 and upper e1-19 (41) in 2009-22; 19 from 2023 | 1984-2008 | 650 |
| pl-om-r1 | 1950 | 12 in 1996-2026 | 1950-1995, from 2027 | 372 |
| pl-om-r2 | 1950 | p1-3 and p4-6 (6) from 1996 | 1950-1995 | 186 |
| pl-omj (existing) | 2006 | 5 from 2010; gap 2020 | 2006-2009 | 80 |
| pl-omj-r1 | 2006 | 7 to 2011; test t1-15 and problems p1-7 (22) from 2012 | none | 372 |
| pl-omj-r2 | 2006 | 4 in 2006, 5 from 2007 | none | 104 |
| cz-sk-mo-r1 | 1952 | home ha/hb/hc 1-6 and school sa/sb/sc 1-3 (27) from 2011 | 1952-2010 | 432 |
| cz-sk-mo-r2 | 1952 | a1-4, b1-4, c1-4 (12) from 2003 | 1952-2002 | 288 |
| cz-sk-mo-z | 2019 | Z5 to Z9 home, district and Z9 regional rounds (50); with the Z9 national round (54) from 2024 | none | 412 |
| hr-jmo | 2017 | 4; two days a1-4 and b1-4 (8) in 2020 and from 2023 | none | 60 |
| hr-girls | 2020 | 4; two days (8) from 2023 | none | 44 |
| **Total** | | | | **11,725** |

de-mo moves from `first_year` 1961 to 1962 (the year of the final of the 1st olympiad, 1961/62), which takes 6 positions off its count; that is in the total. de-mo-r1 to de-mo-r4 have `first_year` 1962 and no era yet. The other new codes with no shape: uk-tmc, uk-stmc, it-squadre, it-oliforum, de-lwm-bw, de-mw-he, de-qedmo, fr-cg, fr-tfjm, es-ome-local, es-fespm, es-bcntech, nl-tst, ch-smo-r1 (the GitHub exams table has no first round), se-skolornas-kval, fi-mo-r1 and fi-peruskoulu (sites that block AI crawlers), ee-mo-reg, ee-open, ee-tst (Cloudflare), lv-amo, lv-tst, lt-tst, lt-vumif, lt-gdl, cz-pythagoriada, hu-arany, hu-durer, hu-zrinyi, hr-tst, hr-state, hr-county, hr-city, si-mo-qual, si-tech, si-tech-qual, si-vega, si-tst, naboj, naboj-junior, cps-junior and hungary-israel.

By group of the PRs: United Kingdom, Ireland, France and the Benelux 4,115; Germany, Switzerland, Italy, Spain and Portugal 3,580 (with ch-smo and the de-mo change); the Nordic and Baltic countries 1,680; Poland, the Czech Republic, Slovakia, Hungary, Slovenia and Croatia 2,350 (with pl-omj).

## 3. Sources

27 new sources: 24 with Z2 statements and solutions, 1 with Z2 statements and Z3 solutions (olympiade-mathematik, whose solutions from the 35th MO are not to be republished), and 2 Z3 (DMFA). Two carry `ai_train: 'no'` and a pending decision (below). All are `state: planned`; 18 are wave 2 and 9 wave 3. The new pmc-ma is the Primary Mathematics Challenge site of the Mathematical Association (no robots.txt; about 275 problems).

### 3.1 robots.txt facts [P]

| Host | Fact (2026-11-16) |
|---|---|
| olympiade-mathematik.de | no robots.txt (404): allowed. Its archive holds the 1st to 34th olympiads (1961/62 to 1994/95); the pages of the later olympiads link to mathematik-olympiaden.de, whose robots.txt disallows all paths. |
| lwmb.de | only a sitemap line: allowed |
| landeswettbewerb-mathematik.de | 404: allowed |
| mathematik-wettbewerb.de | no answer (time out): not read |
| eduscol.education.fr | redirects to eduscol.education.gouv.fr; `*` has Crawl-delay 10; Bytespider, PetalBot, IbouBot, BLEXBot, DataForSeoBot, scoopit, Sogou, Baiduspider, YandexBot and SemrushBot are disallowed |
| apmep.fr | disallows /ecrire/, /lib/, /prive/, archive and login paths; Crawl-delay 1 |
| mathsweek.ie | disallows /2016 to /2022 (PRISM pages of 2016 and 2018 sit there); /2024/prism/ is allowed |
| hanche.folk.ntnu.no | 404: allowed |
| matematikksenteret.no | Drupal paths only |
| georgmohr.dk | 404; the papers say they may not be used to train language models, so ai_train stays no |
| maol.fi | `*` disallows only search, filter and /wp-admin/ paths; under "AI training opt-out" GPTBot, ClaudeBot, anthropic-ai, Google-Extended, CCBot and others are disallowed for all paths |
| durerinfo.hu | `*`: `Content-Signal: search=yes,ai-train=no,use=reference` and `Allow: /`; Amazonbot, Applebot-Extended, Bytespider, CCBot, ClaudeBot, Diffbot, Google-Extended, GPTBot, omgili, anthropic-ai, Claude-Web, cohere-ai, MistralAI-Training, meta-externalagent, GoogleOther, Baiduspider, PetalBot and Awario are disallowed for all paths |
| teaduskool.ut.ee | Cloudflare challenge (403) on every path, robots.txt included: not read, not passed |
| skmo.sk | 404: allowed |
| ics.upjs.sk, math.old.naboj.org, junior.old.naboj.org | no answer from the census machine: not read |
| math.naboj.org | no robots.txt (redirect to a language page) |
| caps.ista.ac.at | allows all |
| repozitorij.dmfa.si | Cloudflare content-signal preamble only, no signal and no rule: allowed |
| natjecanja.math.hr | disallows /images/ and Joomla paths; the papers under /wp-content/uploads/ are allowed |
| azoo.hr | allows all |
| old.matematika.hr | system paths only |
| imosuisse.github.io | allows all |
| lmnsc.lt | disallows all paths: not used |
| primarymathschallenge.org.uk | serves its home page for /robots.txt: no robots.txt, allowed |
| tfjm.org | disallows only /wp-admin/ |
| fespm.es | allows /wp-content/uploads, where the booklets are |
| omb.sbpm.be | disallows /uploads/, /images/ and system paths; where the papers are hosted was not found, so nothing is fetched yet |
| matematiikkakilpailut.fi | disallows /valmennus/2010/ to /valmennus/2023/, /cdn-cgi/ and /matkalasku/; the papers are on Google Drive |
| nms.lu.lv | disallows only print and staff paths |
| pythagoriada.cz | empty Disallow line: allowed |
| bolyai.hu | 404: allowed |
| matek.fazekas.hu | Joomla system paths only |
| stae.is | Crawl-delay 10 |
| mattetavling.se (existing source) | blocks named AI crawlers; no sample fetched, decision pending |

### 3.2 Decision pending

maol-fi and durerinfo allow `*` but block named AI crawlers (ClaudeBot among them), and durerinfo carries `ai-train=no`; the existing source mattetavling (se-skolornas-kval) blocks named AI crawlers too. No sample page was fetched from these three hosts. The owner has not decided whether olymmath-bot may fetch such a site. Their source rows record the exact facts and say "Decision pending"; `ai_train` is `'no'` on both, so the export would leave them out in any case.

## 4. Gaps

The whole-row robots gap of de-mo (mathematik-olympiaden.de disallows all paths) is replaced by a robots gap for 1996 to 2026: olympiade-mathematik.de, which allows all paths, has the papers of the 1st to 34th olympiads (1962 to 1995), and only those years are open again. The same 1996 to 2026 robots gap is set on de-mo-r1 to de-mo-r4.

13 entries added, 1 removed:

| Reason | Entries |
|---|---|
| robots | de-mo, de-mo-r1, de-mo-r2, de-mo-r3, de-mo-r4 (1996-2026); ie-prism (2014-2015, pages under /2016 and /2018 of mathsweek.ie) |
| no-source | lu-mo, hu-zrinyi (whole rows); ie-prism (2006-2013 and 2018-2026, behind the PRISM login) |
| link-only | si-mo-qual, si-tech, si-tech-qual, si-vega (DMFA terms) |
| removed | de-mo robots (whole row) |

## 5. Alias table

24 existing lines changed code (for example "Poland - First Round" from "-" to pl-om-r1, the Febbraio labels to it-febbraio, "Spanish Mathematical Olympiad - Local Stage" to es-ome-local, the Dutch, Swiss, Estonian, Latvian and Croatian TST labels to the new TST codes). 233 lines were added. Against reports/coverage/unknown-competitions.tsv of origin/main, 309 labels with 4,435 rows now resolve: 267 labels (3,665 rows) to a code and 42 labels (770 rows) to "-". The rows that go to "-" are the POFM envois, plain round names that several countries share ("Second Round"), the hf-mathnet rows with only a country name, the Mediterranean competition and mixed sets.

Left open: "Iran" (4 rows of hf-mathnet, a French training set of Iranian problems). A line for it would take every other "Iran" label of the table, so the census of Asia should decide it.

## 6. Phase-2 harvest plan (ranked)

Counts are estimates from the archive pages, not from a harvest. No source states an open licence; each is Z2 (statement with the name of the organizer and a link) unless noted.

| Rank | Source | Codes | Est. problems | Access | Licence and rule |
|---|---|---|---|---|---|
| 1 | olympiade-mathematik | de-mo, de-mo-r1 to r4 | 6,000 | PDF per round and grade, 1st to 34th MO (1962 to 1995) | no licence; later years are a robots gap |
| 2 | skmo + ics.upjs.sk archive | cz-sk-mo, -r1, -r2, -z | 3,500 (A, B, C, 75 years) | PDF, skmo.sk/dokument.php | no licence; read robots of ics.upjs.sk first |
| 3 | natjecanja-math-hr | hr-mo, hr-jmo, hr-girls, hr-tst, hr-city, hr-county, hr-state | 2,000 | PDF under /wp-content/uploads/ | no licence |
| 4 | apmep + eduscol-cg | fr-academiques, fr-cg | 2,000 | PDF; Crawl-delay 1 and 10 | no licence |
| 5 | ukmt-free | uk-jmc to uk-stmc | 2,000 (recent years only) | PDF | Copyright UKMT; papers for sale not fetched |
| 6 | olimpiadi-unibo | it-archimede, it-febbraio, it-classi-prime | 1,800 | WordPress posts and PDFs under /wp-content/uploads/ | no licence |
| 7 | naboj | naboj, naboj-junior | 1,300 | PDF per year (old archive hosts) | no licence; old hosts not reached yet |
| 8 | om-sem + archom | pl-om-r1, pl-om-r2 | 1,100 | PDF and HTML | no licence |
| 9 | stae-is | is-mo, is-mo-pre | 1,000 | HTML and PDF; Crawl-delay 10 | no licence |
| 10 | wiskundeolympiade | nl-nwo-r1, nl-nwo-r2, nl-jwo | 800 | PDF | no licence |
| 11 | bolyai-hu + matek-fazekas | hu-arany | 750 | PDF | no licence |
| 12 | hanche-abel | no-abel-r1, no-abel-r2 | 650 (exact count of the collection) | PDF | no licence |
| 13 | omj | pl-omj, -r1, -r2 | 600 | PDF | no licence |
| 14 | imosuisse | ch-smo-r1, ch-smo-r2, ch-tst | 400 | git (GitHub Pages) | licence of the repo not checked |
| 15 | lwmb, lwm-bw | de-lwm-by, de-lwm-bw | 280 (lwmb; lwm-bw not counted) | PDF | no licence |
| 16 | pmc-ma | uk-pmc | 275 | PDF | no licence; Mathematical Association |
| 17 | rsme-ome | es-ome-local | 200 | PDF | no licence |
| 18 | nms-lu-lv | lv-mo, lv-amo | not counted | PDF | read robots first |
| hold | durerinfo, maol-fi, mattetavling | hu-durer, fi-mo, fi-mo-r1, fi-peruskoulu, se-skolornas-kval | not counted | PDF | ai-train=no / named AI crawlers blocked; decision pending |
| blocked | teaduskool-ut | ee-mo-reg, ee-open | not counted | Cloudflare challenge | not fetched |
| link-only | dmfa-massa, dmfa-massb, dmfa-repozitorij | si-mo-qual, si-tech, si-tech-qual, si-vega | not counted | PDF | Z3 by the DMFA terms |
