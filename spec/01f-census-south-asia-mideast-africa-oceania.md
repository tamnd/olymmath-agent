# 01f. Census: South Asia, the Middle East, Africa and Oceania

This note records the phase 1 census of the competitions of South Asia, the Middle East, Africa and Oceania. The agent read the sites and their robots.txt files on 2026-11-16. Nothing was harvested. The agent read a few sample pages only, to confirm the shape of a paper.

The register rows are in tamnd/olymmath: `registers/competitions.yaml`, `registers/sources.yaml`, `registers/gaps.yaml` and `registers/competition-aliases.tsv`.

## Marks

- [P] The agent read the primary source, such as the official site or an official paper.
- [S] The fact comes from search results or a third-party copy only. Phase 2 must confirm it.

## 1. Census

### South Asia

| Code | Competition | Level | Source and facts | Mark |
|---|---|---|---|---|
| in-prmo-ioqm | PRMO and IOQM | high-school | HBCSE papers. IOQM has 30 questions (10 of 2, 10 of 3 and 10 of 5 marks). KV students write their own paper, so "KVS IOQM" maps here. | [P] |
| in-rmo | RMO | high-school | HBCSE papers from 2001. Link-only by rule. | [P] |
| in-inmo | INMO | high-school | HBCSE papers from 2000. Link-only by rule. | [P] |
| in-tst | India TST | high-school | AoPS collections. | [S] |
| in-egmo-tst | India EGMO TST (new) | high-school | AoPS, 2021 on. | [S] |
| in-nmtc | NMTC (new) | junior, high-school, undergraduate | AMTI. Levels Gauss, Kaprekar, Bhaskara, Ramanujan and Aryabhata, two stages. AMTI publishes no papers. Copies are on coaching sites only. | [S] |
| in-mtrp | MTRP (new) | junior, high-school | integrationfest.in. Junior and Senior. The site gave DEPLOYMENT_NOT_FOUND on 2026-11-16. | [S] |
| in-stems | STEMS (new) | high-school, undergraduate | tessellate.cmi.ac.in. Math categories A, B and C. The CS and physics parts are out of scope. | [S] |
| in-limit | LIMIT (new) | high-school, undergraduate | Students of ISI Bangalore, from 2019. Categories 1 and 2, then A, B and C. No site archive found. | [S] |
| in-assam | Assam Mathematics Olympiad (new) | junior, high-school | aamonline.org.in. Categories I to III, papers of 2017 to 2025. | [P] |
| in-kvjmo | KVS Junior MO (new) | high-school | 1998 to 2018. Copies are on school library blogs only. | [S] |
| (none) | CMI and ISI entrance exams | - | Entrance exams, not competitions. The labels map to `-`. | [S] |
| (none) | INMO junior | - | No separate junior INMO was found. The IOQM and RMO have no junior paper. | [S] |
| bd-bdmo | BdMO | elementary, junior, high-school | matholympiad.org.bd. Primary, Junior, Secondary and Higher Secondary categories, regional and national rounds. Robots: see section 3. Kaggle and GitHub sets for 2004 to 2024 exist. | [P] robots, [S] archive |
| pk-mo | Pakistan MO | high-school | No source. | [S] |
| pk-tst | Pakistan TST (new) | high-school | AoPS. | [S] |
| lk-mo | Sri Lanka MO | high-school | slmathsolympiad.org. | [P] robots |
| lk-slmc | Sri Lanka Mathematics Challenge (new) | high-school | slmathsolympiad.org, AoPS 2022. | [S] |
| np-mo | Nepal MO (Pre-TST) | high-school | mathsinitiatives.org.np. | [P] robots |
| np-tst | Nepal TST (new) | high-school | mathsinitiatives.org.np, AoPS. | [S] |

### Iran and Israel

