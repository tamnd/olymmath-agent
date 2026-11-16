# 01f. Census: Russia, the former USSR, Romania, Bulgaria, the Balkans and Turkey

Date of census: 2026-11-16. Scope: the national and regional competitions of Russia, Ukraine, Belarus, Moldova, Georgia, Armenia, Azerbaijan, Kazakhstan, Uzbekistan, Kyrgyzstan, Tajikistan, Turkmenistan, Romania, Bulgaria, Greece, Cyprus, Serbia, Bosnia and Herzegovina, Montenegro, North Macedonia, Albania, Kosovo and Turkey. The Baltic states, Croatia and Slovenia are in the census of Europe. The Tournament of Towns, Kangaroo and the other multinational contests are in the census of the international competitions.

This note is phase 1 of the census. It adds register rows, sources and gaps. It harvests nothing. The register changes are in tamnd/olymmath, in the PRs listed in section 9.

## 0. Method and conventions

- Probes were run on 2026-11-16 with `curl` (UA `Mozilla/5.0 ...`), WebSearch and WebFetch. The census read HTML pages, robots.txt files and the footers of home pages. To confirm shapes, it downloaded a few sample papers of each shaped row and counted their problems. It kept no problem text and ran no code from a site.
- [P] means primary: the site of the organizer or of the archive, its robots.txt or its terms page. [S] means secondary: a dataset row (MathNet, the AoPS data of AI-MO, the AoPS wiki data), a search summary or a mirror.
- A row with [P] has at least one source that the census checked. A row with [S] is known only from dataset labels. Each [S] row has a `no-source` gap.
- A new row has a shape only for the years whose format a primary paper shows (section 1a). Years with no checked paper have no era, so they count as years of unknown shape. A row with no shape says why in its description.
- Licence zones are those of spec 05 and 11. Every new source is Z2: the sites state no licence, and the spec default for competition statements is Z2. No page that the census read forbids reuse, so no source is Z3.
- Agent decisions are marked "Agent decision, not a person's review." in the register.
- The rule for sites that allow `*` but block named AI crawlers is pending with the owner. Two sources in this note are in that case: hse-olymp (GPTBot blocked, ClaudeBot not named) and math-bas-ch (ClaudeBot blocked). Their notes say that the decision is pending.

## 1. Summary

| Item | Count |
|---|---|
| New competitions | 106 |
| by level | high-school 86, junior 18, open 1, elementary 1 |
| by group | national 102, international 4 |
| New expected positions | 1,282 on 7 rows (section 1a) |
| Rows with `levels` | 21 (19 new rows, plus ru-lomonosov and ru-oral-geometry, which also get `level: high-school`) |
| New sources | 20, all Z2 |
| New alias lines | 239, plus 30 edits of existing lines (from `-` to a code) |
| Gap entries | 72 no-source, 1 robots |

## 1a. Shapes and expected positions

Agent decision, not a person's review: the edition, first_year and position conventions below. The position of problem N of grade G is pG followed by N, as in ru-allrussian and by-mo. first_year is the earliest year that the census found, not the founding year, except for tr-jmo, where the 2010 paper is the 15th olympiad.

| Row | Papers checked [P] | Shape | Years with a shape | first_year | Expected positions |
|---|---|---|---|---|---|
| ru-allrussian-regional | olimpiada.ru and olympiads.mccme.ru: grade 11 of every year, grades 9 and 10 of 2013, 2017, 2018 and 2023 | grades 9 to 11, two days, 8 problems (2011 to 2017) or 10 (2018 to 2024) | 2011 to 2024 | 1993 (seed data) | 378 |
| ru-formulo | olimpiada.ru final papers of grades 5, 7, 9 and 11 for 2023/24 to 2025/26 | grades 5 to 11, 6 problems, final round | 2024 to 2026 | 2014 (olimpiada.ru) | 126 |
| by-mo-regional | mathsmogilev.by paper of 2023/24 | grades 8 to 11, two days, 8 problems | 2024 | 2010 (mathsmogilev.by) | 32 |
| ro-county | mateinfo.ro: grade 9 of 2015, 2019 and 2026, grade 5 of 2013 and 2026, grade 12 of 2026 | grades 5 to 12, 4 problems | 2013 to 2019, 2021 to 2026 | 2001 (seed data) | 416 |
| gr-thales | hms.gr papers of 2014 to 2019, 2021 and 2022 | grades 8 to 12, 4 problems (3 in 2021 and 2022) | 2014 to 2019, 2021, 2022 | 2006 (parmenides-eme) | 150 |
| gr-euclid | hms.gr papers of 2015 to 2020 | grades 8 to 12, 4 problems | 2015 to 2020 | 2006 (parmenides-eme) | 120 |
| tr-jmo | TÜBİTAK second-round papers of 2010 to 2015, 2017, 2019, 2021, 2023 and 2025 | one paper, 4 problems | 2010 to 2015, 2017 to 2025 | 1996 (edition number) | 60 |

