# 01b. Research: the competitions and their primary sources

Status: research note. Date: 2026-10-31. Scope: every maths olympiad and competition we want in olymmath, with the primary place to get its problems and official solutions.

## 0. How to read this note

- **[P]** means a primary source. It is the organiser's own site, file or book.
- **[S]** means a secondary source. It is a mirror, a fan site, a forum, a dataset or an encyclopedia.
- **UNVERIFIED** means we could not confirm the fact during this research pass. Treat it as a lead, not a fact.
- "Est. total" is our own arithmetic: years held times problems per year. It is always approximate. It counts problems from the main rounds only, unless the row says otherwise.
- "Sol." says whether the organiser publishes official solutions. Y = yes, N = no, P = partial (some years or short solutions only), ? = unknown.
- "Licence" records what the source says. Almost no source states an open licence. "None stated" means we found no licence text beyond a copyright line.
- The slug is the proposed `competition` id for the olymmath data model. Nationals use an ISO 3166 alpha-2 prefix, e.g. `cn-mo`, `ru-allrussian`. International contests have no prefix.

Key finding up front: about 210 competitions are catalogued below (157 table rows; some rows group several small countries). We estimate about **120,000 distinct olympiad-level problems** exist (plausible range 90k to 160k). About 50k of them are proof problems. Section 9 gives the reasoning.

---

## 1. Global hubs and compendia

These are not competitions. They are where problems from many competitions are already collected. They matter for discovery, cross-checking and dedup.

| Hub | What it holds | URL | Tag | Notes |
|---|---|---|---|---|
| IMO official site | IMO problems 1959 to 2026, shortlists after embargo, results | https://www.imo-official.org/problems.aspx | [P] | 1980 missing (no IMO). Footer says only "© International Mathematical Olympiad". No licence. 2026 shortlist marked "Available after IMO 2027". |
| AoPS Contest Collections | Forum threads per problem for thousands of contests | https://artofproblemsolving.com/community/c13 | [S] | The de facto global index. Solutions are user posts, not official. Kaggle scrape counts about 3,980 contests and about 36.5k items (UNVERIFIED count). |
| AoPS Wiki | AMC, AIME, USAMO, IMO and others with community solutions | https://artofproblemsolving.com/wiki/index.php/AMC_Problems_and_Solutions | [S] | Returned 403 to our fetcher. Content confirmed through search snippets only. |
| Kalva (John Scholes) | About 4,100 olympiad problems and 1,700 others. Solutions for about 2,300. | https://prase.cz/kalva/ | [S] | Scholes died in 2019. The site lives on as mirrors. The Technion mirror (webee.technion.ac.il/people/aditya/www.kalva.demon.co.uk) refused connections. |
| imomath.com | Problem PDFs from about 42 countries plus international contests | https://imomath.com/index.cgi?page=problemsFromOlympiads | [S] | Showed a bot check. Older counts: IMO 320, Austrian-Polish 229, APMO 95, Balkan 100, Baltic Way 280, Ibero 144, Nordic 96, Bulgaria 686, Canada 238, China 140, Germany 293, India 227, Iran 176, Poland 528, Romania 414, Russia 670, Serbia 582, Vietnam 305, Belarus 232. |
| imogeometry.blogspot.com | Huge directory of PDF collections and year ranges for most olympiads and TSTs | https://imogeometry.blogspot.com/p/geometry-solved-problem-collections.html | [S] | Best single map of "which years exist" per competition. Geometry focus, but it links full-paper collections. |
| problems.ru (MCCME) | Russian problem base. Moscow MO 2003, ToT 1854, All-Russian 1125, Sharygin 915, Regata 557, Okruzhnaya 416, Prazdnik 404, Lomonosov 372, Moscow oral geometry 185, Belarus 132 | https://problems.ru/ | [P] for Russian contests | Footer "© 2004-… МЦНМО". The catalogue page view_by_source_new.php?parent=204563 shows "Всего задач 8157" for one source subtree. |
| Evan Chen's archive | IMO notes 1997 to 2026, USAMO 1996 to 2026, JMO 2010 to 2026, TSTST 2011 to 2024, TST 2012 to 2025, USEMO 2019 to 2025, ELMO 2010 to 2022, Taiwan TST from 2014 | https://web.evanchen.cc/problems.html | [S] (some [P] for USEMO) | Says USA(J)MO committee solutions are "copyrighted by MAA" and not published. |
| USA TST public archive | IMO TST, EGMO TST, TSTST problems, solutions, statistics | https://github.com/usa-tst-public/USA-TST-Archive | [P] | The new home from 2025/2026. The older vEnhance repo name returned 404. |
| IMO Compendium | IMO longlists and shortlists 1959 to 2009 | https://link.springer.com/book/10.1007/978-1-4419-9854-5 | [S] | Djukić, Janković, Matić, Petrović. 2nd ed., Springer 2011. A 1st-edition PDF circulates at nagyzoli.web.elte.hu/compendium.pdf (copyright status unclear; do not ingest). |
| KöMaL archive | Hungarian school maths journal since 1894. Over 35,000 pages scanned. | https://www.komal.hu/lap/archivum.e.shtml | [P] | English problem statements for 40+ years. Points contest in English since 1997/98. Short Hungarian solutions since 1999/2000. komal.hu returned an empty page to our fetcher. |
| Crux Mathematicorum | Olympiad Corner and problem sections since 1975 | https://cms.math.ca/publications/crux/ | [P] | Open access since 2019 (Intact Foundation). Latest seen: Vol 52 No 8 (Oct 2026). No CC licence stated. |
| MIT MathNet | 30,676 problems, 47 countries, 143 competitions, 17 languages, built from official booklets only | https://mathnet.mit.edu | [S] | Paper arXiv 2604.18584 (CC BY 4.0). Also openreview zPvdG1Va5Q. Dataset licence must be checked separately. |
| AI-MO/aops (HF) | 80,661 problems scraped from AoPS | https://huggingface.co/datasets/AI-MO/aops | [S] | Includes forum material that is not from contests. |
| AI-MO/olympiads-ref (HF) | Official solution PDFs, parsed | https://huggingface.co/datasets/AI-MO/olympiads-ref | [S] | Useful as a pointer to which official booklets exist. |
| olimpicos.net | Spanish and Portuguese mirrors: Baltic Way, OMCPLP, Ibero | https://www.olimpicos.net/ | [S] | |
| guas.info | Russian-language mirror of many olympiads (e.g. Tuymaada) | https://www.guas.info/competit/tuyme.htm | [S] | |
| phoxiv.org | Olympiad mirror (e.g. IZhO) | https://phoxiv.org/olympiads/izho | [S] | UNVERIFIED scope. |
| batmath.it | Kürschák/Eötvös collection PDF | https://www.batmath.it/matematica/raccolte_es/ek_competitions/ek_competitions.pdf | [S] | |

---

## 2. International olympiads (high school)

### 2.1 Global