| Code | Competition | Level | Source and facts | Mark |
|---|---|---|---|---|
| ir-mo-r1 | Iran MO first round (new) | high-school | AoPS. Short answer paper. | [S] |
| ir-mo-r2 | Iran MO second round | high-school | AoPS, amoozz.ir. | [S] |
| ir-mo | Iran MO third round | high-school | AoPS, imomath. | [S] |
| ir-tst | Iran TST | high-school | AoPS. | [S] |
| ir-rmm-tst | Iran RMM TST (new) | high-school | AoPS. | [S] |
| ir-combinatorics | Iranian Combinatorics Olympiad (new) | high-school | AoPS. | [S] |
| ir-ourmo | Iranian Our MO (new) | high-school | AoPS. Set by students. | [S] |
| igo | IGO, all three levels | junior (Elementary paper), high-school | igo-official.com, booklets with solutions. The junior and senior labels map to igo. | [P] |
| by-iran-friendly (from #246), india-iran, iran-taiwan | Friendly competitions (India-Iran and Iran-Taiwan new) | high-school | AoPS. | [S] |
| (none) | Iran junior olympiad | - | Not found. | [S] |
| il-gillis | Gillis olympiad | high-school | weizmann.ac.il/math/gillis/node/11: 53 PDF links, papers of 1983 to 1998 (no 1995) and Hebrew years 5759 to 5786 (no 5764, 5765), many with solutions. | [P] |
| il-grossman | Grossman olympiad (Technion) | high-school | No source. | [S] |
| il-revenge | Israel Olympic Revenge (new) | high-school | AoPS. | [S] |
| il-oral | Israel Oral Olympiad (new) | high-school | AoPS. | [S] |
| (none) | Israel junior olympiad | - | Not found. | [S] |

### Gulf and Levant

| Code | Competition | Level | Source and facts | Mark |
|---|---|---|---|---|
| gulf | Gulf MO | high-school | ds.abegs.org/gmo shows GMO 2019 (7th, Muscat) only. Direct links to 2015 and 2016 work. No GMO in 2018. 8th GMO in Saudi Arabia, 2022. Footer: all rights reserved, ABEGS. | [P] |
| arab | Arab MO | high-school | Every two years with ALECSO. 4th edition in Doha, November 2024. No official problems found. | [S] |
| sa-samc | Saudi competitions and TSTs | high-school | Booklets: 2013 on faculty.ksu.edu.sa, 2014 and 2015 on AoPS, 2016 and 2021 on molympiad.net. Mawhiba IMO booklets of 2022 and 2024 on document sites. The booklets mix TSTs, so their labels map to `-`. | [S] |
| ae-mo | UAE | high-school | competitions.ae (Masterminds). Link to a national olympiad not confirmed. | [P] robots |
| kw-mo, qa-mo, bh-mo, om-mo | Kuwait, Qatar, Bahrain, Oman | high-school | No source. | [S] |
| jo-mo | Jordan | high-school | JMO and JNMO named on a third-party page. No source. | [S] |
| sy-mo | Syria | high-school | sites.google.com/view/syrian-mathematical-olympiad: training notes and "problems from old exams". | [S] |
| lb-mo, iq-mo, ps-mo | Lebanon, Iraq, Palestine | high-school | No source. Palestine first went to the IMO in 2022. | [S] |

### Egypt and North Africa

| Code | Competition | Level | Source and facts | Mark |
|---|---|---|---|---|
| eg-mo | Egypt | high-school | Nothing found. | [S] |
| ma-mo | Morocco | high-school | empimo.github.io/library/maroc.html: tests of 1983 to 2016, papers of 2019, 2021, 2022 on Google Drive. AoPS 2011. | [S] |
| ma-tst | Morocco TST | high-school | AoPS 2005, 2011; empimo 2017. | [S] |
| ma-mmc | Math&Maroc Competition (new) | high-school | mathmaroc.org. Two-day olympiad contest. | [S] |
| tn-mo | Tunisia | high-school | ATSM. No archive. | [S] |
| dz-mo | Algeria | high-school | start.algerianmo.com, ALMO. No archive. | [S] |

### Rest of Africa

| Code | Competition | Level | Source and facts | Mark |
|---|---|---|---|---|
| pamo | PAMO | high-school | pamoofficial.org/problems. | [P] |
| pamo-sl | PAMO shortlist (new) | high-school | AoPS, imogeometry, 2017 on. | [S] |
| (none) | PAMO junior | - | Not found. | [S] |
| eamo | East African MO (new) | high-school | eamo-olympiad.org. From 2023. /past-papers gave 404. | [P] |
| za-samo | SAMO | junior, high-school | samf.ac.za. Junior (grades 8, 9) and Senior (10 to 12), three rounds, round 1 has 20 questions. Past papers need a login. | [P] |
| (none) | SA Mathematics Challenge (grades 4 to 7) | - | Papers go to schools by e-mail. Not added. | [S] |
| safest | South Africa - Estonia (new) | high-school | AoPS. | [S] |
| ng-mo | Nigeria | junior, high-school | NMC olympiad, junior and senior, rounds 1 to 4. AoPS senior rounds 2 to 4. | [S] |
| ke-mo | Kenya (KKMO) | junior, high-school | University of Nairobi. Three rounds. Copies on Scribd and Course Hero only. | [S] |
| gh-mo, ug-mo, rw-mo | Ghana, Uganda, Rwanda | high-school | misemaths.org (no answer), ums.ug, rwandamatholympiad.com. No archive. | [S] |
| bf-mo | Burkina Faso (new) | high-school | cmathburkina.org. Robots: see section 3. | [P] robots |
| tz-mo, zw-mo, bw-mo, ci-mo, sn-mo, cm-mo, et-mo | Other countries | high-school | No source. | [S] |

### Australia, New Zealand and the Pacific

| Code | Competition | Level | Source and facts | Mark |
|---|---|---|---|---|
| au-amo | AMO | high-school | Link-only by rule (gaps.yaml). Not changed. | [P] |
| au-amc | AMC | elementary, junior, high-school | Divisions Middle Primary, Upper Primary, Junior, Intermediate, Senior. No paper on the AMT site. | [S] |
| au-mcyb | MCYA | elementary, junior, high-school | Years 3 to 10. Challenge, Enrichment and AMOC Intermediate Contest. Books only. | [S] |
| au-aimo | AIMO (new) | high-school | One paper for years 10 and below. AMT: papers of 2017 and 2018 with solutions. Ten problems. Book of 1999 to 2013. | [P] |
| au-amoc-senior | AMOC Senior Contest (new) | high-school | Books only. | [S] |
| au-unsw | UNSW School MC (new) | high-school | UNSW page: problems of 2009 to 2024, no 2020. Parabola 2023, 2024. | [P] |
| au-wajo | WA Junior MO (new) | junior | UWA, years 7 to 9. No source. | [S] |
| (none) | State competitions | - | Maths Talent Quest and Tournament of Minds are investigations, not problem contests. | [S] |
| nz-nzmo | NZMO | high-school | mathsolympiad.org.nz, CC BY 4.0. Round One 8 take-home problems, Round Two 5 problems in 3 hours, 2019 to 2026. | [P] |
| (none) | NZMOC camp selection problems | - | A training set. The label maps to `-`. | [S] |
| (none) | Pacific islands | - | No competition found. | [S] |

## 2. Census numbers

The register rows are merged in tamnd/olymmath#243 (South Asia, Iran and Israel), #244 (the Middle East and Africa) and #245 (Australia and New Zealand). Each one passed the targeted check (tcheck) with no new lines, and the slim check with no errors apart from the T03 lines of the partial tree.

- New competitions: 27. By highest level: high-school 23, undergraduate 3, junior 1. By country: India 7, international 5, Australia 4, Iran 4, Israel 2, Pakistan, Sri Lanka, Nepal, Morocco and Burkina Faso 1 each.
- Changed rows: 19, among them by-iran-friendly and nz-nzmo. by-iran-friendly came to main in #246 and replaces the belarus-iran row that this census first added; it gets the shape of 2022 to 2024 and the first year 2022. nz-nzmo gets the shape of 2019 on and the first year 2019 (13 positions a year, so 104 expected positions for 2019 to 2026).
- Expected positions added: 291. Each new row gets the shape of the years that papers show, and no other years:

| Code | Expected | Evidence |
|---|---|---|
| nz-nzmo | 104 | NZMOC papers of 2019 to 2026: Round One 8, Round Two 5. The existing Round One records cover 56. |
| in-assam | 55 | AAM papers of 2023: Category I 18, Category II 19, Category III 18 questions. The 2018 Category III paper has 12, so other years have no shape. |
| au-unsw | 35 | UNSW papers: 2012 and 2022 have 6 Junior and 6 Senior problems, 2019 has 6 and 5. |
| ir-combinatorics | 21 | AoPS papers of 2020, 2021 and 2025, 7 problems each. |
| au-aimo | 20 | AMT papers of 2017 and 2018, 10 problems each. |
| il-revenge | 20 | AoPS papers of 2021 to 2025, 4 problems each. |
| by-iran-friendly | 18 | AoPS papers of 2022 to 2024, 6 problems each. |
| in-egmo-tst | 18 | AoPS tests of 2022 to 2024, 6 problems each. |

  No paper shows the shape of the other 21 new rows. Their papers do not agree from year to year (india-iran, ir-rmm-tst, np-tst, in-stems, in-limit, pamo-sl), AoPS has one year only (iran-taiwan, ir-mo-r1, ir-ourmo, il-oral, pk-tst, lk-slmc, in-mtrp, safest), or no paper is online (in-nmtc, in-kvjmo, au-amoc-senior, au-wajo, eamo, ma-mmc, bf-mo).
- New sources: 15 (Z1: 1, Z2: 14), and gulf-official gets terms and robots facts.
- Labels mapped: 79 alias lines, which cover 97 labels and 1,215 rows of the unknown-competitions report (306 of them map to `-`). The label hungaryisrael_binational is left to the hungary-israel row that #255 added. The label "Belarus - Iran Friendly Competition" is left to by-iran-friendly.
- Gaps: 33 new no-source entries, and the za-samo entry gets a new note.
- Levels follow the scope PR (#260). `level` is the highest level, and `levels` lists all levels, lowest first, when there are two or more. A paper for two levels takes the higher one. 12 rows get `levels`: au-amc, au-mcyb, bd-bdmo, igo, in-nmtc, in-mtrp, in-stems, in-limit, in-assam, za-samo, ng-mo and ke-mo. The Elementary paper of igo gets `level: junior`. au-amc had no level and gets high-school.

## 3. Source rules

- Spec 11: a site that forbids reuse is Z3. Paths that robots.txt disallows go into gaps.yaml with reason robots. A competition with no allowed source gets a no-source gap. The agent never bypasses robots.txt, logins or paywalls.
- NZMOC is Z1, because its resources page puts past NZMO problems and solutions under CC BY 4.0.
- Sites with no licence are Z2: the content holds the statements with the organizer named and a link to the paper.
- Pending decision: matholympiad.org.bd, competitions.ae, cmathburkina.org (and pamoofficial.org and artofproblemsolving.com) serve the Cloudflare managed robots.txt. Its * group says `Content-Signal: search=yes,ai-train=no,use=reference` and `Allow: /`, and groups for ClaudeBot, GPTBot, CCBot and other AI crawlers say `Disallow: /`. The om fetcher matches the * group. The owner has not decided how to treat these sites. The sources record the facts and mark the decision as pending. bg-mo and pamo-official are not changed.
- SAMF past papers need a login, so za-samo keeps a no-source gap.

## 4. Robots.txt facts (2026-11-16)

| Host | Facts |
|---|---|
| www.mathsolympiad.org.nz | * group: Crawl-Delay 20, Disallow /admin, /workshops/problems |
| amt.edu.au | * group: Disallow /wp-admin/, Crawl-delay 10 |
| www.unsw.edu.au | * group: Crawl-delay 10, Disallow /content/dam/images/fsdd/no-follow/ |
| www.parabola.unsw.edu.au | Drupal defaults |
| www.weizmann.ac.il | * group: Crawl-Delay 10, many disallows, none under /math/gillis/ |
| olympiads.hbcse.tifr.res.in | WordPress default |
| ds.abegs.org | * group: Disallow /admin/, /manage/ |
| www.samf.ac.za | * group: Disallow /admin, /bin/, /content/files/, shop and forum paths |
| matholympiad.org.bd, competitions.ae, cmathburkina.org, pamoofficial.org | Cloudflare managed file, see section 3 |
| www.slmathsolympiad.org | * group: Allow /, Disallow /api/, /compete, /admin, /ph_return_ |
| www.mathsinitiatives.org.np | No * group; named bots, ClaudeBot among them, allowed |
| aamonline.org.in | WordPress default |
| sites.google.com | Disallow /feeds and /*/_/ |
| drive.google.com | Allow /file, Disallow /open and / |
| empimo.github.io, eamo-olympiad.org, igo-official.com, www.isical.ac.in | 404, no rules |
| tessellate.cmi.ac.in | App page instead of robots.txt, no rules |
| integrationfest.in | Vercel DEPLOYMENT_NOT_FOUND |
| mathmaroc.org, greatfuture.org.pk, amoozz.ir, mtrp.in, amtionline.com | Allow all |
| molympiad.net | Disallow /search, /share-widget |
| www.imomath.com | 403 |
| mathemaroc.com, misemaths.org, kkmo-official.com, eamo.africa, aimo.amt.edu.au | No answer |

## 5. Phase 2 harvest sources, ranked

| Rank | Source | Competition | Problems (estimate) | Access | Licence |
|---|---|---|---|---|---|
| 1 | nzmoc | nz-nzmo | about 104 (8 years of 13), with solutions | PDF per round and year, crawl delay 20 s | CC BY 4.0 (Z1) |
| 2 | weizmann-gillis | il-gillis | about 250 (about 40 papers of 5 to 7 problems), many with solutions | PDF, crawl delay 10 s, Hebrew | none stated (Z2) |
| 3 | igo-official | igo | about 180 (12 years of 15) | PDF booklets | none stated (Z2) |
| 4 | aam | in-assam | about 150 (9 years, 3 categories) | PDF | none stated (Z2) |
| 5 | unsw-smc | au-unsw | about 150 (15 years, 2 divisions) | PDF | none stated (Z2) |
| 6 | empimo | ma-mo, ma-tst | about 200 | Google Drive /file pages | none stated (Z2) |
| 7 | gulf-official | gulf | about 12 (2015, 2016, 2019) | PDF | all rights reserved (Z2) |
| 8 | amt | au-aimo | 20 (2017, 2018) | PDF | © AMT (Z2) |
| 9 | stems, mtrp | in-stems, in-mtrp | about 100 each | PDF, sites to confirm | none stated (Z2) |
| 10 | bdmo-official, uae-competitions, cmathburkina | bd-bdmo, ae-mo, bf-mo | unknown | HTML | pending decision |