- An edition is the calendar year in which the paper is written. So the olimpiada.ru school year Y is the edition Y+1 for ru-allrussian-regional, and the school year 2023/24 is the edition 2024 for ru-formulo and by-mo-regional. The tr-jmo paper of the 2021 season was written on 7 January 2022 and stays the edition 2021.
- A grade paper below the level of the competition has its own `level`: grade 5 is elementary, grades 6 to 9 are junior.
- The TÜBİTAK archive has no tr-jmo second-round paper for 2016, so 2016 has no era. The seed data has 8 rows a year for tr-jmo in 2010 to 2022, so phase 2 must check whether these rows mix the two rounds.
- The other Belarus papers on mathsmogilev.by are on Google Drive, where robots.txt blocks the download path `/uc`. So by-mo-regional has a shape only for 2024.
- Not shaped, with the reason in the description: the Bulgarian rounds on klasirane.com (the format changes between grades and years, and many papers are scans), ua-kyiv (the matholymp.com.ua pages do not show the papers as text), the North Macedonian rows (smm.org.mk shows only the syllabus), the rounds where each region or county sets its own papers (ru-allrussian-municipal, ro-local, ua-umo-regional, ua-umo-district), the Sirius school stage (several variants), the university olympiads (several variants or papers not checked), and all [S] rows.

New competitions by country: RO 24, RU 15, UA 10, BG 7, MK 6, TR 6, AZ 5, KZ 5, GR 4, RS 4, BA 4, AL 4, BY 3, GE 2, CY 2, UZ 1, MD 1, KG 1, TJ 1, TM 1. No new row was needed for Armenia, Montenegro or Kosovo: am-mo, me-mo, xk-mo and xk-tst cover their labels.

Rows that were in the register before the census and are not changed: the international rows balkan, balkan-sl, jbmo, jbmo-sl, sharygin, tuymaada, zhautykov, silk-road, rmm, caucasus, imac and danube; the Russian rows ru-allrussian, ru-moscow, ru-spb, ru-239-open, ru-okruzhnaya, ru-regata, ru-prazdnik, ru-lomonosov, ru-euler, ru-oral-geometry, ru-tst and ru-student; ua-umo, by-mo, md-mo, ge-mo, am-mo, az-mo, kz-mo, uz-mo, kg-mo, tj-mo, tm-mo; ro-nmo, ro-tst, ro-egmo-tst, journal-gazeta; bg-mo, bg-jbmo-tst, bg-egmo-tst, bg-winter, bg-spring; gr-hms, cy-mo, rs-smo, ba-mo, me-mo, mk-mo, al-mo, xk-mo, xk-tst, tr-mo and tr-mo-r1.

## 2. Russia

- The All-Russian olympiad now has a row for each stage: ru-allrussian-school (grades 4 to 11, the Sirius online papers from 2020), ru-allrussian-municipal (a paper set for each region), ru-allrussian-regional (one paper for each of the grades 9 to 11, two days) and the existing ru-allrussian (final stage). [P] olimpiada.ru, vos.olimpiada.ru, siriusolymp.ru.
- The university olympiads: ru-lomonosov-msu (from 2005, [P] olymp.msu.ru), ru-hse ([P] olymp.hse.ru), ru-phystech and ru-ommo ([P] task lists on olimpiada.ru), ru-pvg, ru-vsesib and ru-spbu (archive not found).
- Formula of Unity / Third Millennium is ru-formulo, an international row with the country RU. [P] formulo.org gives one PDF of problems and solutions for each round.
- The Moscow MO, St Petersburg, Euler, the Math Festival (ru-prazdnik) and the Sharygin and Tuymaada olympiads were already in the register. problems.ru is not a source of text (rule of the project). It is used only as a private match key.