| Slug | Competition | First | Per year | Est. total | Primary source | Sol. | Licence | Lang | Tag |
|---|---|---|---|---|---|---|---|---|---|
| `imo` | International Mathematical Olympiad | 1959 | 6 (7 in 1960, 1962) | ~404 | https://www.imo-official.org/problems.aspx | N on site (shortlist booklets have solutions) | "© International Mathematical Olympiad"; none stated | 60+ languages | [P] |
| `imo-sl` | IMO Shortlist | ~1959 (lists known from 1960s; systematic from ~1989) | ~30 | ~1,700 shortlist; more with longlists (UNVERIFIED) | https://www.imo-official.org/problems/IMO2024SL.pdf (pattern) | Y | Embargoed until the next IMO | en | [P] |
| `igo` | Iranian Geometry Olympiad | 2014 | 3 levels x 5 | ~180 | https://igo-official.com/ (e.g. /wp-content/uploads/2021/09/IGO_English_2015.pdf). Old domain igo-official.ir | Y (booklets) | None stated | en, fa, many | [P] |
| `kangaroo` | Mathematical Kangaroo (AKSF) | 1991 (France) | ~6 levels x 24 to 30 MCQ | ~5,000+ (UNVERIFIED) | https://www.aksf.org/ (papers via national organisers) | Answers only | Varies by country. Thales Cyprus' Kangourou English contest forbids publishing papers. | 40+ | [P] AKSF; UNVERIFIED per country |
| `tot` | Tournament of Towns | 1980 (as "Olympiad of Three Towns", 1979/80) | 2 rounds x 2 tiers (O/A) x 2 levels, ~50 | ~2,300 (problems.ru has 1,854) | https://www.turgor.ru/en/problems/ (English, 38th to 45th, 48th listed) | P | None stated | ru, en | [P]; turgor.ru returned 403 to fetcher |
| `tot` mirror | Toronto ToT archive 2001 to 2016 | | | | http://www.math.toronto.edu/oz/turgor/archives.php | P | | en | [S] |
| `iom` | International Olympiad of Metropolises | 2016 | ~6 individual (UNVERIFIED) | ~60 | https://imomath.com/srb/zadaci/2019_metropolis_e.pdf | ? | ? | en, ru | [S]; no official site found |
| `sharygin` | Sharygin Geometry Olympiad | 2005 | Correspondence ~24 + final ~24 | ~1,000 (problems.ru 915) | https://geometry.ru/olimp/olimpsharygin.php (e.g. /olimp/2009/zaochsol-e.pdf) | Y | None stated | ru, en (from 2007) | [P] |
| `tuymaada` | Tuymaada (Yakutsk) | 1994 | Senior 8 + Junior 8 | ~500 | https://tuymaada.lensky-kray.ru/en/arhiv/ (1994 to 2021, 2024 to 2026) | P (MCCME book for 1994 to 2013) | None stated | ru, en | [P]. 2026 was the 33rd edition, 2 to 10 July. |
| `zhautykov` | International Zhautykov Olympiad (IZhO) | 2005 | 6 | ~130 | https://izho.kz/ | ? | ? | ru, en, kk | [P] site; papers via imomath.com/srb/zadaci/2014_zhautykov_resenja_e.pdf [S] |
| `silk-road` | Silk Road Mathematical Competition | 2002 | 4 | ~100 | No official archive. Results: scoreboard.bc-pf.org | ? | ? | ru, en | [S] UNVERIFIED |
| `usemo` | US Ersatz Math Olympiad | 2019 | 6 | ~42 | https://web.evanchen.cc/problems.html and the USEMO page | Y | None stated | en | [P] (organiser is Evan Chen) |
| `elmo` | ELMO | 1999 | 6 (+ shortlist ~30) | ~160 + ~400 SL (UNVERIFIED) | https://web.evanchen.cc/problems.html (2010 to 2022, not 2015) | P | None stated | en | [S]; no separate official site found |
| `olympic-revenge` | Olympic Revenge (Brazil) | 2002 | 6 | ~140 | https://artofproblemsolving.com/community/c3274 | N | | pt, en | [S] |
| `duel` | Mathematical Duel (Bílovec, Chorzów, Graz, Přerov) | 1993 | ~20 across categories (UNVERIFIED) | ~600 | https://www.mathematicalduel.eu/index.php/problems/archive | Y (book) | Book: World Scientific doi 10.1142/10544 | en, cs, pl, de | [P] |
| `iom-junior` and others | Junior variants (IZhO junior, Tuymaada junior) | | | | Counted in parent rows | | | | |

### 2.2 Europe

