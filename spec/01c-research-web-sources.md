# 01c. Research: forums, databases and journals

Date of survey: 2026-10-31. Scope: websites, forums, problem databases and journals that hold olympiad problems with human-written solutions. For each one: what it holds, how big it is, who wrote the solutions, how it is built, how to harvest it, what the robots file, terms and licence say, and the priority for olymmath. Hugging Face datasets are in 01a. Competition bodies and official booklets are in 01b. Formal sources are in 01d.

## 0. Method and conventions

- Probes were run from one residential macOS host on 2026-10-31 with `curl` (UA `Mozilla/5.0 ...`), WebSearch and WebFetch. Only HTML, JSON, robots.txt and PDF text were read. No downloaded code was executed.
- [P] means primary: the site itself, its robots.txt, its terms page, or the source code of a working harvester. [S] means secondary: a search summary, a paper, Wikipedia or a mirror.
- UNVERIFIED means I could not confirm the fact directly. Treat it as a lead.
- "Problems" means distinct problem statements, as counted by the site, unless a note says it is my estimate.
- Priority: **P0** = harvest first, core of olymmath. **P1** = harvest in the first wave. **P2** = later or partial. **P3** = metadata or link only. **X** = out of scope.
- Licence zone (see section 9): **Z1** full text (statement and solution) can be republished. **Z2** statement plus short excerpt and link; solutions stay at the source unless permission is obtained. **Z3** link and metadata only.

## 1. Summary table