| Code | Name | Level | Group | Sources | Mark |
|---|---|---|---|---|---|
| ru-formulo | Formula of Unity / Third Millennium Olympiad | high-school | international | formulo, olimpiada-ru | [P] |
| ru-allrussian-regional | All-Russian Mathematical Olympiad, regional stage | high-school | national | olimpiada-ru, vos-moscow, problems-ru | [P] |
| ru-allrussian-municipal | All-Russian Mathematical Olympiad, municipal stage | high-school | national | vos-moscow, olimpiada-ru | [P] |
| ru-allrussian-school | All-Russian Mathematical Olympiad, school stage | high-school | national | sirius-olymp, vos-moscow | [P] |
| ru-kurchatov | Kurchatov Olympiad in Mathematics | high-school | national | kurchatov-official | [P] |
| ru-lomonosov-msu | Lomonosov Olympiad (Moscow State University) | high-school | national | msu-olymp | [P] |
| ru-pvg | Pokori Vorobyovy Gory Olympiad | high-school | national | none | [S] |
| ru-hse | Vysshaya Proba Olympiad in Mathematics | high-school | national | hse-olymp | [P] |
| ru-phystech | Phystech Olympiad in Mathematics | high-school | national | olimpiada-ru | [P] |
| ru-ommo | Joint Interuniversity Mathematical Olympiad | high-school | national | olimpiada-ru | [P] |
| ru-vsesib | All-Siberian Open Olympiad in Mathematics | high-school | national | none | [S] |
| ru-spbu | St Petersburg State University Olympiad in Mathematics | high-school | national | none | [S] |
| ru-yumsh | YuMSh Olympiad | high-school | national | yumsh | [P] |
| ru-oral-67 | Moscow Oral Olympiad for Grades 6 and 7 | junior | national | none | [S] |
| ru-adygea-teachers | Adygea Teachers' Geometry Olympiad | open | national | hf-ai-mo-aops | [S] |

## 3. Ukraine

- The Ukrainian olympiad now has rows for its stages: ua-umo-district (II), ua-umo-regional (III) and the existing ua-umo (IV, final). ua-kyiv is the Kyiv city olympiad. [P] matholymp.com.ua, olimp.ippo.kubg.edu.ua.
- apex-minds.org is the largest open archive of the region: 21,566 problems from 58 competitions of 1970 to 2025, 11,171 of them from Ukrainian competitions, almost all with solutions [P, about page].
- The Ukrainian geometry and correspondence olympiads, the Champions Tournament, Kharkiv and the Soros olympiad are known only from the AoPS data [S].

| Code | Name | Level | Group | Sources | Mark |
|---|---|---|---|---|---|
| ua-umo-regional | Ukrainian Mathematical Olympiad, regional stage | high-school | national | ippo-kubg, apex-minds | [P] |
| ua-umo-district | Ukrainian Mathematical Olympiad, district stage | high-school | national | ippo-kubg, apex-minds | [P] |
| ua-kyiv | Kyiv City Mathematical Olympiad | high-school | national | matholymp-kyiv, ippo-kubg, apex-minds | [P] |
| ua-correspondence | Ukrainian Correspondence Mathematical Olympiad | high-school | national | hf-ai-mo-aops, hf-aops-wiki | [S] |
| ua-geometry | Ukrainian Geometry Olympiad | high-school | national | hf-ai-mo-aops, hf-aops-wiki | [S] |
| ua-yasinskyi | Yasinskyi Geometry Olympiad | high-school | national | hf-ai-mo-aops | [S] |
| ua-tasks | From Tasks to Tasks | high-school | national | hf-ai-mo-aops | [S] |
| ua-champions | Champions Tournament | high-school | national | hf-ai-mo-aops, hf-aops-wiki | [S] |
| ua-kharkiv | Kharkiv City Mathematical Olympiad | high-school | national | hf-ai-mo-aops | [S] |
| ua-soros | Soros Olympiad in Mathematics | high-school | national | hf-ai-mo-aops | [S] |

## 4. The other states of the former USSR

- Belarus: by-mo-regional is the third stage. The school site mathsmogilev.by keeps the papers of the second, third and final stages (final stage 2011 to 2025, second stage 2010 to 2025), and gymn1math.by keeps the final stage from 1997 [P].
- Kazakhstan: matol.kz has a page for each year and grade of the district, regional and final stages from 2003, and the junior, Almaty, Zhautykov, Al-Farabi and Smagulov olympiads [P home page]. Its robots.txt gave 403 Forbidden on 2026-11-16, so no rule is readable. Check again before a harvest.
- Azerbaijan, Georgia, Moldova, Uzbekistan, Kyrgyzstan, Tajikistan and Turkmenistan: no official archive was found. The rows come from the AoPS data [S].
- The four international rows are ru-formulo, uz-ijmo (Al-Khwarizmi IJMO), ge-euler (an olympiad that the AoPS data calls "Euler Olympiad"; it is not ru-euler) and by-iran-friendly.