| Slug | Competition | First | Per year | Est. total | Primary source | Sol. | Licence | Lang | Tag |
|---|---|---|---|---|---|---|---|---|---|
| `egmo` | European Girls' MO | 2012 (Cambridge) | 2 days x 3 | ~90 | https://www.egmo.org/ (language list at /data/papers.csv). Hosts: 2026 Bordeaux, 2027 Šibenik, 2028 Vilnius | Y (host booklets) | None stated | many | [P] |
| `balkan` | Balkan MO | 1984 | 4 | ~170 | No central archive. Host sites, e.g. https://bmo2023.tubitak.gov.tr/problems, https://bmo2020.ssmr.ro/problems, https://www.hms.gr/43bmo2026/problems.html | P | None stated | en | [P] per host |
| `balkan-sl` | Balkan MO shortlist | 2007 | ~25 | ~450 (UNVERIFIED) | imogeometry collections | Y | | en | [S] |
| `jbmo` | Junior Balkan MO | 1997 (Belgrade) | 4 (5 in 1997) | ~120 | Host sites, e.g. https://jbmo2023.al/ | P | | en | [P] per host; [S] AoPS wiki 1997_JBMO |
| `jbmo-sl` | JBMO shortlist | 2000 | ~25 | ~600 (UNVERIFIED) | imogeometry | Y | | en | [S] |
| `baltic-way` | Baltic Way (team) | 1990 (Riga) | 20 | ~720 | https://math.olympiaadid.ut.ee/eng/html/?id=bw (1990 to 2025) | Y except 2009, 2019, 2022 | None stated | en | [P]. Mirrors: olimpicos.net/balticway, balticway2009.no/prev |
| `memo` | Middle European MO | 2007 | 4 individual + 8 team | ~240 | https://www.memo-official.org/ with yearly subdomains, e.g. https://memo2020.memo-official.org/ | Y | None stated | en, de, cs, pl, hu, … | [P] |
| `austrian-polish` | Austrian-Polish MC (predecessor of MEMO) | 1978 | ~9 | ~260 (imomath 229) | Kalva, imomath | P | | de, pl, en | [S]. 29 editions, 1978 to 2006. |
| `rmm` | Romanian Master of Mathematics | 2008 | 4 (2008 to 09), 6 from 2010 | ~110 | https://rmms.lbi.ro/rmmYYYY/index.php?id=problems_math | P (some years) | None stated | en | [P]. RMM shortlists exist for 2016 to 2019. |
| `nordic` | Nordic MC | 1987 | 4 | ~160 | https://georgmohr.dk/nmcperm/probl/ (all+sol.pdf, 1987 to 2011, compiled by Matti Lehtinen) | Y to 2011 | None stated | en, Nordic langs | [P] |
| `benelux` | Benelux MO (BxMO) | 2009 | 4 | ~72 | https://www.wiskundeolympiade.nl/old/bxmo/ and https://bxmo.org/results/YYYY | P | None stated | en, nl, fr | [P] |
| `caps` | Czech-Austrian-Polish-Slovak Match | ~2007 (UNVERIFIED) | ~9 | ~150 | Via Czech/Slovak/Austrian national sites | ? | | en | UNVERIFIED |
| `kurschak` | Kürschák (Eötvös until 1938) | 1894 | 3 | ~360 | Hungarian Problem Book I-II (AMS NML 11). KöMaL publishes English versions. | Y (books) | Books are copyrighted | hu, en | [S] collection batmath.it |
| `ofm` | Olympiade Francophone de Mathématiques | 2020 | Junior 4 + Senior 4 | ~56 | https://igm.univ-mlv.fr/~juge/ofm/ and https://ofm2024.mathraining.be/historique.html | ? | None stated | fr | [P] |
| `stpetersburg` | St Petersburg MO (Leningrad MO from 1934) | 1934 | ~50 across grades (UNVERIFIED) | ~3,000 | https://www.pdmi.ras.ru/~olymp/ | P (Fomin and Kokhas book, 1992 to 2008, ~940 problems) | None stated | ru | [P] |
| `ru-239-open` | 239 Lyceum Open Olympiad | ~1990s (UNVERIFIED) | ~20 | ~500 | https://239.ru/ | ? | | ru | UNVERIFIED |
| `moscow-mo` | Moscow MO | 1935 (not held 1942 to 44) | ~30 | ~2,000 (problems.ru 2,003) | https://problems.ru/ and https://olympiads.mccme.ru/mmo/ (UNVERIFIED path) | Y (problems.ru) | © МЦНМО | ru | [P]. Book: 60-Odd Years of MMO 1935 to 1997. |
| `caucasus` | Caucasus MO | 2015 | Junior + Senior, ~8 each (UNVERIFIED) | ~150 | https://cmo.adygmath.ru/ (e.g. /en/node/258; /images/cmo26/cmo26-day2-eng-sol.pdf) | Y (English) | None stated | ru, en | [P]. 2026 was the 11th edition. |

### 2.3 Asia-Pacific and Middle East

| Slug | Competition | First | Per year | Est. total | Primary source | Sol. | Licence | Lang | Tag |
|---|---|---|---|---|---|---|---|---|---|
| `apmo` | Asian Pacific MO | 1989 | 5 (4 hours) | ~190 | https://www.apmo-official.org/ (1989 to 2026, problems and solutions) | Y | None stated; GitHub contributions welcome (developer Leonardo Martinez) | en | [P] |
| `gulf-mo` | Gulf MO (ABEGS) | ~2012 | ~4 | ~60 | https://ds.abegs.org/gmo | ? | | ar, en | [P] |
| `arab-mo` | Arab MO | 2018 (imogeometry) | ~4 to 6 | ~40 | No official site found | ? | | ar, en | UNVERIFIED |

### 2.4 Americas