| # | Source | URL | Est. problems | Solutions | Lang | Structure | Harvest | Access barrier | Licence zone | Priority |
|---|---|---|---|---|---|---|---|---|---|---|
| 1 | AoPS Community, Contest Collections | https://artofproblemsolving.com/community/c13_contests | ~80K problems in contest threads ([P] AI-MO/aops card); 1,076,712 topics site-wide ([S] arXiv 2501.14275) | User posts, many per problem | en (plus zh, vi, ru, es, etc.) | JSON from `/m/community/ajax.php`; BBCode + LaTeX | AJAX through a real browser session | Cloudflare managed challenge; `ai-train=no` content signal | Z2 (statements), Z3/Z2 for user solutions | P0 (index), statements P0 |
| 2 | AoPS Wiki | https://artofproblemsolving.com/wiki/ | 12,677 articles ([S] Template:Statistics, date unknown); every AMC/AIME/USAMO problem page | User-written wiki solutions | en | MediaWiki | `api.php` (`action=parse`, `action=query&prop=revisions`) | Cloudflare challenge (same wall) | Z2 (AMC text MAA-copyright); wiki solutions Z2 | P0 |
| 3 | problems.ru (MCCME "Задачи") | https://problems.ru/ | ~21K by source tree; 8,157 olympiad problems [P] | Official/editorial, most problems | ru | Static PHP HTML, TeX in `$…$` + MathJax | HTML crawl by id | robots `Crawl-delay: 20` [P] | Z2 (link + id required; non-commercial) | P0 |
| 4 | Crux Mathematicorum (CMS) | https://cms.math.ca/publications/crux/ | v1 1975 - v52 2026, ~10 issues/yr; est. 5 to 6K problems + Olympiad Corner (my estimate) | Reader solutions, edited | en, fr | PDF per issue | PDF download + OCR/LaTeX extraction | None technical | Z2 (all rights reserved, personal use only) [P] | P1 |
| 5 | KöMaL | https://www.komal.hu/ | A/B/C/K series since 1894; archive > 35K pages [S]; problem count UNVERIFIED | Official, partly en, most in hu | hu, en | Dynamic HTML (`feladat?a=feladat&f=B3802&l=en`); scans 1893 to 2001 | HTML crawl; OCR scans | None found | Z2 (UNVERIFIED licence) | P1 |
| 6 | Mathematical Reflections | https://www.awesomemath.org/mathematical-reflections/ | 2006 to 2026, 6 issues/yr, ~24 problems/issue: est. ~2.9K | Reader solutions, edited | en | PDF (problems, solutions, articles) | PDF + OCR | None | Z2 (© AwesomeMath, all rights reserved) | P1 (mathreflect-cli exists) |
| 7 | Evan Chen notes | https://web.evanchen.cc/problems.html | IMO 1997 to 2026, USAMO 1996+, JMO, TST, TSTST, EGMO, ELMO, USEMO: 318 files [P] | Author-written | en | PDF + **TeX source** | Download `.tex` | None | Z2 (no licence stated; ask) | P1 (ask permission → Z1) |
| 8 | Math StackExchange | https://math.stackexchange.com/ | 10,806 questions tagged `contest-math` [P API] | User answers | en | JSON API; XML dumps | SE API v2.3 or community dump | Cloudflare on HTML; API open | **Z1** CC BY-SA (with attribution) | P1 |
| 9 | MathOverflow | https://mathoverflow.net/ | Small olympiad share (tag `puzzle` 140) [P API] | User answers | en | Same as #8 | Same | Same | Z1 CC BY-SA | P2 |
| 10 | imomath.com | https://imomath.com/index.cgi?page=problemsFromOlympiads | ~8.5K national-olympiad problems (sum of per-country counts) [S] | Some solutions | en | CGI HTML | HTML crawl after JS puzzle | SHA-256 proof-of-work JS gate [P] | Z2 | P1 (imomath-cli exists) |
| 11 | Diễn đàn Toán học (VMF) | https://diendantoanhoc.org/ | 638,488 posts site-wide [S]; olympiad sub-forums | User posts | vi | IPS forum HTML | Browser-session crawl | Cloudflare challenge [P] | Z3/Z2 | P2 |
| 12 | Kalva (John Scholes) | mirror https://prase.cz/kalva/ | IMO 1959+, ~400 ISL, APMO, USAMO 1972+, Putnam 1938+ [P meta] | Author-written | en | Static HTML (frames) | HTML crawl | None | Z2 (no licence; author's own text) | P2 |
| 13 | cut-the-knot | https://www.cut-the-knot.org/ | Thousands of pages; olympiad share UNVERIFIED | Author + readers | en | Static HTML | HTML crawl | robots `Crawl-delay: 120` [P] | Z2/Z3 | P2 |
| 14 | Pi Mu Epsilon Journal | https://pme-math.org/pme-journal-problem-department | Problem Dept since 1949; PDFs per issue [S] | Reader solutions | en | PDF scans | PDF + OCR | Cloudflare 403 on curl [P] | Z2 | P2 |
| 15 | Parabola (UNSW) | https://www.parabola.unsw.edu.au/ | 1964 to present, Problems section [P] | Reader solutions | en | HTML + PDF | PDF | None | Z2 (Diamond OA; licence not stated) | P2 |
| 16 | Romanian Mathematical Magazine (SSMRMH) | https://www.ssmrmh.ro/ | Thousands of proposed problems (est.) | Reader solutions | en | WordPress + PDF | PDF | None | Z2 | P2 |
| 17 | Gazeta Matematică B | https://gmb.ssmr.ro/ | Monthly since 1895; online PDFs ~2010s to 2026 [S] | Solutions in later issues | ro | PDF | PDF + OCR | 403 on curl to ssmr.ro/gazeta [P] | Z2 | P2 |
| 18 | THTT (Toán học và Tuổi trẻ) | (no official archive found) | Monthly since 1964; issue 583 = 01/2026 [S] | Reader solutions | vi | Scanned PDFs on third-party sites | PDF OCR | Third-party mirrors only | Z3 | P3 |
| 19 | 中等数学 (Zhongdeng Shuxue) | https://qkcbzx.tjnu.edu.cn/qkjj/zdsx.htm | Monthly since 1982, "数学奥林匹克问题" column [S] | Official + reader | zh | CNKI PDFs (paywall) | Not harvestable openly | Paywall | Z3 | P3 |
| 20 | mathchina.com BBS | http://www.mathchina.com/bbs/ | UNVERIFIED | User posts | zh | Discuz! forum | HTML crawl | Timed out from probe host | Z3 | P3 (mathchina-cli exists) |
| 21 | JMO (Japan) | https://www.imojp.org/domestic/jmo_overview.html | JMO since 1991, JJMO since 2003 [S] | Prelim answers only; final round no solutions [S] | ja | PDF | PDF | Use form required for problem reuse [S] | Z3/Z2 | P2 |
| 22 | olympiads.win.tue.nl/imo | https://olympiads.win.tue.nl/imo/ | Link directory, not maintained since 2007 [P] | none | en | Static | Link mining only | None | Z3 | P3 |
| 23 | Yufei Zhao handouts | https://yufeizhao.com/olympiad/ | 25 PDF handouts [P] | Author | en | PDF | Download | None | Z3 (no licence) | P3 |
| 24 | ProofWiki | https://proofwiki.org/ | Few olympiad items | Users | en | MediaWiki | API | Cloudflare challenge on API [P]; `ai-train=no` | Z1 CC BY-SA (UNVERIFIED version) | P3 |
| 25 | zbMATH Open | https://zbmath.org/ | Metadata on problem literature | Reviews | multi | REST + OAI-PMH | API | 403 on HTML curl [P]; API open | Z1 metadata CC BY-SA 4.0 [S] | P3 (metadata) |
| 26 | Reddit r/mathriddles | https://www.reddit.com/r/mathriddles/ | Puzzles, mixed level | Users | en | Reddit API | Data API (restricted) | API terms | Z3 | P3 |
| 27 | MathLinks | http://www.mathlinks.ro/ → AoPS c89 [P redirect] | Merged into AoPS | none | none | none | Same as #1 | none | none | (part of #1) |
| 28 | Brilliant | https://brilliant.org/ | Community problems; status UNVERIFIED | Users | en | JS app | Sitemap | robots open [P]; ToS proprietary | Z3 | P3 |
| 29 | MATHCOUNTS | https://www.mathcounts.org/ | Middle-school; below olympiad level | Official | en | PDF | none | none | Z3 | P3 |
| 30 | Project Euler | https://projecteuler.net/ | Computational, not olympiad proofs | none | none | none | none | none | none | X |

## 2. Art of Problem Solving (AoPS)

### 2.1 Community "Contest Collections"