| Code | Name | Level | Group | Sources | Mark |
|---|---|---|---|---|---|
| uz-ijmo | Al-Khwarizmi International Junior Mathematical Olympiad | junior | international | hf-ai-mo-aops | [S] |
| ge-euler | Euler Olympiad (Georgia) | high-school | international | hf-ai-mo-aops | [S] |
| by-iran-friendly | Belarus-Iran Friendly Competition | high-school | international | hf-ai-mo-aops | [S] |
| by-mo-regional | Belarus Mathematical Olympiad, regional stage | high-school | national | mathsmogilev, gymn1math | [P] |
| by-tst | Belarus IMO Team Selection Test | high-school | national | hf-mathnet | [S] |
| md-jbmo-tst | Moldova JBMO Team Selection Test | junior | national | hf-ai-mo-aops | [S] |
| ge-tst | Georgia Team Selection Test | high-school | national | hf-ai-mo-aops | [S] |
| az-tst | Azerbaijan Team Selection Tests | high-school | national | hf-ai-mo-aops, hf-aops-wiki | [S] |
| az-jtst | Azerbaijan Junior Team Selection Tests | junior | national | hf-ai-mo-aops, hf-aops-wiki | [S] |
| az-jmo | Azerbaijan Junior Mathematical Olympiad | junior | national | hf-ai-mo-aops, hf-aops-wiki | [S] |
| az-lotfi-zadeh | Lotfi Zadeh Olympiad | high-school | national | hf-ai-mo-aops, hf-aops-wiki | [S] |
| az-istek | Istek Lyceum Mathematical Olympiad | high-school | national | hf-ai-mo-aops | [S] |
| kz-mo-regional | Kazakhstan Mathematical Olympiad, regional stage | high-school | national | matol-kz | [P] |
| kz-mo-district | Kazakhstan Mathematical Olympiad, district stage | high-school | national | matol-kz | [P] |
| kz-junior | Kazakhstan Junior Mathematical Olympiad | junior | national | matol-kz, hf-ai-mo-aops | [P] |
| kz-almaty | Almaty City Mathematical Olympiad | high-school | national | matol-kz, hf-ai-mo-aops | [P] |
| kz-smagulov | Smagulov Olympiad | junior | national | matol-kz | [P] |
| kg-regional | Kyrgyzstan Regional Mathematical Olympiad | high-school | national | hf-ai-mo-aops | [S] |
| tj-tst | Tajikistan Team Selection Test | high-school | national | hf-ai-mo-aops, hf-aops-wiki | [S] |
| tm-regional | Turkmenistan Regional Mathematical Olympiad | high-school | national | hf-ai-mo-aops | [S] |

## 5. Romania

- The olympiad now has ro-local and ro-county next to ro-nmo (national round). Each round has one paper for each of the grades 5 to 12. [P] olimpiade.ro (papers and marking schemes) and mateinfo.ro (papers by county and grade).
- ssmr.ro did not resolve in DNS from the probe host on 2026-11-16. The note of ssmr-onm now says so.
- Stars of Mathematics, the IMAR competition, the Gazeta Matematică olympiad (ONGM) and 17 inter-county competitions are known only from MathNet and the AoPS data [S]. No official archive of their papers was found.