| Slug | Competition | First | Per year | Est. total | Primary source | Sol. | Licence | Lang | Tag |
|---|---|---|---|---|---|---|---|---|---|
| `ibero` | Iberoamerican MO (OIM) | 1985 | 6 | ~250 | The OEI site removed its archive (see https://link.springer.com/article/10.1007/s11858-022-01338-6). Use imomath collectionIb, AoPS wiki OIM_Problems_and_Solutions, olimpicos.net | P | | es, pt | [S] |
| `centro` | Centroamerican and Caribbean MO (OMCC) | 1999 | 2 x 3 | ~170 | Kalva centro.html; ommenlinea collection 1999 to 2014 | P | | es | [S] |
| `istmo` | Istmo Centroamericano | 2017 | ~6 | ~18 | 2017 to 2019 only (UNVERIFIED) | ? | | es | UNVERIFIED |
| `cono-sur` | Cono Sur MO | 1989 (Uruguay); not held 1990 | 6 | ~220 | https://www.oma.org.ar/enunciados/con1.htm. 2026 = 37th, Lima: https://ompucp.org.pe/conosur2026 | P | | es, pt | [P] |
| `rioplatense` | Olimpiada Rioplatense | ~1990s (UNVERIFIED) | 3 levels x ~6 | ~500 | oma.org.ar (UNVERIFIED) | ? | | es | UNVERIFIED |
| `omcplp` | Lusophon MO (OMCPLP) | 2011 (Coimbra) | 2 x 3 | ~90 | https://omcplp.obmep.org.br/index.php/site/provas | ? | None stated | pt | [P]. Mirror: olimpicos.net/omcplp |

### 2.5 Africa

| Slug | Competition | First | Per year | Est. total | Primary source | Sol. | Licence | Lang | Tag |
|---|---|---|---|---|---|---|---|---|---|
| `pamo` | Pan-African MO | 1987 | 2 x 3 | ~200 | https://pamoofficial.org/ (33 editions; 2027 in Kigali, 4 to 12 May) | ? | | en, fr, ar | [P]. Missing years conflict between sources (imogeometry says no PAMO in 2011, 2014, 2020; AoPS differs). |

---

## 3. Undergraduate competitions

| Slug | Competition | First | Per year | Est. total | Primary source | Sol. | Licence | Lang | Tag |
|---|---|---|---|---|---|---|---|---|---|
| `putnam` | William Lowell Putnam | 1938 (not held 1943 to 45) | 12 now (varied 10 to 14) | ~1,000 | https://kskedlaya.org/putnam-archive/ (problems 1985 to 2025, solutions 1995 to 2025). Official summaries in the American Mathematical Monthly. | Unofficial (Bhargava, Kedlaya, Ng) | Problems "copyrighted by the MAA". Authors ask readers to link, not reproduce. | en | [S] (archive), [P] for MAA books |
| `putnam` books | MAA collections: 1938 to 64 (Gleason, Greenwood, Kelly; 347 problems), 1965 to 84 (Alexanderson, Klosinski, Larson), 1985 to 2000 (Kedlaya, Poonen, Vakil), 2001 to 16 (Kedlaya, Kane, Kane, O'Dorney) | | | | MAA | Y | © MAA | en | [P] |
| `imc` | International Mathematics Competition for University Students | 1994 | 6/day to ~2008; 5/day from 2009; 4/day 2020 to 22 | ~350 | https://www.imc-math.org.uk/ | Y | None stated | en | [P]. Mirror studolymp.com/imc |
| `seemous` | SEEMOUS (MASSEE) | 2007 (Cyprus) | 4 to 5 | ~90 | https://www.massee-org.eu/mathematical/seemous | P | | en | [P]. Compilation 2007 to 2025 on eclass.uoa.gr (Seemous-2007-2025.pdf) [S] |
| `schweitzer` | Miklós Schweitzer | 1949 | ~10 to 12 (open-book, 10 days) | ~800 | https://www.bolyai.hu/files/Schweitzer_2022_English.pdf (pattern); archive https://www.math.u-szeged.hu/~mmaroti/schweitzer/ | P (Székely book covers 1962 to 91) | None stated | hu, en | [P]. AoPS c3253 [S] |
| `vtrmc` | Virginia Tech Regional MC | 1979; suspended after 2022 | ~7 to 8 | ~330 | https://math.vt.edu/undergrad-math/vtrmc.html; https://personal.math.vt.edu/plinnell/Vtregional/ | Y (1987 to 2022) | None stated | en | [P] |
| `ciim` | Iberoamerican Interuniversity MC (CIIM) | 2009 | 2 x 3 | ~100 | https://ciim.uan.edu.co/ (e.g. ciim-2018-pruebas); 2025 hosted in Costa Rica: https://paginas.cimpa.ucr.ac.cr/CIIM/ | P | | es, pt | [P] |
| `ir-university` | Iranian university olympiad | ? | ? | ? | Not found | ? | | fa | UNVERIFIED |
| `ru-student` | Russian/Soviet student olympiads | ? | ? | ? | Not found | ? | | ru | UNVERIFIED |

---

## 4. Group 2: national olympiads by region

The per-year counts below cover the main proof round(s), the national final, and the TST where noted. Lower rounds (school, regional) are often large and answer-only. We list them separately where they matter.

### 4.1 North America

| Slug | Competition | First | Per year | Est. total | Primary source | Sol. | Licence | Lang | Tag |
|---|---|---|---|---|---|---|---|---|---|
| `us-ahsme` | AHSME (pre-AMC 12) | 1950 | 50 (1950s), then 40, 35, 30 (from 1983) | ~1,750 | AoPS wiki AMC hub | Y (MAA) | © MAA | en | [S] |
| `us-amc10` | AMC 10 A/B | 2000 (A/B from 2002) | 25 per paper | ~1,200 | https://maa.org/student-programs/amc/ (UNVERIFIED path); AoPS wiki | Y (MAA solution pamphlets) | © MAA. Older notices allowed non-commercial web posting with notice. 2015 AMC 10B notice allowed paper only. | en | [P]/[S] |
| `us-amc12` | AMC 12 A/B | 2000 | 25 per paper | ~1,200 (overlaps AMC 10 by ~10 to 15 per paper) | as above | Y | © MAA | en | [P]/[S] |
| `us-amc8` | AMC 8 (AJHSME until 1999) | 1985 | 25 | ~1,000 | as above | Y | © MAA | en | [S] |
| `us-aime` | AIME (I and II from 2000) | 1983 | 15 per paper | ~1,070 | as above | Y | © MAA | en | [S] |
| `us-usamo` | USAMO | 1972 | 5 (to 1995), 6 (from 1996) | ~306 | MAA; Evan Chen notes 1996 to 2026 | Committee solutions not published | © MAA | en | [S] |
| `us-usajmo` | USAJMO | 2010 | 6 | ~102 (some shared with USAMO) | as above | as above | © MAA | en | [S] |
| `us-tst` | USA IMO TST, TSTST, EGMO TST | TST ~2000s; TSTST 2011 | ~6 to 9 each | ~300 | https://github.com/usa-tst-public/USA-TST-Archive | Y | None stated | en | [P] |
| `us-hmmt` | Harvard-MIT Math Tournament (Nov, Feb, HMIC) | 1998 | ~250 | ~7,000 | https://www.hmmt.org/www/archive/ | Y | None stated | en | [P] |
| `us-pumac` | Princeton University MC | 2007 (UNVERIFIED) | ~150 | ~2,800 | https://jason-shi-f9dm.squarespace.com/archives | Y | ? | en | UNVERIFIED that this is the official site |
| `us-cmimc` | Carnegie Mellon Informatics and MC | 2016 | ~100 | ~1,100 | https://cmimc.com/archive (2016 to 2026) | Y | ? | en | [P] |
| `us-smt` | Stanford Math Tournament | ~2011 | ~150 | ~2,000 | https://sumo.stanford.edu/smt | ? | ? | en | UNVERIFIED |
| `us-bmt` | Berkeley Math Tournament | ~2012 | ~150 | ~2,000 | https://berkeley.mt | ? | ? | en | UNVERIFIED |
| `us-arml` | ARML | 1976 | ~50 to 60 | ~2,700 | https://www.arml.com/ (public contest files); book 2009 to 14 at arml2.com | Y | ARML "encourages the reproduction of our contest problems for non-commercial, educational purposes" | en | [P] |
| `us-mathcounts` | MATHCOUNTS | 1983 | ~300+ (UNVERIFIED) | ~10,000+ | https://www.mathcounts.org/resources/past-competitions (last year free only) | Answers | © MATHCOUNTS | en | [P]. Mostly below olympiad level. |
| `us-regional` | Other US contests (e.g. NIMO, OMO (both defunct), SMT, others) | | | | Wikipedia list of US regional contests; NIMO/OMO on Evan Chen's Google Drive | | | en | [S] |
| `ca-cmo` | Canadian MO | 1969 | 5 | ~290 | https://cms.math.ca/competitions/cmo/ (problems 1969 to 2026, solutions 1994 to 2026) | Y from 1994 | CMS: personal copying only | en, fr | [P] |
| `ca-cjmo` | Canadian Junior MO | 2020 | 5 | ~35 | as above | Y | as above | en, fr | [P] |
| `ca-euclid` | Euclid (CEMC) and CEMC contests | 1974 (UNVERIFIED) | 10 | ~10 years online | https://cemc.uwaterloo.ca/contests/euclid | Y | © CEMC | en, fr | [P] |
| `mx-omm` | Mexican MO (OMM) | 1987 | 6 | ~240 | https://www.ommenlinea.org/ ; Tzaloa magazine | P | | es | [P] |

### 4.2 Latin America (others)

| Slug | Competition | First | Per year | Est. total | Primary source | Sol. | Licence | Lang | Tag |
|---|---|---|---|---|---|---|---|---|---|
| `br-obm` | Brazilian MO (OBM) | 1979 | 6 (level 3) plus levels 1, 2, U | ~1,500 all levels | https://www.obm.org.br/como-se-preparar/provas-e-gabaritos/ | Y (Eureka! magazine 1998 to 2015; NOIC from 2015) | ? | pt | [P] |
| `br-obmep` | OBMEP | 2005 | large | large | obmep.org.br (UNVERIFIED) | Y | | pt | UNVERIFIED |
| `ar-oma` | Argentine MO (OMA) | 1984 (UNVERIFIED) | several levels | ~1,000 | https://www.oma.org.ar/enunciados/ | P | | es | [P] |
| `pe`, `co`, `cl`, `ve`, `uy`, `cr`, `cu`, … | Other Latin American nationals | | | | AoPS c13, imogeometry | | | es | UNVERIFIED |

### 4.3 United Kingdom and Ireland

| Slug | Competition | First | Per year | Est. total | Primary source | Sol. | Licence | Lang | Tag |
|---|---|---|---|---|---|---|---|---|---|
| `gb-bmo1` | British MO Round 1 | 1965 | 6 | ~360 | https://bmos.ukmt.org.uk/home/bmo.shtml (papers from 1973; 1965 to 72 not online; a PDF holds all BMO problems from 1993) | P | © UKMT | en | [P] |
| `gb-bmo2` | BMO Round 2 (FIST 1972 to 91) | 1972 | 4 | ~200 | as above | P | © UKMT | en | [P] |
| `gb-ukmt` | UKMT Challenges, Kangaroos, MOG | 1996 | large, MCQ | ~5,000 | https://www.ukmt.org.uk/free-past-papers/ | Y | © UKMT | en | [P] |
| `gb-tst` | UK IMO selection tests | ? | ? | ? | Not public | N | | en | UNVERIFIED |
| `ie-irmo` | Irish MO | 1988 | 10 | ~360 | https://www.irmo.ie/problems.html and https://www.irmo.ie/irmo.pdf (1988 to 2023) | P | None stated | en | [P] |

Book: Gardiner, *The Mathematical Olympiad Handbook* covers BMO 1965 to 96 [S].

### 4.4 Western, Northern and Southern Europe

| Slug | Competition | First | Per year | Est. total | Primary source | Sol. | Licence | Lang | Tag |
|---|---|---|---|---|---|---|---|---|---|
| `de-mo` | Mathematik-Olympiade (DDR from 1961, all-German from 1990/91), 4 rounds, grades 3 to 13 | 1961 | ~150+ across grades | ~6,000 (olympiade-mathematik.de lists 6,032 problems and 1,710 solutions) | https://www.mathematik-olympiaden.de/ (archive from 1995, 35th MO) | P (no permanent publication of solutions planned from 35th) | Linking allowed with permission of the Verein | de | [P] reported via [S] https://www.olympiade-mathematik.de/ |
| `de-mo` historical | 1960 to 1994 all 3,775 problems with solutions | | | | https://mathematikalpha.de/wp-content/uploads/2020/01/Aufgaben_und_Loesungen_der_Mathematik-Olympiade_1960-1994.pdf | Y | ? | de | [S] |
| `de-bwm` | Bundeswettbewerb Mathematik | 1970 | 2 rounds x 4 | ~400 | bundeswettbewerb-mathematik.de (online from 1999); Springer book 1970 to 2019, doi 10.1007/978-3-662-61166-1 | Y | © Springer for book | de | [P] |
| `de-tst` | German IMO selection | ? | ? | ? | Not found | | | de | UNVERIFIED |
| `fr-pofm` | French olympiad (POFM, Coupe Animath, TST) | ~2000s | ~20 | ~400 | https://maths-olympiques.fr/?page_id=41 | Y | ? | fr | [P] |
| `fr-academiques` | Olympiades académiques (regional) | 2001 | large | large | Each académie's site | Y | | fr | [S] |
| `it-cesenatico` | Italian MO (Cesenatico final), plus Febbraio and Archimede rounds | 1985 (UNVERIFIED) | 6 (final) | ~240 final | https://olimpiadi.dm.unibo.it/le-gare/gara-nazionale/ (2000 to 2025 with solutions) | Y | ? | it | [P] |
| `es-ome` | Spanish MO (OME) | 1964 (1963/64) | 6 | ~360 | https://www.rsme.es/olimpiada-matematica-espanola/ (1993 to 2010 plus local phases); OME2004.pdf compendium 1963 to 2004 | P | ? | es | [P] |
| `pt-opm` | Portuguese MO (OPM) | 1980s (UNVERIFIED) | ~6 | ~250 | https://www.mat.uc.pt/~opm/ ; olimpiadas.spm.pt | P | | pt | [P] |
| `nl-nwo` | Dutch MO (NWO) finals | 1962 | ~5 | ~300 | https://www.wiskundeolympiade.nl/wedstrijdarchief/finale (2001 to 2026, problems and solutions, English for recent years); 1962 to 1990 report at /old/opgaven/index.html | Y | ? | nl, en | [P] |
| `nl-nwo-r2` | Dutch second round (March) | 2011 | ~8 | ~120 | https://www.wiskundeolympiade.nl/wedstrijdarchief/2e-ronde | Y | | nl | [P] |
| `be-vwo`, `be-omb` | Belgian Flemish (VWO) and francophone (OMB) | 1976 (UNVERIFIED) | ? | ? | vwo.be, omb.sbpm.be (UNVERIFIED) | ? | | nl, fr | UNVERIFIED |
| `lu` | Luxembourg | ? | ? | ? | Not found | | | | UNVERIFIED |
| `ch-smo` | Swiss MO and IMO selection | ~1990s | ~10 | ~300 | https://imosuisse.github.io/ (official archive on GitHub); https://mathematical.olympiad.ch/ | Y | GitHub repo licence not checked | de, fr, it, en | [P] |
| `at-oemo` | Austrian MO (ÖMO) | 1970 | multiple rounds, ~20 | ~900 | https://www.oemo.at/OeMO/aufgaben/ (German and English, with solutions); mirror https://www.math.aau.at/OeMO | Y | ? | de, en | [P]. 2026 was the 57th. |
| `dk`, `no`, `se`, `fi`, `is` | Nordic nationals (Georg Mohr, Abel, Skolornas, Finnish, Icelandic) | various | ~5 to 10 | ~1,000 combined | georgmohr.dk; abelkonkurransen.no; Kalva | P | | Nordic | [S] UNVERIFIED per country |
| `ee`, `lv`, `lt` | Estonian, Latvian, Lithuanian MOs | ~1950s | ~10 to 20 | ~1,500 combined | https://math.olympiaadid.ut.ee/ (Estonia, English booklets) | Y (Estonia) | | et, lv, lt, en | [P] Estonia; others UNVERIFIED |
| `gr-hms` | Greek MO (Thales, Euclides, Archimedes rounds) | 1980s (UNVERIFIED) | ~15 | ~600 | https://www.hms.gr/ (e.g. ARXIMIDHS_2015_ALL.pdf) | Y | ? | el | [P] |
| `cy` | Cyprus MO | ? | ? | ? | Thales Foundation / Cyprus Math Society | ? | | el, en | UNVERIFIED |
| `mt`, `ad`, `li`, `mc` | Small European states | | | | Not found | | | | UNVERIFIED |

### 4.5 Central and Eastern Europe

| Slug | Competition | First | Per year | Est. total | Primary source | Sol. | Licence | Lang | Tag |
|---|---|---|---|---|---|---|---|---|---|
| `pl-om` | Polish MO (Olimpiada Matematyczna) | 1949/50 | 3 rounds, ~18 | ~1,400 (imomath 528 for finals) | https://om.sem.edu.pl/problems/ (47th to 77th OM); older: archom.ptm.org.pl | Y | ? | pl | [P] |
| `pl-omj` | Polish Junior MO | 2005 (UNVERIFIED) | ~15 | ~300 | https://omj.edu.pl/zadania | Y | | pl | [P] |
| `cz-sk-mo` | Czech-Slovak MO (category A, B, C, Z) | 1951/52 | many | ~3,000+ | https://www.matematickaolympiada.cz/en (English booklets 2007 to 2023); https://skmo.sk/?jazyk=en ; DML-CZ yearly reports from 1951/52 | Y | DML-CZ terms | cs, sk, en | [P] |
| `hu-oktv` | Hungarian OKTV (3 categories, 3 rounds) | 1923 (UNVERIFIED) | ~30 | ~1,500 online | https://www.oktatas.hu/kozneveles/tanulmanyi_versenyek_/oktv_kereteben/versenyfeladatok_javitasi_utmutatok/tartalomjegyzek/matematika_II (2004/05 onward, with marking guides) | Y (marking guides) | ? | hu | [P]. Year PDFs also at matek.fazekas.hu [S] |
| `hu-komal` | KöMaL problem contests (A, B, C, K, …) | 1894 | ~250 | ~20,000 to 30,000 (UNVERIFIED) | https://www.komal.hu/lap/archivum.e.shtml | Short sol. from 1999/2000 | © KöMaL | hu, en | [P] |
| `ro-onm` | Romanian National MO and TST | 1949 (UNVERIFIED) | ~30 (grades 7 to 12) | ~2,000 | No official archive found. SSMR: https://ssmr.ro/ ; GIL books | P | | ro | UNVERIFIED |
| `bg-nmo` | Bulgarian MO and TST | 1949 (UNVERIFIED) | ~6 to 12 | ~700 (imomath 686) | https://www.math.bas.bg/bcmi/ (old); UBM books | P | | bg | [P] old site |
| `rs-smo` | Serbian MO (municipal, district, state, SMO, TST) | 1960s (Yugoslav) | many | ~1,500 | https://dms.rs/matematika-srednje-skole/ | Y | ? | sr | [P] |
| `hr`, `si`, `ba`, `mk`, `me`, `al`, `xk` | Croatia, Slovenia, Bosnia, North Macedonia, Montenegro, Albania, Kosovo | various | ~4 to 10 | ~2,000 combined | AoPS, imogeometry | ? | | various | UNVERIFIED |
| `ru-allrussian` | All-Russian Olympiad (ВсОШ), regional and final stages | 1961 (all-Russian; all-Union 1967 to 91) | final ~24 + regional ~24 | ~2,500 (problems.ru 1,125 final) | https://olympiads.mccme.ru/vmo/ (e.g. /vmo/2014/final/v14-1.pdf, /vmo/2026/iii-2.pdf; 2026 = 52nd); full archive https://olympiads.mccme.ru/archives.htm (olympiads.zip, 631 MB); цпм.рф from 2009/10 | Y | None stated | ru | [P] |
| `ru-other` | Okruzhnaya, Regata, Prazdnik, Lomonosov, Euler, Moscow oral geometry, etc. | various | | ~2,000 (problems.ru) | https://problems.ru/ | Y | © МЦНМО | ru | [P] |
| `ua-umo` | Ukrainian MO and TST | 1961 (UNVERIFIED) | ~20 | ~1,000 | imomath, imogeometry | ? | | uk | [S] |
| `by-mo` | Belarus MO | 1950s (UNVERIFIED) | ~20 | ~1,000 (imomath 232; problems.ru 132) | imomath, problems.ru | ? | | ru, be | [S] |
| `md`, `ge`, `am`, `az` | Moldova, Georgia, Armenia, Azerbaijan | various | ~8 | ~1,000 combined | AoPS | ? | | various | UNVERIFIED |

### 4.6 East Asia

| Slug | Competition | First | Per year | Est. total | Primary source | Sol. | Licence | Lang | Tag |
|---|---|---|---|---|---|---|---|---|---|
| `cn-league` | China National High School Mathematics League (全国高中数学联赛; from 2022 全国中学生数学奥林匹克竞赛（预赛）) | 1981 | ~15 (both papers) | ~650 | https://www.cms.org.cn/ ; HIT Press books 1981 to 2018 | Y (books) | © publishers | zh | [P] |
| `cn-mo` | Chinese MO (CMO, national final) | 1986 | 6 | ~240 | Bin Xiong (ed.), *Mathematical Olympiad in China*, World Scientific (latest vol. 29 covers 2023) | Y | © World Scientific | zh, en | [S] |
| `cn-tst` | China TST | 1986 (UNVERIFIED) | ~18 to 24 | ~700 | as above | Y | © | zh, en | [S] |
| `cn-gmo` | China Girls' MO | 2002 | 8 | ~190 | as above | Y | © | zh, en | [S] |
| `cn-wmo` | China Western MO | 2001 | 8 | ~200 | as above | Y | © | zh, en | [S] |
| `cn-smo` | China Southeast MO | 2004 (UNVERIFIED) | 8 | ~170 | as above | Y | © | zh, en | [S] |
| `jp-jmo` | Japan MO (preliminary and final) | 1990 (2026 = 36th) | prelim 12 answer-only (3h) + final 5 | ~600 | https://www.imojp.org/domestic/jmo_overview.html (e.g. /archive/mo2026/jmo/problems/jmo36yqa.pdf) | N official solutions | ? | ja | [P]. Books: 『数学オリンピック 20xx-20xx』 (日本評論社) |
| `jp-jjmo` | Japan Junior MO | 2003 (UNVERIFIED) | ~17 | ~400 | as above | N | | ja | [P] |
| `kr-kmo` | Korea MO (KMO, FKMO) | 1987 (UNVERIFIED) | prelim 20 + final 8 | ~1,000 | https://www.kmo.or.kr/ ; https://www.kms.or.kr/ (board code=junior2 for KJMO) | ? | ? | ko | [P]. Compendium: toomates.net/biblioteca/CompendiumKMO.pdf [S]. FKMO problems are 7 points, 2.5 h. |
| `tw-tst` | Taiwan TST and national olympiad | ? | ~20 | ~400 | Evan Chen (Taiwan TST from 2014); MathNet (Taiwan 2011 to 2024) | P | | zh | [S] |
| `hk-pc` | Hong Kong Preliminary Selection and TST | ~1990s | ~20 | ~600 | https://www.hkage.edu.hk/ (prelim papers 2008 to 2025); https://www.imohkc.org.hk/tests (1997 to 2023) | P | ? | zh, en | [P] |
| `mo-macau` | Macau MO | ? | ? | ? | Not found | | | zh, pt | UNVERIFIED |
| `mn-mo` | Mongolian MO | ~1964 (2007 = 43rd) | ~6 to 12 | ~500 | https://imomath.com/othercomp/Mon/MonMO07.pdf | ? | | mn | [S] |
| `hk-excalibur` | Mathematical Excalibur (HKUST newsletter) | 1995 | ~30 problem corner | ~800 | https://www.math.hkust.edu.hk/excalibur/ | Y | ? | en, zh | [P] |

### 4.7 South-East Asia and Oceania

| Slug | Competition | First | Per year | Est. total | Primary source | Sol. | Licence | Lang | Tag |
|---|---|---|---|---|---|---|---|---|---|
| `vn-vmo` | Vietnam MO (VMO) and TST | 1962 | 6 to 7 + TST 6 | ~600 (imomath 305 for VMO) | Ministry of Education (papers with grading guides since 2022); archives at titanhanoi.vn, mathvn.com | P | ? | vi | [S] mostly |
| `sg-smo` | Singapore MO (junior, senior, open) | 1995 (as SMO) | ~35 x 3 + open round 2 | ~2,500 | https://sms.math.nus.edu.sg/ (rules only seen) | ? | | en | UNVERIFIED archive |
| `th-tmo` | Thailand MO | ~2004 (UNVERIFIED) | ~10 | ~200 | MathNet covers 2007 to 2017 | ? | | th | [S] |
| `ph-pmo` | Philippine MO | 1999 (UNVERIFIED) | ~25 | ~600 | https://pmo.ph/ (10th to 24th PMO with solutions); https://cjquines.com/math/contests.html | Y | ? | en | [P] |
| `my-omk` | Malaysian MO (OMK) | ? | ? | ? | https://omk-persama.org.my/bank_soalan_omk/ | ? | | ms, en | [P] |
| `id-osn` | Indonesia OSN | 2002 (UNVERIFIED) | ~8 | ~200 | No official archive found; defantri.com | ? | | id | [S] |
| `kh`, `la`, `mm`, `bn` | Cambodia, Laos, Myanmar, Brunei | | | | Not found | | | | UNVERIFIED |
| `au-amo` | Australian MO | 1979 | 8 | ~380 | https://www.amt.edu.au/department/past-papers (free 2016 to 2020); AMT books 1979 to 1995 and 1996 to 2011 | Y (books) | © AMT | en | [P] |
| `au-amc`, `au-mcyb` | Australian MC, Maths Challenge for Young Australians | 1978 | large, MCQ | large | AMT | Y | © AMT | en | [P] |
| `nz-nzmo` | New Zealand MO | ~1990s | ~5 | ~150 | nzmos.org (UNVERIFIED) | ? | | en | UNVERIFIED |

### 4.8 South Asia, Central Asia, Middle East, Africa

| Slug | Competition | First | Per year | Est. total | Primary source | Sol. | Licence | Lang | Tag |
|---|---|---|---|---|---|---|---|---|---|
| `in-inmo` | Indian National MO | 1986 (INMO from 1989 at ISI) | 6 | ~230 | https://olympiads.hbcse.tifr.res.in/how-to-prepare/past-papers/ ; https://www.isical.ac.in/~rmo/resources.html ; index https://jpsaha.github.io/MOTP (INMO 1986 to 2026) | Y (recent) | ? | en, hi | [P] |
| `in-rmo` | Indian Regional MO | 1990 | 6 (several regions) | ~500 | as above | Y | ? | en | [P] |
| `in-prmo-ioqm` | PRMO (to 2019), IOQM (from 2020; run by MTA(I)) | 2012 | ~30 | ~400 | as above | Answers | ? | en | [P] |
| `in-tst` | India TST | ? | ~12 | ~200 | AoPS | ? | | en | [S] |
| `bd-bdmo` | Bangladesh MO | 2001 (UNVERIFIED) | ? | ? | Not found | | | bn | UNVERIFIED |
| `pk`, `lk`, `np` | Pakistan, Sri Lanka, Nepal | | | | Not found | | | | UNVERIFIED |
| `kz-mo` | Kazakhstan MO | ~1990s | ~12 | ~400 | imomath, imogeometry | ? | | kk, ru | [S] |
| `uz`, `kg`, `tj`, `tm` | Uzbekistan, Kyrgyzstan, Tajikistan, Turkmenistan | | | | AoPS | | | | UNVERIFIED |
| `ir-mo` | Iran MO (3 rounds) and TST | 1983 (UNVERIFIED) | ~25 | ~800 (imomath 176 for some) | English booklets circulate (e.g. 2024, "41st IMO-selection olympiad", on Scribd); AoPS collections | Y (booklets) | ? | fa, en | [S] |
| `tr-mo` | Turkish MO and TST | 1993 (UNVERIFIED) | ~12 | ~400 | https://bilimolimpiyatlari.tubitak.gov.tr/ (first-round PDFs); matolimp.akdeniz.edu.tr | P | ? | tr | [P] |
| `il-gillis` | Israel National Olympiad (Gillis, Weizmann) | ~1990s (UNVERIFIED) | ~6 | ~200 | https://www.weizmann.ac.il/math/gillis/ (UNVERIFIED direct fetch); AoPS printable 2024 collection https://artofproblemsolving.com/downloads/printable_post_collections/3677058 | ? | | he, en | [S] |
| `il-grossman` | Grossman olympiad (Technion) | ? | ? | ? | Not found | | | he | UNVERIFIED |
| `sa-samc` | Saudi Arabia MC and TST | ~2000s | ~10 | ~150 | https://faculty.ksu.edu.sa/sites/default/files/samc2013.pdf (booklet pattern) | Y | ? | ar, en | [S] |
| `ae`, `kw`, `qa`, `bh`, `om`, `jo`, `sy`, `lb`, `iq`, `ps` | Gulf and Levant nationals | | | | Not found | | | ar | UNVERIFIED |
| `eg`, `ma`, `tn`, `dz` | North African nationals | | | | Not found | | | ar, fr | UNVERIFIED |
| `za-samo` | South African MO | 1966 (UNVERIFIED) | ~20 | ~1,000 | samf.ac.za (UNVERIFIED) | ? | | en | UNVERIFIED |
| `ng`, `ke`, `gh`, `rw`, `ug`, `tz`, `zw`, `bw`, `ci`, `sn`, `cm`, `et` | Sub-Saharan nationals | | | | Not found | | | various | UNVERIFIED |

---

## 5. Sources that are themselves the problem (journals with problem sections)

These are not olympiads, but they contain many olympiad-level problems with solutions. They should be modelled as `source` kinds, not competitions.

| Slug | Name | Since | Est. problems | URL | Tag |
|---|---|---|---|---|---|
| `journal-crux` | Crux Mathematicorum | 1975 | ~5,000 proposals plus olympiad corner | https://cms.math.ca/publications/crux/ | [P] |
| `journal-komal` | KöMaL | 1894 | 20k to 30k (UNVERIFIED) | https://www.komal.hu/ | [P] |
| `journal-kvant` | Kvant (Задачник Кванта) | 1970 | ~3,000 (UNVERIFIED) | kvant.mccme.ru (UNVERIFIED) | UNVERIFIED |
| `journal-excalibur` | Mathematical Excalibur | 1995 | ~800 | https://www.math.hkust.edu.hk/excalibur/ | [P] |
| `journal-tzaloa` | Tzaloa (Mexico) | 2009 | ? | ommenlinea.org | [P] |
| `journal-eureka` | Eureka! (Brazil) | 1998 to 2015 | ? | obm.org.br | [P] |
| `journal-amm` | American Mathematical Monthly problems | 1894 | ~12,000 (UNVERIFIED) | MAA / Taylor & Francis | Paywalled |
| `journal-mathreflections` | Mathematical Reflections | 2006 | ~3,000 (UNVERIFIED) | awesomemath.org (UNVERIFIED) | UNVERIFIED |
| `journal-gazeta` | Gazeta Matematică (Romania) | 1895 | very large | ssmr.ro | UNVERIFIED |

---

## 6. Copyright and licence summary

- No olympiad body we checked publishes problems under an open licence (CC or similar).
- IMO: only a copyright line. The shortlist is embargoed for one year.
- MAA (AMC, AIME, USAMO, Putnam): explicit copyright. Older notices allowed non-commercial web posting with notice. The 2015 AMC 10B notice allowed paper only. Po-Shen Loh's https://live.poshenloh.com/past-contests shows they will grant permission on request. USA(J)MO committee solutions are not published.
- Kedlaya's Putnam archive asks for links, not copies.
- CMS (CMO, Crux): personal copying only, though Crux is open access.
- UKMT, AMT, CEMC, MATHCOUNTS, World Scientific, Springer: copyright, no reuse terms found.
- ARML: explicitly encourages non-commercial, educational reproduction. This is the most permissive notice we found.
- German MO e.V.: allows direct linking with permission; solutions from 1995 not to be republished.
- Kangaroo English (Thales Cyprus): forbids publication.
- MathNet paper is CC BY 4.0. That covers the paper, not the problems.
- Practical stance for olymmath: store statements with full attribution and a link to the primary source. Write our own solutions. Link, do not copy, official solution booklets unless permission is obtained. Ask MAA, CMS, UKMT, AMT, KöMaL and MCCME for permission early. This is a legal question and is out of scope for this note.

---

## 7. Gaps (UNVERIFIED list)

We could not find or confirm a primary archive for:

- Romanian national olympiad and TST (no official archive found).
- Iran (no official archive; booklets circulate).
- China (no official free archive; books only).
- Vietnam before 2022.
- Israel (Gillis site not fetched; Grossman not found).
- Indonesia, Bangladesh, Pakistan, Sri Lanka, Nepal.
- Arab MO and most Gulf and African nationals.
- Rioplatense, Istmo, OBMEP.
- Silk Road and IOM (no official site found).
- SMT, BMT, PUMaC official URLs.
- Iranian and Russian university olympiads.
- CAPS start year.
- PAMO missing years (sources conflict).
- turgor.ru (403), AoPS wiki (403), komal.hu (empty), imomath (bot check), Technion Kalva mirror (refused). Their content was seen through search snippets or other mirrors.

---

## 8. Competition count

| Group | Count |
|---|---|
| Table rows in sections 2 to 4 (international, undergraduate, national) | 157 |
| Extra countries folded into grouped rows (e.g. "hr, si, ba, ...") | ~56 |
| Total | about 210 competitions |

Journals in section 5 are not counted. Many nationals have several rounds. Each round could become its own competition id later.

The ~110 IMO countries are all covered by at least one row. About 60 of them have a confirmed primary source.

---

## 9. How many distinct olympiad problems exist?

We estimate about **120,000** distinct olympiad-level problems (range 90k to 160k). Of those, about **50,000** are proof problems. The rest are short-answer.

Reasoning, bottom up:

1. **International contests.** The sum of the "Est. total" column in section 2 is about 14,000. That includes shortlists (IMO SL ~1,700; JBMO and Balkan SL ~1,000), ToT ~2,300, Sharygin ~1,000, Baltic Way ~720, and Kangaroo ~5,000 MCQ. Without Kangaroo, it is about 9,000.
2. **Undergraduate contests.** Putnam ~1,000, Schweitzer ~800, IMC ~350, VTRMC ~330, others ~200. Total about 2,700.
3. **US short-answer contests.** AHSME, AMC 8/10/12 and AIME together are about 6,000. HMMT, PUMaC, SMT, BMT, CMIMC and ARML add about 17,000. Total about 23,000. MATHCOUNTS would add another ~10,000, but most of it is below olympiad level. We exclude it.
4. **Russian-language contests.** All-Russian, Moscow, St Petersburg and the other problems.ru sources together hold about 10,000 (problems.ru alone lists thousands per source; one subtree shows 8,157).
5. **Hungary.** KöMaL alone is likely 20,000 to 30,000 problems, but many are school-level. We count about 10,000 as olympiad level. OKTV and Kürschák add about 2,000.
6. **German MO.** About 6,000 problems across grades. About 3,000 are olympiad level.
7. **Other nationals.** About 100 countries. Each has, on average, about 25 years of a national final and TST at about 15 problems a year. That gives about 37,000. Strong countries (China, Romania, Bulgaria, Poland, Czech-Slovak, Serbia, Iran, Vietnam, Korea, Japan, Brazil) are well above the average, and small states well below.
8. **Journals.** Crux, Kvant, Excalibur and similar add about 10,000 olympiad-style problems that are not from any contest.

Sum: 9k + 2.7k + 23k + 10k + 12k + 3k + 37k + 10k ≈ **107k**. Add about 10 to 15 percent for lower rounds and contests we have not catalogued. That gives about 120k.

Cross-checks:

- MIT MathNet holds 30,676 problems from official booklets of 47 countries and 143 competitions. That is one quarter of our estimate. It is reasonable because MathNet omits most short-answer contests, journals and the Russian and Hungarian bases.
- AI-MO/aops has 80,661 problems scraped from AoPS. AoPS covers most contests in our list but also forum problems that are not from contests. It misses much of KöMaL, problems.ru and the German MO.
- AoPS Contest Collections hold about 36.5k items in one Kaggle scrape (UNVERIFIED). That is a lower bound on contest problems, since many contests on AoPS are only partly posted.
- Duplication is real. JMO and USAMO share problems. AMC 10 and 12 share problems. Shortlist problems reappear in TSTs and national olympiads. Translations appear as separate items. We expect 10 to 20 percent of raw scraped items to be duplicates. This is already reflected in the "distinct" figure.

Confidence: medium on the order of magnitude, low on the exact figure. The two largest uncertainties are KöMaL (how many of its problems count as olympiad level) and the long tail of national lower rounds.