What it is.
- Root category is `c13` ("Contest Collections"), at https://artofproblemsolving.com/community/c13_contests [P URL].
- Each node is a *category* with a numeric id. URLs look like `/community/c<category_id>` with an optional slug, e.g. `c3223_imo_shortlist` (all IMO Shortlist years) and `c3381519` (2023 IMO) [P, code comment in https://github.com/MAA-Contest-Tester/search/blob/HEAD/backend/scrape/forum.go].
- Tree shape: c13 → region or contest family → contest → year → items. Items are problem posts, each linked to a *topic* (thread) that holds the user solutions [P same file].
- Topic URL: `/community/c<cat>h<topic_id>`; post URL: `/community/c<cat>h<topic_id>p<post_id>` [P https://github.com/approach0/a0-crawlers/blob/HEAD/crawler-artofproblemsolving.com.py].
- MathLinks (the old forum) now redirects to AoPS `c89` [P: `curl -L http://www.mathlinks.ro/` → `https://artofproblemsolving.com/community/c89`].

Size.
- Project Numina's `AI-MO/aops` has 80,661 problems; its `metadata.path` values look like `Contest Collections/2022 Contests/.../2759376.json`, so it was built by walking the c13 tree [P https://huggingface.co/datasets/AI-MO/aops/raw/main/README.md].
- The DSL-Lab crawl got 1,076,712 topics site-wide; after filtering, about 478K were math topics, giving 600K+ QA pairs [S https://arxiv.org/abs/2501.14275].
- The forum adds at least 15K new math questions a year [S same paper].

Internal AJAX endpoint: `POST https://artofproblemsolving.com/m/community/ajax.php`, form-encoded. Known actions, from working open-source harvesters [P code]:

| Action `a=` | Key params | Returns | Source |
|---|---|---|---|
| `fetch_category_data` | `category_id` | `response.category{category_id, category_name, items[]}`; each item has `item_id, item_text, item_type, post_data{topic_id, post_id, category_id, post_rendered, post_canonical}` | https://github.com/MAA-Contest-Tester/search/blob/HEAD/backend/scrape/forum.go ; https://github.com/C1vical/math_contest_bot/blob/HEAD/aops/community_parser.py |
| `fetch_topic` | `topic_id` | `response.topic{…, posts, tags}` | MAA-Contest-Tester forum.go |
| `change_focus_topic` | `new_topic_id, topic_fetch=initial, fetch_first=1, fetch_all=1` | `response.posts[]` with `post_canonical`, `post_rendered`, `post_time`, plus `initialization_time` | https://github.com/DSL-Lab/aops/blob/main/aops_crawler/aops_crawler/spiders/aops_spider.py |
| `fetch_topics` | `category_id, fetch_before=<unix>, fetch_archived, fetch_announcements` | Topic list; paginate by last `last_post_time` | https://github.com/bigdata-pw/bigdata/blob/HEAD/aops/topics.py ; approach0 crawler |
| `fetch_posts_for_topic` | topic id, start | Posts page | approach0 crawler |
| `search_topics`, `fetch_topic_contents` | as in our scaffold | UNVERIFIED (scaffold guess) | /Users/apple/github/tamnd/aops-cli/aops/forum.go |

Every call also sends the guest session: `aops_logged_in=false`, `aops_user_id=1`, `aops_session_id=<32 chars>`, and cookies `aopsuid=1; aopssid=<sid>` [P DSL-Lab spider and middlewares.py]. The session id is read from the page HTML (`AoPS.session = {…}`) and `aopssid` from `Set-Cookie` [P https://github.com/DSL-Lab/aops/blob/main/aops_crawler/aops_crawler/middlewares.py].

`post_canonical` holds the BBCode source with raw LaTeX. That is the field to store. `post_rendered` is HTML and can be dropped [P DSL-Lab spider deletes `post_rendered`].

Access barrier.
- From the probe host, every `artofproblemsolving.com` URL (community, wiki, `api.php`, ToS) returned **HTTP 403 with `cf-mitigated: challenge`** to curl and to WebFetch [P probe 2026-10-31].
- MAA-Contest-Tester explains: the `cf_clearance` cookie is pinned to the solver's TLS fingerprint. Their fix is to run headless Chromium (chromedp), solve the challenge on `/community/c3413`, then issue `fetch()` calls from inside the page [P forum.go comments].
- Their observed limit: ~1,000 requests pass, then a ~20 s wall. Pacing at 5 req/s (200 ms) with max 4 concurrent avoids it [P forum.go]. approach0 sleeps 0.6 s per request [P]. bigdata-pw uses `curl_cffi` with `impersonate="chrome"` and 0.5 to 1.2 s random delays [P].
- Our own note already records the same wall [P https://github.com/tamnd/Wiki-Notes/blob/HEAD/Spec/9000/9010_aops.md].

robots.txt (fetched 2026-10-31) [P https://artofproblemsolving.com/robots.txt]:
- `User-agent: *` → `Content-Signal: search=yes,ai-train=no,use=reference` and `Allow: /`.
- Named AI crawlers (GPTBot, ClaudeBot, CCBot, Google-Extended, Applebot-Extended, Bytespider, Amazonbot, Diffbot, omgili) → `Disallow: /`.
- The file says restrictions are "express reservations of rights under Article 4" of EU Directive 2019/790 (TDM opt-out).

Terms of Service (https://artofproblemsolving.com/company/tos; 403 to us, read via search snippets) [S]:
- AoPS "does not claim ownership of the Content you submit". Users grant AoPS a non-exclusive, perpetual, sub-licensable licence. That licence goes to AoPS only. Third parties get no reuse right.
- AoPS may set limits on "the maximum number of times … you may access the Services in a given period".
- The AoPS Academy terms (same company) forbid access "through any automated (including use of scrapers, scripts, robots, spiders, or web crawlers)" means [S https://aopsacademy.org/legal/terms]. Whether the main ToS has the same clause is UNVERIFIED.

Risk notes.
- User solutions belong to their authors. AoPS cannot license them to us. Republishing them in full needs each author's consent, or fair use.
- Technical evasion of Cloudflare is a risk under the ToS. The `ai-train=no` signal also binds any later use of the corpus for model training.
- Recommended olymmath model: harvest the **category tree and problem statements** (index + statement), store `post_canonical` privately for dedup and source linking, publish statement + link to the thread + author name + post count. Do not publish user solutions in full unless the author opts in (e.g. a signed contributor licence on olymmath).

### 2.2 AoPS Wiki

- MediaWiki at https://artofproblemsolving.com/wiki/. API: `/wiki/api.php` [P path in our scaffold config.go; used by https://github.com/vqbc/trivial and https://github.com/megagames-me/amc-trainer (`api.php?action=parse&page=AMC_8_Problems_and_Solutions&format=json`)].
- Size: "296,875 edits to 12,677 articles since January 2006; 4,022 files" [S https://artofproblemsolving.com/wiki/index.php/Template:Statistics; date of figure unknown].
- Coverage: every AMC 8/10/12, AIME and USAMO problem, plus many IMO and other olympiads, each with one or more wiki solutions [S our note 9010_aops.md].
- Page pattern: `<Year>_<Contest>_Problems/Problem_<n>` (e.g. `2023_USAMO_Problems/Problem_1`) [P URL tested, 403 behind Cloudflare].
- Harvest: `action=query&list=allpages` to enumerate, then `action=query&prop=revisions&rvprop=content&rvslots=main` for wikitext (keeps `$…$` LaTeX). Same Cloudflare wall as the forum. WikiTeam lists the API as alive, so a WikiTeam XML dump may exist on archive.org [S https://github.com/WikiTeam/wikiteam listsofwikis] (UNVERIFIED).
- Licence: the wiki's copyright page says "Currently deciding on a documentation license" [S https://artofproblemsolving.com/wiki/index.php/AoPS_Wiki:Copyright]. So there is no open licence.
- AMC pages carry the footer "The problems on this page are copyrighted by the Mathematical Association of America's American Mathematics Competitions" [S https://artofproblemsolving.com/wiki/index.php?title=2023_AMC_8].
- Priority P0 for the problem index and AMC/AIME/USAMO statements (Z2). Wiki solutions: Z2 excerpt + link.

## 3. Problem databases

### 3.1 problems.ru (МЦНМО, "Задачи")

- Run by MCCME with ЦПМ support. Problems come from books, journals, circles and olympiads; "most problems come with solutions"; geometry has figures [P https://problems.ru/about_system.php].
- Counts on the source tree root [P https://problems.ru/view_by_source_new.php?parent=]: books and journals 4,628; olympiads and tournaments 8,160; internet resources 7,539; circles 645. Total ≈ 21K.
- Subject tree root [P https://problems.ru/view_by_subject_new.php?parent=]: logic 1,358; algebra 6,061; geometry 12,775; combinatorics 1,029; probability 145; analysis 423; methods 4,283 (with overlap).
- Olympiads subtree [P https://problems.ru/view_by_source_new.php?parent=204563], total 8,157: Moscow MO 2,003; Tournament of Towns 1,854; All-Russian 1,125; Sharygin 915; Math Prazdnik 404; Lomonosov 372; Moscow regatta 557; Okrug 416; others.
- URL patterns [P]: problem `/view_problem_details_new.php?id=<n>` (ids seen: 30320, 103744, 104072); lists `view_by_source_new.php?parent=<id>&start=<k>`; subject `view_by_subject_new.php?parent=<id>`. Ids are sparse (1, 35000, 100000 return "not found").
- Markup: UTF-8 HTML. Formulas are TeX inside `$…$`/`$$…$$`, rendered by MathJax 3.2.2 [P]. Sections: "Условие", "Решение", "Ответ", "Источники и прецеденты использования" (with book, year, chapter, problem number). Difficulty and grade are given.
- robots.txt: `Crawl-delay: 20`; disallow `/feedback.php?`, `/show_document.php?`, `*&into_basket=` [P https://problems.ru/robots.txt]. At 20 s per page, 21K pages take ~5 days.
- Licence ("О копирайте") [P about_system.php]: free for personal non-commercial use, lessons and circles. "When placing problems from our site on other sites, you must link to http://www.problems.ru and the problem's unique number." Rubrics and difficulty tables are MCCME property. All rights reserved.
- Zone Z2 by default. A link + id is required. Ask MCCME for permission to show full solutions (this would make it Z1 and the largest single Russian source). Priority **P0**.

### 3.2 Other MCCME / Russian sites

| Site | URL | What | Notes |
|---|---|---|---|
| ИПС «Задачи по геометрии» | https://zadachi.mccme.ru/ → `/2012` | Geometry problem retrieval system (Prasolov-style) [P title] | cp1251 HTML. Size UNVERIFIED. P2 |
| Moscow Math Olympiad | https://mmo.mccme.ru/ (from olympiads.mccme.ru/mmo/) | Yearly pages `2026/`, results, problems with solutions [P links] | Mostly also in problems.ru. P1 for recent years |
| All-Russian MO | https://olympiads.mccme.ru/vmo/ | Archive pages [P 200] | P1 |
| Tournament of Towns | https://www.turgor.ru/ | Problems and solutions by season [P 200] | Ru + some en. P1 |
| Sharygin geometry olympiad | https://geometry.ru/olimp.php | Problems + solutions PDFs [P 200] | P1 |
| olimpiada.ru, math.ru | https://www.olimpiada.ru/, https://math.ru/lib/ | Olympiad portal; e-library | 502 on probe [P]. Retry later |
| Kvant | kvant.tamnd.com | Done (own corpus) | none |
| Matematika v shkole | none | Methodological journal; few olympiad problems | Paywalled print. P3 metadata only (UNVERIFIED access) |

### 3.3 KöMaL (Hungary)

- Monthly problem journal since 1894 (first volume 1893 to 94) [S http://db.komal.hu/KomalHU/tortenet.phtml].
- Live site: problem pages `https://www.komal.hu/feladat?a=feladat&f=<series><n>&l=en`, monthly lists `feladat?a=honap` [P]. Series seen: A, B, C, K (e.g. B3802, A368, C800, K37) [P].
- English statements exist; many solutions are "(in Hungarian)" [P page text].
- Scans 1893 to 2001 under http://db.komal.hu/scan/ by year folder; searchable text 1893 to 1901 and 1965 to 2019; total > 35,000 pages [S https://www.komal.hu/lap/archivum.e.shtml; search summary]. Also EPA/OSZK mirror https://epa.oszk.hu/00500/00551/ [S].
- robots.txt only disallows `/dir1`, `/dir2/` [P https://www.komal.hu/robots.txt].
- Licence: UNVERIFIED (imprint page timed out). Treat as all rights reserved. Z2. P1 for statements; Hungarian solutions need translation.

### 3.4 imomath.com

- "Problems from Olympiads" at https://imomath.com/index.cgi?page=problemsFromOlympiads [S]. Per-country counts include Bulgaria 686, Russia 670, Serbia 582, Poland 528, Romania 414, Moldova 357, Vietnam 305, Germany 293 … ≈ 8.5K total [S search summary].
- Front page is a JavaScript **SHA-256 proof-of-work** gate ("Checking the Browser …"), then redirect to `index.cgi?page=…&vPWorkS=…&vPWorkQPrime=…` [P curl 2026-10-31]. robots.txt returns 403 [P].
- Our imomath-cli targets `https://www.imomath.com/index.cgi?p=…` [P local code].
- Z2. P1. Solve the puzzle in-process (it is a 4-character brute force over 10 letters) or use a browser.

### 3.5 Kalva

- John Scholes's archive: all IMO from 1959 with solutions, ~400 ISL problems, APMO from 1989, USAMO from 1972, Putnam from 1938 [P meta description https://prase.cz/kalva/].
- Original host kalva.demon.co.uk is dead (no response) [P probe]. Mirror at prase.cz/kalva (frameset, static HTML) [P].
- Kalva notes: longlists public until 1989; from 1990 not published; shortlists are released after the next IMO [P https://prase.cz/kalva/short.html].
- Solutions are Scholes's own text. No licence. Z2. P2 (good for old IMO/USAMO/Putnam second solutions).

### 3.6 cut-the-knot

- Alexander Bogomolny's site, 1996 to 2018. Last update 6 July 2018. Wolfram Foundation is steward; no substantive changes planned [S https://www.cut-the-knot.org/wanted.shtml; https://aperiodical.com/2018/07/alexander-bogomolny-creator-of-cut-the-knot-has-died/].
- robots.txt: `Crawl-delay: 120` for `*`; many folders disallowed [P https://www.cut-the-knot.org/robots.txt]. At 120 s, 10K pages take ~14 days.
- © 1996 to 2018 Bogomolny. Z2/Z3. P2 (inequalities and geometry with olympiad sources).

### 3.7 Other open problem pages

| Site | URL | Notes | Priority |
|---|---|---|---|
| Mathematical Excalibur (HKUST) | https://www.math.hkust.edu.hk/excalibur/ | Free PDF newsletter with problem corner [P 200] | P2 |
| UKMT / BMO | https://bmos.ukmt.org.uk/ , https://ukmt.org.uk/ | Official papers (see 01b) [P 200] | (01b) |
| CMS CMO | https://www.cms.math.ca/competitions/cmo/ | robots **disallows** `/competitions/cmo/problem-sets*` [P https://cms.math.ca/robots.txt] | Respect; use other copies or ask CMS |
| Mathematik-Olympiaden (DE) | https://www.mathematik-olympiaden.de/aufgaben/ | Official problems + solutions PDFs [P 200] | (01b) |
| Korean MS | https://kms.or.kr/ | KMO is run by KMS; problems mostly via AoPS (UNVERIFIED) | P3 |

## 4. Journals with problem sections

| Journal | URL | Span | Format | Access | Licence | Zone | Priority |
|---|---|---|---|---|---|---|---|
| Crux Mathematicorum | https://cms.math.ca/publications/crux/ | v1 (1975) - v52 (2026); "currently 10 issues per year" [P] | Whole-issue PDF `https://cms.math.ca/wp-content/uploads/<yyyy>/<mm>/Wholeissue_<vol>_<n>.pdf`; issue page `/publications/crux/issue?volume=51&issue=1` [P] | "All issues are open to the public"; free online since Jan 2019 (Intact Foundation) [P] | "permission to individual readers … to copy articles for their own personal and private use. All other rights are reserved … No further distribution … without the express written permission" [P https://cms.math.ca/publications/crux/permissions-copyright/] | Z2 | P1; email managing-editor@cms.math.ca for a licence |
| Mathematical Reflections | https://www.awesomemath.org/mathematical-reflections/archives/ | 2006 (Titu Andreescu) - 2026; free online [P] | PDFs `/wp-pdf-files/math-reflections/mr-<yyyy>-<mm>/mr_<n>_<yyyy>_problems_1.pdf`, `…_solutions.pdf` [P]; archive page lists 105 PDFs, 2022 to 2026 [P]; pre-2022 URL guess returned 404 (UNVERIFIED path) | Open | "© 2014 AwesomeMath. All Rights Reserved" [P] | Z2 | P1 |
| Pi Mu Epsilon Journal | https://pme-math.org/pme-journal-problem-department | 1949 - present | Issue PDFs `https://www.pme-math.org/journal/issues/PMEJ.Vol.<v>.No.<n>.pdf`; problem sets 2005 to 2013 and 2018 to 2026 [S] | Cloudflare 403 on curl [P] | UNVERIFIED | Z2 | P2 |
| Parabola (UNSW) | https://www.parabola.unsw.edu.au/issues | 1964 - present; "Diamond Open Access" [P] | Per-volume pages `/issues/1964-1969/year-1964-volume-1` [P] | Open | Licence not stated on site [P] | Z2 | P2 |
| Romanian Math. Magazine (SSMRMH) | https://www.ssmrmh.ro/ | Online, active 2026 [P] | WordPress + PDF | Open | UNVERIFIED | Z2 | P2 |
| Gazeta Matematică B | https://gmb.ssmr.ro/ | 1895 - present; online PDFs `ssmr.ro/gazeta/gmb/<yyyy>/<m>/problemepropuse.pdf` [S] | PDF | 1895 to 2009 collection sold [S] | © SSMR | Z2 | P2 |
| Mathematics Magazine, AMM, College Math. J. (MAA) | via Taylor & Francis | Problem sections; Math Magazine "recently resumed publishing" USAMO solutions [P Evan Chen page] | Paywalled PDF; old years on JSTOR | Paywall | © MAA | Z3 | P3 metadata |
| Math Horizons (MAA) | T&F | Problem section | Paywall | © MAA | Z3 | P3 |
| Mathematical Spectrum | none | 1968 to 2016, ceased [S https://proofwiki.org/wiki/Book:Journal/Mathematical_Spectrum] | Print; miscellany on archive.org is borrow-only [S] | none | © APT | Z3 | P3 |
| THTT | (third-party PDFs) | Since 15 Oct 1964; No. 583 = Jan 2026 [S https://www.ebookbkmt.com/2017/07/tong-hop-cac-so-bao-tap-chi-toan-hoc.html] | Scans | No official open archive found | © NXB Giáo dục | Z3 | P3 |
| 中等数学 | https://qkcbzx.tjnu.edu.cn/qkjj/zdsx.htm | Since 1982; "数学奥林匹克问题" column [S] | CNKI | Paywall | © | Z3 | P3 |
| Kvant | kvant.tamnd.com | Done | none | none | none | none | none |

## 5. Q&A sites and wikis

### 5.1 Math StackExchange and MathOverflow

- Tag counts via `https://api.stackexchange.com/2.3/tags?site=math&inname=contest-math` [P 2026-10-31]: `contest-math` 10,806; `inequality` 31,900; `problem-solving` 4,611; `puzzle` 3,469. MathOverflow: `puzzle` 140 [P].
- API quota without key: 300 requests/day (seen `quota_remaining` 299) [P]. With a free key it is higher (10,000/day, UNVERIFIED for 2026).
- HTML pages are behind Cloudflare (robots.txt returned a challenge) [P].
- Dumps: official dumps left archive.org in 2024. They are now behind login, with terms that bar LLM-training use [S https://wiki.archiveteam.org/index.php/Stack_Exchange; https://search.feep.dev/blog/post/2025-02-20-state-of-stackexchange]. Community mirrors exist: 2025-03-31 and 2025-09-30 on Academic Torrents; 2026-03-31 on HF `flymin/stackexchange_20260331` [S]. The 2025-06-30 dump had watermark rows [S].
- Licence: posts are CC BY-SA 2.5/3.0/4.0 by date [S]. Republishing needs author name, link, licence, and share-alike.
- Zone Z1. Priority P1 (filter `contest-math` and questions that cite a named contest; many are "how to solve ISL 20xx A2" threads). Use the API with a key; it avoids the dump terms.

### 5.2 Wikipedia, ProofWiki, OEIS

- Wikipedia list pages (IMO, national olympiads) give metadata and links; CC BY-SA 4.0; the API works without challenge [P search API call]. Z1 metadata. P3.
- ProofWiki: `ai-train=no` content signal and Cloudflare challenge on `api.php` [P robots and probe]. Licence believed CC BY-SA 3.0 (UNVERIFIED). Few olympiad items. P3.
- OEIS: references for counting answers only. P3 (cross-link).

### 5.3 zbMATH Open

- REST API https://api.zbmath.org and OAI-PMH https://oai.zbmath.org. "All data are allowed to be used under a CC-BY-SA license" [S https://oai.zbmath.org/static/terms-and-conditions.html]. Only FIZ's own data and publisher-authorised data are in the API [S].
- Use: bibliographic metadata on problem books, journal problem sections, MSC 97U40 (problem books), 00A07 (problem collections). P3 metadata.

### 5.4 Reddit, Brilliant, arXiv

- Reddit r/mathriddles: puzzle-level, mixed quality. Reddit Data API terms restrict bulk and commercial use (UNVERIFIED details). Z3, P3.
- Brilliant: robots allows most paths and has a sitemap [P https://brilliant.org/robots.txt]. Community problem pages: status in 2026 UNVERIFIED. Proprietary ToS. Z3, P3.
- arXiv: olympiad-related papers (benchmarks, problem collections). Each has its own licence (many CC BY). Use for metadata and for CC-licensed problem collections only. P3.

## 6. Personal sites and handouts

| Site | URL | Content | Format | Licence | Zone | Priority |
|---|---|---|---|---|---|---|
| Evan Chen | https://web.evanchen.cc/problems.html | IMO 1997 to 2026 notes, USAMO 1996+, USAJMO, TST, TSTST 2011 to 2024, EGMO 2012 to 2021, ELMO, USEMO; 318 linked files, many with `.tex` [P] | PDF + TeX | No licence on page or in TeX header [P]. He writes "the internal problems and solutions, from the actual USA(J)MO committee, are copyrighted by MAA" [P] | Z2 → Z1 if he agrees | P1 (ask first) |
| Evan Chen OTIS Excerpts | https://web.evanchen.cc/otis-excerpts.html | Textbook excerpts | PDF | No licence found [P] | Z3 | P3 |
| Yufei Zhao | https://yufeizhao.com/olympiad/ | 25 handout PDFs (MOP, Canadian camps) [P] | PDF | None stated | Z3 | P3 |
| Kalva | see 3.5 | | | | | |

robots.txt on web.evanchen.cc and imo-official.org contains only the Cloudflare content-signal preamble without a `Content-Signal:` line, so no reservation is expressed there [P].

## 7. Regional forums and sites (non-English)

| Site | URL | Lang | Size | Tech | Barrier | Zone | Priority |
|---|---|---|---|---|---|---|---|
| Diễn đàn Toán học (VMF) | https://diendantoanhoc.org/ | vi | 638,488 posts, 121,262 members [S homepage]; "VMF's Marathon Hình học Olympic" 435 replies [S] | IPS Community Suite, `/forum/<id>-slug`, `/topic/<id>-slug` [S URLs] | Cloudflare challenge, even robots.txt [P] | Z3/Z2 | P2 |
| mathvn.com | https://mathvn.com/ | vi | Blog of exam PDFs | Blogger [P] | None | Z3 | P3 |
| toanmath.com | https://toanmath.com/ | vi | Exam PDF aggregator [P 200] | WordPress | None | Z3 (reposts third-party PDFs) | P3 |
| mathchina.com | http://www.mathchina.com/bbs/forum.php | zh | UNVERIFIED | Discuz! [P our CLI] | Timeout from probe host [P] | Z3 | P3 |
| AoPS China and other AoPS language forums | AoPS community | zh, vi, es … | Part of #1 | Same AJAX | Same | Same as #1 | with #1 |
| JMO / JJMO | https://www.imojp.org/ | ja | JMO prelims with answers; finals no solutions [S] | PDF | Reuse needs a form ("問題使用申請書") faxed to the foundation [S] | Z3 (Z2 after permission) | P2 |
| CPM (Ukraine) | https://www.cpm.org.ua/ | uk | UNVERIFIED | HTML | None | Z2 | P3 |

## 8. PDF-only sources and OCR

Sources that ship only PDF (or scans) and need extraction:

| Source | PDF type | Need |
|---|---|---|
| Crux 1975 to 2026 | Born-digital for recent years; old volumes are scans (typewritten in v1 to v10, UNVERIFIED) | Math OCR + layout |
| Mathematical Reflections | Born-digital LaTeX PDFs | Text + math extraction |
| KöMaL scans 1893 to 2001 | Scans (TIFF G4 / PNG on CD) [S] | OCR, Hungarian |
| Pi Mu Epsilon Journal | Old scans (typed) | OCR |
| Gazeta Matematică, THTT, Parabola old issues | Scans | OCR (ro, vi, en) |
| National olympiad booklets (01b) | Mixed | Math OCR |
| JMO, Evan Chen PDFs | Born-digital (Evan has TeX: skip OCR) | Prefer TeX |

Extraction tools (GitHub API, 2026-10-31) [P `gh api repos/...`]:

| Tool | Repo | Licence (repo) | Stars | Last push | Notes |
|---|---|---|---|---|---|
| marker | datalab-to/marker | Apache-2.0 (model weights may carry a separate licence, UNVERIFIED) | 40.7K | 2026-10-29 | PDF→Markdown with LaTeX; uses surya |
| surya | datalab-to/surya | Apache-2.0 | 21.5K | UNVERIFIED | OCR + layout |
| olmOCR | allenai/olmocr | Apache-2.0 | 19.8K | 2026-03 | VLM-based; good on scans |
| MinerU | opendatalab/MinerU | NOASSERTION (custom) | 82.8K | UNVERIFIED | Formula detection + LaTeX |
| docling | docling-project/docling | MIT | 69.9K | UNVERIFIED | Layout; weaker math |
| nougat | facebookresearch/nougat | MIT (code) | 10K | 2025-02 | Academic PDFs → LaTeX; idle |
| GOT-OCR 2.0 | Ucas-HaoranWei/GOT-OCR2.0 | none listed | UNVERIFIED | UNVERIFIED | |
| LaTeX-OCR (pix2tex) | lukas-blecher/LaTeX-OCR | MIT | UNVERIFIED | UNVERIFIED | Single formulas |
| Mathpix | commercial API | Paid | none | none | Best-known math OCR; per-page cost |
| Vision LLM | none | none | none | none | Use as second pass and for verification |

Recommendation: born-digital PDFs → `pdftotext` + marker; scans → olmOCR or MinerU; then a vision-LLM pass that compares the rendered LaTeX with the page image; keep page image crops as provenance. Mark every OCR-derived text with `extraction: ocr/<tool>@<version>` and a confidence score.

## 9. Legal notes and licence-zone policy

### 9.1 Facts and findings

- **Problems can be copyrighted.** US courts treat exam questions as "original works of authorship". *ETS v. Katzman*, 793 F.2d 533 (3d Cir. 1986) [P case citation; URL UNVERIFIED]. *ETS v. Simon*, 95 F. Supp. 2d 1081 (C.D. Cal. 1999): paraphrasing does not avoid infringement [P case citation; URL UNVERIFIED]. Ideas, theorems and answers are not protected; the wording is.
- **Solutions are clearly protected** literary works of their authors (forum users, journal solvers, Evan Chen, Kalva).
- **IMO:** the General Regulations (approved at IMO 2025) contain no copyright or licensing clause; grep for "copyright", "intellectual", "licen" found nothing [P https://www.imo-official.org/assets/documents/imo-general-regulations.pdf]. §6.6 keeps the Shortlist confidential until the end of the next IMO [S]. The site footer says "© International Mathematical Olympiad" [P https://www.imo-official.org/problems/]. Problems are published in many languages for free. Reuse of statements with attribution is common practice. Status: Z2 by default; ask the IMO Board for a written statement.
- **MAA AMC:** past papers say that after the contest window "permission to make copies of problems in paper or electronic form including posting on web-pages for educational use is granted without fee provided that copies are not made or distributed for profit or commercial advantage and that copies bear the copyright notice" (2015 AMC 10A wording); the 2015 AMC 8 allowed paper only [S https://www.isinj.com/mt-amc10/2015-AMC-10-A-Solutions.pdf]. Po-Shen Loh's archive hosts AMC "with official legal permission of the MAA" [S https://live.poshenloh.com/past-contests]. MAA owns trademarks AMC, AIME, USAMO [S https://trademark.justia.com/owners/the-mathematical-association-of-america-1271861]. Evan Chen: committee USAMO solutions are copyrighted by MAA [P].
- **AoPS:** user posts are owned by users; AoPS has a licence only for itself [S ToS]; robots.txt says `ai-train=no, use=reference` as an EU DSM Art. 4 reservation [P]; wiki has no licence [S].
- **problems.ru:** reuse allowed for personal non-commercial use; other sites must link to problems.ru and the problem number [P].
- **Crux:** personal use only; redistribution needs written permission [P].
- **CC BY-SA sources:** Stack Exchange posts [S], Wikipedia, zbMATH API data [S], ProofWiki (UNVERIFIED).
- **EU TDM:** Art. 3 (research organisations) and Art. 4 (anyone, unless the owner opts out in machine-readable form) of Directive 2019/790. AoPS and ProofWiki have opted out for training via content signals [P robots].

### 9.2 Recommended licence-zone policy for olymmath

| Zone | What olymmath shows | Applies to (default) | How to move up |
|---|---|---|---|
| **Z1 Full** | Statement + full solutions, with author, source URL, licence | CC BY / CC BY-SA sources (Math SE, MO, zbMATH metadata, Wikipedia); public-domain items (pre-1955 journals whose authors died > 70 years ago; check per country); sources that grant written permission; olymmath's own contributors under CC BY-SA 4.0 | Written permission from rights holder |
| **Z2 Statement + link** | Problem statement (short factual text, attributed), metadata (contest, year, number, topic, answer), our own original solution if written in-house, a link to every human solution at the source, author names, a short excerpt (≤ ~2 lines) of each external solution | IMO, national olympiads, AMC (with MAA notice, non-commercial), AoPS statements, problems.ru (with id link), Crux, MR, KöMaL, Kalva, Evan Chen, imomath, cut-the-knot, PME, Parabola, RMM | Ask; record permission in `licence.yml` per source |
| **Z3 Link only** | Title, metadata, link | Paywalled journals (AMM, Math Mag, CMJ, Math Horizons, 中等数学), forum posts with unclear status (VMF, mathchina), third-party scan mirrors (THTT, toanmath), JMO until the form is filed, Brilliant, Reddit | Permission or a lawful open copy |

Operational rules:
1. Store harvested raw text privately (for dedup, search and linking). Publish only what the zone allows.
2. Every record carries `source_url`, `source_id`, `author`, `licence`, `zone`, `retrieved_at`, `extraction`.
3. Honour robots.txt crawl delays (problems.ru 20 s, cut-the-knot 120 s) and disallows (CMS CMO problem sets).
4. Do not use AoPS or ProofWiki content to train models (`ai-train=no`).
5. Prefer official statements (01b) over forum copies. Use AoPS mainly as an index and as a pointer to solutions.
6. Send permission requests early to: MCCME (problems.ru), CMS (Crux), AwesomeMath (MR), Evan Chen, KöMaL/MATFUND, MAA (AMC), IMO Board, imojp.org. A yes moves a source to Z1 and is the single biggest gain for a "human-written first" site.
7. Offer a takedown address and a fast removal process.

## 10. Harvest order (proposal)

1. **P0** AoPS c13 tree (category ids, problem statements, topic ids, post counts) via an in-browser session at ≤ 5 req/s; AoPS Wiki AMC/AIME/USAMO pages; problems.ru full crawl at 20 s delay.
2. **P1** Math SE `contest-math` via API (Z1); Crux PDFs; Mathematical Reflections PDFs; Evan Chen TeX (after asking); imomath; KöMaL live site; MMO, VMO, Tournament of Towns, Sharygin.
3. **P2** Kalva mirror; cut-the-knot; PME; Parabola; RMM; Gazeta Matematică; VMF; JMO.
4. **P3** metadata only: MAA journals, 中等数学, THTT, zbMATH, Wikipedia, Brilliant, Reddit.

## 11. Open questions

- AoPS main ToS: exact anti-scraping wording (page is behind Cloudflare and Wayback returned 429). Re-read in a browser.
- Exact count of AoPS c13 problems today (walk the tree once to count).
- Mathematical Reflections pre-2022 PDF paths.
- KöMaL and Parabola licence terms.
- Whether AoPS would grant an index/statement licence or an API key.
- Status of Brilliant community problems in 2026.