| Code | Name | Level | Group | Sources | Mark |
|---|---|---|---|---|---|
| ro-county | Romanian Mathematical Olympiad, county round | high-school | national | olimpiade-ro, mateinfo-ro, hf-mathnet, hf-ai-mo-aops | [P] |
| ro-local | Romanian Mathematical Olympiad, local round | high-school | national | olimpiade-ro, hf-mathnet | [P] |
| ro-jbmo-tst | Romanian JBMO Team Selection Test | junior | national | hf-mathnet, hf-ai-mo-aops | [S] |
| ro-stars | Stars of Mathematics | high-school | national | hf-mathnet, hf-ai-mo-aops, imogeometry-blog | [S] |
| ro-imar | IMAR Mathematical Competition | high-school | national | hf-mathnet, imogeometry-blog | [S] |
| ro-ongm | Gazeta Matematică National Olympiad | high-school | national | hf-mathnet | [S] |
| ro-clock-tower | Clock-Tower School Competition | high-school | national | hf-mathnet | [S] |
| ro-bogdan-stan | Bogdan Stan Mathematics Competition | high-school | national | hf-ai-mo-aops | [S] |
| ro-ivanescu | Cezar Ivănescu Mathematics Competition | high-school | national | hf-ai-mo-aops | [S] |
| ro-moisil | Grigore Moisil Mathematics Competition (Urziceni) | high-school | national | hf-ai-mo-aops | [S] |
| ro-coculescu | Nicolae Coculescu Mathematics Competition | high-school | national | hf-ai-mo-aops | [S] |
| ro-mihaileanu | N. N. Mihăileanu Mathematics Competition | high-school | national | hf-ai-mo-aops | [S] |
| ro-suceava | Centres of Excellence of Suceava Competition | high-school | national | hf-ai-mo-aops | [S] |
| ro-trident | Petru Moroșan-Trident Mathematics Competition | high-school | national | hf-ai-mo-aops | [S] |
| ro-titeica | Gheorghe Țițeica Mathematics Competition | high-school | national | hf-ai-mo-aops | [S] |
| ro-ramnicean-hope | Râmnicean Hope Mathematics Competition | high-school | national | hf-ai-mo-aops | [S] |
| ro-topan | Teodor Topan Mathematics Competition | high-school | national | hf-ai-mo-aops | [S] |
| ro-paun | Nicolae Păun Mathematics Competition | high-school | national | hf-ai-mo-aops | [S] |
| ro-unirea | Unirea Mathematics Competition | high-school | national | hf-ai-mo-aops | [S] |
| ro-duican | Laurențiu Duican Mathematics Competition | high-school | national | hf-ai-mo-aops | [S] |
| ro-panaitopol | Laurențiu Panaitopol Mathematics Competition | high-school | national | hf-ai-mo-aops | [S] |
| ro-valcovici | Victor Vâlcovici Mathematics Competition | high-school | national | hf-ai-mo-aops | [S] |
| ro-science-on | Science ON Competition | high-school | national | hf-ai-mo-aops | [S] |
| ro-golden-digits | The Golden Digits Competition | high-school | national | hf-ai-mo-aops | [S] |

## 6. Bulgaria

- The olympiad now has bg-mo-municipal and bg-mo-regional next to bg-mo. klasirane.com keeps the papers of all rounds and many tournaments, grades 4 to 12 [P].
- Chernorizets Hrabar (bg-hrabar): the official site is www.math.bas.bg/ch/. The robots.txt of math.bas.bg disallows all paths for ClaudeBot, GPTBot, Bytespider and Amazonbot. klasirane.com says it publishes no new papers of Hrabar at the request of the organizers. The row has a robots gap. Note: before the robots fact was known, the census read the title of the page math.bas.bg/ch/ once. It will not read that site again.
- bg-mo itself stays as it was: its robots block is a rule of the project.

| Code | Name | Level | Group | Sources | Mark |
|---|---|---|---|---|---|
| bg-mo-regional | Bulgarian Mathematical Olympiad, regional round | high-school | national | klasirane, hf-mathnet, hf-ai-mo-aops | [P] |
| bg-mo-municipal | Bulgarian Mathematical Olympiad, municipal round | high-school | national | klasirane | [P] |
| bg-tst | Bulgarian IMO and BMO Team Selection Test | high-school | national | hf-mathnet, hf-ai-mo-aops | [S] |
| bg-autumn | Bulgarian Autumn Mathematical Tournament | high-school | national | klasirane, hf-mathnet, hf-ai-mo-aops | [P] |
| bg-hrabar | Chernorizets Hrabar Mathematics Tournament | high-school | national | math-bas-ch, klasirane | [P] |
| bg-salabashev | Ivan Salabashev Mathematics Competition | high-school | national | klasirane | [P] |
| bg-ifym | International Festival of Young Mathematicians (Sozopol) | high-school | national | hf-ai-mo-aops | [S] |

## 7. The Balkans and Turkey

- Greece: gr-thales, gr-euclid and gr-small-euclid split the competitions of the Hellenic Mathematical Society. hms.gr gives recent papers [P]. The blog page parmenides51.blogspot.com/p/eme.html links Google Drive files with the papers and official solutions of Thales and Euclid 2006 to 2019 and Archimedes 2009 to 2020, then files for each year [P].
- Serbia: rs-primary covers the DMS competitions for primary schools [P dms.rs]. yu-federal is the federal competition of Yugoslavia and of Serbia and Montenegro before 2006.
- North Macedonia: mk-regional, mk-primary and mk-jmo have the SMM site as source [P smm.org.mk]. The tests and the Cane memorial are [S].
- Albania and Kosovo: al-cup and al-xk-mo are new. No official archive was found.
- Turkey: tr-jmo (junior olympiad) has TÜBİTAK as source [P]. The selection tests, Olympic Revenge and the Akdeniz olympiad are [S].
- JBMO and the Balkan MO shortlists were already in the register with a shape. The census added no change to them.

### Greece and Cyprus

| Code | Name | Level | Group | Sources | Mark |
|---|---|---|---|---|---|
| gr-thales | Thales Competition | high-school | national | hms, parmenides-eme | [P] |
| gr-euclid | Euclid Competition (Greece) | high-school | national | hms, parmenides-eme | [P] |
| gr-small-euclid | Small Euclid Competition | elementary | national | hms | [P] |
| gr-tst | Greek Team Selection Examination | high-school | national | hf-mathnet | [S] |
| cy-tst | Cyprus Team Selection Test | high-school | national | hf-ai-mo-aops, hf-aops-wiki | [S] |
| cy-jbmo-tst | Cyprus JBMO Team Selection Test | junior | national | hf-ai-mo-aops, hf-aops-wiki | [S] |

### Serbia and Bosnia and Herzegovina

| Code | Name | Level | Group | Sources | Mark |
|---|---|---|---|---|---|
| rs-tst | Serbian Team Selection Test | high-school | national | hf-ai-mo-aops | [S] |
| rs-jbmo-tst | Serbian JBMO Team Selection Test | junior | national | hf-ai-mo-aops, hf-aops-wiki | [S] |
| rs-primary | Serbian Mathematics Competitions for Primary Schools | junior | national | dms-rs | [P] |
| yu-federal | Yugoslav Federal Mathematics Competition | high-school | national | hf-ai-mo-aops | [S] |
| ba-tst | Bosnia and Herzegovina Team Selection Test | high-school | national | hf-ai-mo-aops | [S] |
| ba-jbmo-tst | Bosnia and Herzegovina JBMO Team Selection Test | junior | national | hf-ai-mo-aops | [S] |
| ba-egmo-tst | Bosnia and Herzegovina EGMO Team Selection Test | high-school | national | hf-ai-mo-aops, hf-aops-wiki | [S] |
| ba-srpska | Republic of Srpska Regional Olympiad | high-school | national | hf-ai-mo-aops | [S] |

### North Macedonia

| Code | Name | Level | Group | Sources | Mark |
|---|---|---|---|---|---|
| mk-regional | North Macedonia Regional Mathematics Competition | high-school | national | smm, hf-mathnet | [P] |
| mk-primary | North Macedonia State Mathematics Competition for Primary Schools | junior | national | smm, hf-mathnet | [P] |
| mk-jmo | North Macedonia Junior Mathematical Olympiad | junior | national | smm, hf-mathnet | [P] |
| mk-tst | North Macedonia Team Selection Test | high-school | national | hf-mathnet, hf-ai-mo-aops | [S] |
| mk-jbmo-tst | North Macedonia JBMO Team Selection Test | junior | national | hf-ai-mo-aops, hf-aops-wiki | [S] |
| mk-cane | Aleksandar Blazhevski-Cane Memorial Competition | high-school | national | hf-mathnet, hf-ai-mo-aops, hf-aops-wiki | [S] |

### Albania and Kosovo

| Code | Name | Level | Group | Sources | Mark |
|---|---|---|---|---|---|
| al-tst | Albania Team Selection Test | high-school | national | hf-ai-mo-aops, hf-aops-wiki | [S] |
| al-jbmo-tst | Albania JBMO Team Selection Test | junior | national | hf-ai-mo-aops | [S] |
| al-cup | Albanians Cup in Mathematics | high-school | national | hf-ai-mo-aops | [S] |
| al-xk-mo | Kosovo and Albania Mathematical Olympiad | high-school | national | hf-ai-mo-aops, hf-aops-wiki | [S] |

### Turkey

| Code | Name | Level | Group | Sources | Mark |
|---|---|---|---|---|---|
| tr-tst | Turkish IMO Team Selection Test | high-school | national | hf-mathnet, hf-ai-mo-aops | [S] |
| tr-jbmo-tst | Turkish JBMO Team Selection Test | junior | national | hf-mathnet | [S] |
| tr-egmo-tst | Turkish EGMO Team Selection Test | high-school | national | hf-mathnet, hf-ai-mo-aops | [S] |
| tr-jmo | Turkish Junior Mathematical Olympiad | junior | national | tubitak, hf-mathnet | [P] |
| tr-revenge | Turkey Olympic Revenge | high-school | national | hf-ai-mo-aops | [S] |
| tr-akdeniz | Akdeniz University Mathematical Olympiad | high-school | national | hf-ai-mo-aops | [S] |

## 8. New sources

All sources are Z2 for statements and solutions. "No licence" means that the census found no licence and no terms of reuse. All facts were read on 2026-11-16 [P].

| Source id | URL | Access | robots.txt | Terms | Note |
|---|---|---|---|---|---|
| olimpiada-ru | https://olimpiada.ru/ | PDF | `*` disallows /activities/*, /activity/*/page* and search; task pages allowed; no AI crawler named | © Олимпиада.ру 1996 to 2026, no licence | /activity/N/tasks lists papers by year, stage and grade |
| vos-moscow | https://vos.olimpiada.ru/ | PDF | 404, so all allowed | © 2026, legal page, no licence | school, municipal and regional stages in Moscow |
| sirius-olymp | https://siriusolymp.ru/ | PDF | `*` disallows some single pages; no AI crawler named | © 2026 Фонд «Талант и успех», no licence | school stage from 2020, with solutions |
| formulo | https://www.formulo.org/ | PDF | all allowed | no copyright line | one PDF of problems and solutions for each round |
| kurchatov-official | https://olimpiadakurchatov.ru/ | PDF | `*` disallows system and form paths | © 2026, no licence | archive pages not yet located |
| msu-olymp | https://olymp.msu.ru/ | PDF | `*` with no Disallow | legal page, no licence | held from 2005 |
| hse-olymp | https://olymp.hse.ru/ | PDF | `*` Crawl-delay 3; GPTBot disallowed /; ClaudeBot not named | no licence | decision pending |
| yumsh | https://www.yumsh.ru/ | HTML | `*` Crawl-delay 10 | no licence | archive pages not yet located |
| matholymp-kyiv | https://matholymp.com.ua/archive | PDF | only `User-agent: *` | © 2010 to 2026, no licence | Kyiv stage, with solutions and criteria |
| matholymp-ua | https://matholymp.danmysak.com/ | HTML | `*` with no Disallow | © 2008 to 2026, no licence | final stage of the Ukrainian olympiad, both rounds |
| ippo-kubg | https://olimp.ippo.kubg.edu.ua/archives/category/matem | PDF | disallows /wp-admin/ only | © 2026 Grinchenko university institute, no licence | II and III stages in Kyiv |
| apex-minds | https://apex-minds.org/ | HTML | allows /, disallows /api/ | "open archive", no licence | 21,566 problems, 11,171 Ukrainian; translated texts of other contests stay link-only |
| mathsmogilev | https://mathsmogilev.by/1415/ | PDF | allows / | no copyright line | Belarus second, third and final stages |
| gymn1math | https://gymn1math.by/ | PDF | empty Disallow | © 2021, no licence | Belarus final stage from 1997 |
| matol-kz | https://matol.kz/ | HTML | 403 on both hosts | no licence; sells books of papers | check robots again before a harvest |
| olimpiade-ro | https://www.olimpiade.ro/ | PDF | Yandex and MJ12bot disallowed; `*` disallows system paths | Copyright © 2011, no licence | local, county and national rounds, grades 5 to 12, with marking schemes |
| mateinfo-ro | https://www.mateinfo.ro/ | PDF | disallows /wp-admin/ only | © 2026, all rights reserved | local and county rounds by county, inter-county contests |
| klasirane | https://klasirane.com/ | PDF | empty Disallow | no copyright line | no new Hrabar papers, at the request of the organizers |
| math-bas-ch | https://www.math.bas.bg/ch/ | PDF | ClaudeBot, GPTBot, Bytespider, Amazonbot disallowed / | not read | robots gap; decision pending |
| parmenides-eme | https://parmenides51.blogspot.com/p/eme.html | PDF (Google Drive) | disallows /search only | no licence | link page to the Greek papers and solutions |

## 9. Gaps, aliases and PRs

Gaps (in `registers/gaps.yaml`):

- 72 `no-source` gaps, one for each [S] row of sections 2 to 7. 68 rows have only dataset sources. ru-pvg, ru-vsesib, ru-spbu and ru-oral-67 have no source.
- 1 `robots` gap: bg-hrabar, for the official site math.bas.bg/ch/.

Aliases (in `registers/competition-aliases.tsv`):

- 239 new lines map the labels of the region in `reports/coverage/unknown-competitions.tsv`. The estimate before the coverage run: 252 report lines (label and source) and 3,384 seed rows now map to a code, and 7 lines (106 rows) map to `-`.
- 30 existing lines change from `-` to a code, for example "District Olympiad" to ro-county, "Bulgaria TST" to bg-tst and "Turkey Junior National Olympiad" to tr-jmo.
- `-` is kept for labels that are not one competition, for example "Romania Olympiad", "Problems from Ukrainian Authors" and "Ukrainian Mathematical Competitions".
- Left without a code: the Romanian JBMO selection labels that MathNet files under Brazil, the 2003 JBMO selection tests of several countries, the Mediterranean competition, the Girls EMO tests of North Macedonia and the Zhautykov city olympiad. They need a person's review.

PRs in tamnd/olymmath, one for each country group:

- tamnd/olymmath#246: Russia, Ukraine and the former USSR (45 rows, 536 expected positions).
- tamnd/olymmath#247: Romania and Bulgaria (31 rows, 416 expected positions).
- tamnd/olymmath#248: Greece, Cyprus, the Western Balkans and Turkey (30 rows, 330 expected positions).

The census comments are on tamnd/olymmath-agent#6.

## 10. Phase 2: ranked harvest sources

The counts are my estimates from the page lists, unless a note says [P]. All sources are Z2: the statement with a link, and the solutions stay at the source unless the owner gives permission.

| Rank | Source | Competitions | Est. problems | Access | Licence and conditions |
|---|---|---|---|---|---|
| 1 | apex-minds | ua-umo (all stages), ua-kyiv and other Kyiv contests | 11,171 Ukrainian [P] | HTML pages, search filters; /api/ disallowed | Z2; no licence; take only the Ukrainian originals |
| 2 | olimpiade-ro | ro-local, ro-county, ro-nmo, ro-tst | 3,000 to 5,000 (8 grades × 4 problems × 3 rounds × ~20 years, plus local papers) | PDF with marking schemes | Z2; SSMR holds the rights to its problems |
| 3 | klasirane | bg-mo-municipal, bg-mo-regional, bg-autumn, bg-salabashev | 3,000 to 4,000 | PDF by year and grade | Z2; skip Hrabar |
| 4 | olimpiada-ru, vos-moscow, sirius-olymp | ru-allrussian (all stages), ru-phystech, ru-ommo, ru-euler | 2,000 to 3,000 | PDF by year, stage and grade | Z2; problems.ru never used as text |
| 5 | matol-kz | kz-mo (all stages), kz-junior, kz-almaty, kz-smagulov | ~2,000 | HTML page for each year and grade | Z2; robots.txt gives 403, check before a harvest |
| 6 | formulo | ru-formulo | ~1,000 | PDF with solutions | Z2 |
| 7 | mathsmogilev, gymn1math | by-mo, by-mo-regional | ~1,500 | PDF | Z2 |
| 8 | matholymp-kyiv, ippo-kubg, matholymp-ua | ua-kyiv, ua-umo-regional, ua-umo | ~1,500, much of it also in apex-minds | PDF and HTML | Z2 |
| 9 | hms and parmenides-eme | gr-thales, gr-euclid, gr-hms | ~800 | PDF on hms.gr and Google Drive | Z2 |
| 10 | tubitak | tr-jmo, tr-mo | ~600 | PDF | Z2 |
| 11 | dms-rs, smm | rs-primary, rs-smo, mk-regional, mk-primary, mk-jmo | ~1,000 | PDF | Z2 |
| 12 | mateinfo-ro | ro-county, ro-local, inter-county contests | overlaps rank 2 | PDF | Z2; all rights reserved |
| 13 | msu-olymp, kurchatov-official, yumsh | ru-lomonosov-msu, ru-kurchatov, ru-yumsh | ~500 | PDF and HTML | Z2; yumsh Crawl-delay 10 |
| Hold | hse-olymp, math-bas-ch | ru-hse, bg-hrabar | | | AI crawler rule pending with the owner |

## 11. Open items

- The rule for sites that block named AI crawlers (hse-olymp, math-bas-ch) is pending with the owner.
- `levels` is given only where the census confirmed the grades of the papers: 19 new rows (the official page [P], or for ro-stars, ro-clock-tower and gr-tst the junior and senior papers in the dataset labels [S]) and the existing ru-lomonosov (grades 6 to 11, olimpiada.ru/activity/4) and ru-oral-geometry (grades 8 to 11, olimpiada.ru/activity/18), which had no level before. Grade 5 counts as elementary, so ro-local, ro-county, ru-formulo and ru-allrussian-school have three levels. mk-regional, rs-primary, tr-jmo, gr-small-euclid, kz-mo-regional, kz-mo-district, ru-kurchatov, ru-hse, ru-yumsh, bg-autumn and bg-salabashev keep one level until their grades are confirmed.
- matol.kz robots.txt and ssmr.ro DNS must be checked again before phase 2.
- Only 7 new rows have a shape, and only for the years in section 1a. Phase 2 must read the official papers of the other years and rows before it adds a layout. olimpiade.ro returned 403 (Mac) and 503 (server2) on 2026-11-16, so the ro-county shape comes from mateinfo.ro.
