# olymmath-agent

[![ci](https://github.com/tamnd/olymmath-agent/actions/workflows/ci.yml/badge.svg)](https://github.com/tamnd/olymmath-agent/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/tamnd/olymmath-agent?sort=semver)](https://github.com/tamnd/olymmath-agent/releases)
[![licence](https://img.shields.io/badge/licence-Apache--2.0-blue)](LICENSE)

This repository is the toolchain of [olymmath](https://github.com/tamnd/olymmath). olymmath collects the problems and the solutions of all mathematical olympiads. It puts the human text first and shows the source of each text. The site is at https://olymmath.tamnd.com.

The `om` binary does these tasks:

- It builds the site from the content repository.
- It checks the content files and reports the errors.
- It harvests the sources and keeps each capture in a raw store (milestone M3).
- It merges duplicate problems and selects one canonical statement for each problem (milestone M4).
- It writes and grades AI solutions, with a label on each (milestone M7).

## Install

Download a binary from the [releases](https://github.com/tamnd/olymmath-agent/releases). Each archive has a checksum and a build attestation. To check an archive, use this command:

```sh
gh attestation verify om_v0.1.0_linux_amd64.tar.gz --repo tamnd/olymmath-agent
```

You can also build from the source. You need Go 1.27 or later.

```sh
go install github.com/tamnd/olymmath-agent/cmd/om@latest
```

## Use

Get the content repository and build the site:

```sh
git clone https://github.com/tamnd/olymmath
om check -content olymmath
om site build -content olymmath -out build/site
```

Open `build/site/index.html` in a browser, or serve the directory with any static file server.

The build renders the pages with one worker for each CPU. It keeps the rendered formulas in `olymmath/site` in the cache directory of the user (`~/Library/Caches` on macOS, `~/.cache` on Linux), about 5 MB, so the next build is faster. `-cache off` turns the cache off. The output is the same with and without the cache. Set `OM_CPUPROFILE=file` to write a CPU profile of any command for `go tool pprof`.

Each competition page has a year grid. A row is a year and a cell is a position. The colour of a cell gives the class of its text, a dot marks a human solution, and an empty cell is a position with no problem yet. The years come from `first_year`, `last_year` and `gaps` in the register. The page `/coverage/` gives the measures C0 to C8 of spec 14 for each competition.

Use `om help` to see all the commands.

## API and MCP

The build writes the read records of the API as JSON under `build/site/api/`: `/api/index.json`, `/api/problem/<tag>.json`, `/api/solutions/<tag>.json` and `/api/edition/<competition>/<edition>.json`. A problem record has no answer and no solution, so that a practice tool can hide them.

`om serve` answers the same records over HTTP, and adds search and a random pick:

```sh
om serve -content olymmath -addr 127.0.0.1:8080
curl 'http://127.0.0.1:8080/api/problem/0JY6M52386'
curl 'http://127.0.0.1:8080/api/search?q=floor&limit=5'
curl 'http://127.0.0.1:8080/api/random?competition=imo&topic=N&band_min=4'
```

The same server runs the MCP server at `/mcp`. `om serve -mcp` runs it on stdin and stdout instead, for an MCP client that starts the command. Its tools are `get_problem`, `get_solutions`, `search_problems`, `list_edition` and `random_problem`.

## Edit the content

To add a problem, a solution or a tag, use these commands:

```sh
om new problem imo/2025/p1 -source imo-official -url https://www.imo-official.org/problems.aspx -content olymmath
om new problem -loose -content olymmath
om new solution imo/2025/p1 -origin human -content olymmath
om tag new -n 3
om fmt -content olymmath
om audit -content olymmath
```

The `om new` command makes a file with the correct front matter and adds the new tag to `registers/tags.tsv`. The command does not write over a file. The `om fmt` command puts the front matter keys in the standard order. Use `om fmt -check` in CI to find files that are not formatted. The `om audit` command runs all the rules. Use `om audit -list` to see the rules.

## Anchor to the official papers

```
om anchor -content olymmath imo-official
```

The `om anchor` command gets the official English IMO papers and the shortlist booklets from 2006 into the raw store and reads their text with `pdftotext`. You must install Poppler to get `pdftotext`. The command compares each problem of a paper with the canonical text of the same problem in the content. When the words agree, the command adds a link to the paper and its page. When the words do not agree, the command adds a review item of kind `official` to the review queue. The command does not change the words of a canonical text. Use `-from` and `-to` to select the years.

A booklet has the problems and the official solutions. The text of each problem ends where its solution starts, and the command removes the running heads, the page numbers and the headings of the parts. A booklet is published after the IMO of the next year, so the command does not ask for a booklet of the current year.

The text layer of a PDF loses the layout of the formulas, so the text of a paper is never a statement. But a dataset row can be a faithful transcription of the paper. The command traces a transcription to the paper when two tests pass. First, the words of the two texts agree in their order: twice the longest common sequence of words, over the words of both texts, is 0.9 or more. Second, the two texts have the same set of numbers. The old papers set the point O so that the text layer gives 0, so a 0 that only the paper has does not count. A traced statement gets `traced: S3`, where S3 is the link to the paper, and it becomes the canonical statement. It has the class of the paper, H1 (spec 06.2). The site shows it as a transcription checked against the official paper.

Some datasets give the IMO problems of a year with no problem number. The command `om import -anchor -comp imo` compares each such text with the official papers in the raw store and gives it the position of the problem whose words agree best. The best signal must be 0.3 or more, and it must be at least 0.15 above the second best signal. Otherwise the row has no position, and the import skips it.

A text in zone Z3 is a link only, so the import does not take it. For example, the NuminaMath rows from the notes files have the texts of Evan Chen, and the register gives these texts the zone Z3. With `-anchor`, the import uses these texts only to find the position of a row with no number in other competitions, such as an AoPS row of a USAMO problem. The same thresholds apply. Two copies of one text do not compete with each other.


### Record the papers in all languages

```sh
om papers -content olymmath imo-official
```

The `om papers` command reads the page of each IMO year on imo-official.org and records the official paper in each language in the edition file, under `official`. Each entry has the name of the language, the code that the organizer uses and the URL of the PDF. The edition page of the site lists these papers. The texts stay in the PDFs, so the command stores only links.

### Check a sample against the papers

```sh
om sample -content olymmath -n 200 imo-official
```

The `om sample` command takes a random sample of the IMO and ISL problems that have a text in an official English paper. It compares the canonical text of each problem with the text of the paper and writes the report `reports/sample/imo-official.md`. A problem is exact when the order signal is at least 0.9 and the numbers agree, and it agrees when the order signal is at least 0.6 and the numbers agree. For each other problem, the report gives the two texts, so that a person can compare them. The same seed gives the same sample.

## Read a curated archive

Some curated sources give their problems as TeX files. For such a source, `om seed` gets each file into the raw store, converts each problem to Markdown with math and writes one seed row for each problem. The math does not change. A problem with TeX that has no Markdown form, for example a figure, gets no row. The command prints the reason for each such problem. The first archive is the Putnam archive of Kiran Kedlaya, from 1985:

```sh
om seed -content olymmath kedlaya-putnam
om import -content olymmath -comp putnam -promote
```

An import never changes the canonical text of an existing problem, unless you give `-promote`. With `-promote`, a candidate of class H1 or H2 becomes the canonical statement when it ranks above the canonical statement, its text agrees with the canonical text (the signal T is 0.5 or more) and the answers do not disagree. The old text stays in the file as another statement. Each other better candidate goes to the review queue.

### Record the USA TST papers

```sh
om papers -content olymmath usa-tst-archive
```

The USA TST Archive on GitHub has the problems, solutions and statistics of the IMO TST, the TSTST and the EGMO TST. The archive states no licence, so its texts are in zone Z2. The `om papers` command lists the files of the archive with the GitHub API and records each problems file as the official English paper of its edition. When the corpus does not have the edition, the command writes the edition file.

### Place the rows of a national olympiad

```sh
om papers -content olymmath om-sem
om papers -content olymmath bundeswettbewerb
om fetch -content olymmath om-sem bundeswettbewerb
om import -content olymmath -anchor -comp pl-om,de-bundeswettbewerb
om anchor -content olymmath om-sem
```

The `om papers` command reads the listing page of the Polish Mathematical Olympiad (om.sem.edu.pl) or of the Bundeswettbewerb Mathematik (mathe-wettbewerbe.de) and records each paper in the edition file. An entry can name the paper of the layout that the file holds, such as `round2`, and the source of the file. The papers of the first and the second stages of the Polish olympiad have the paper `stage1` or `stage2`. Their problems have no position in the content, and they only help to find the rows of those stages. The `om fetch` command gets the papers into the raw store, with a delay of two seconds between requests.

With `-anchor`, `om import` places each row of a competition that has official papers. The papers are in Polish or in German, so the command compares the formulas, not the words: the numbers, the letters alone and the names of points, such as ABC. A row gets the position of the best segment when the signal is at least 0.3 and at least 0.2 above the best signal of another position. A row that is nearest to a segment of an earlier stage gets no position. The `om anchor` command then links each problem to its paper and page when the paper places the canonical text at the position of the problem. Else the problem gets a review item.

The British Mathematical Olympiad works the same way, with the source `ukmt`:

```sh
om papers -content olymmath ukmt
om fetch -content olymmath ukmt
om import -content olymmath -anchor -comp uk-bmo1,uk-bmo2
om anchor -content olymmath -missing -pages pages.tsv ukmt
```

The `om papers` command reads the BMO page of bmos.ukmt.org.uk. A file of BMO1 or of the single olympiad before 1992 goes to `uk-bmo1`, and a file of BMO2 or of FIST goes to `uk-bmo2`. The edition is the year of the file, which is the year in which the school year ends. The papers are in English, so the command compares the words. Many papers have two pages side by side, and the first page holds the instructions, so the reader cuts the pages in half and drops the instructions.

With `-missing`, `om anchor` makes a new problem for each position of a paper that has no problem in the content. The problem has no text. Its only statement links to the paper and the page, and the site tells the reader to read the problem in the paper. Use this when the licence of the papers does not let the content hold their text. The papers before 1993 are scans with no text, so the command cannot read their positions. The file of `-pages` gives them: one edition on each line, a tab, and the page of each problem from p1, such as `uk-bmo1/1975<TAB>1 1 1 2 2 2 2 2`.

The Indian olympiads work the same way, with the source `hbcse`:

```sh
om papers -content olymmath hbcse
om fetch -content olymmath hbcse
om import -content olymmath -anchor -comp in-rmo,in-inmo
om anchor -content olymmath -missing -pages pages.tsv hbcse
```

The `om papers` command reads the page of past papers of olympiads.hbcse.tifr.res.in and records each paper of the RMO and the INMO. The edition is the year in which the olympiad is held. Some years of the RMO have more than one paper. The CRMO papers of 2012 to 2016 are `paper1` to `paper4`, and the Mumbai paper is `mumbai`. The Kerala and Tamil Nadu paper of 2018 and the KV and JNV paper from 2023 are `paper2`. A question paper wins over a solutions file of the same paper. The command skips the files of PRMO, IOQM and the other olympiads, and the old links of the page, which give no file. The site gives no open licence, so the content only links to the papers.

The Japanese olympiads work the same way, with the source `imojp`:

```sh
om papers -content olymmath imojp
om fetch -content olymmath imojp
om import -content olymmath -anchor -comp jp-jmo,jp-jjmo
om anchor -content olymmath -missing -pages pages.tsv imojp
```

The `om papers` command reads the overview pages of the JMO and the JJMO on www.imojp.org. The edition of the JMO n is the year 1990 + n, and the edition of the JJMO n is the year 2002 + n. The final round is the paper of the edition. The preliminary round is the paper `prelim`, which has no positions. The command skips the single paper of the JJMO of 2003 to 2008. The papers are in Japanese, so the command compares the formulas. Most papers before 2024 are HTML pages of scans with no text, so the pages file gives their pages. The site forbids reproduction without permission, so the content only links to the papers.

The Australian Mathematical Olympiad works the same way, with the source `amt`:

```sh
om papers -content olymmath amt
om fetch -content olymmath amt
om anchor -content olymmath -missing amt
```

The `om papers` command reads the past papers page of amt.edu.au and records the AMO paper of each year that it lists. Each file holds the eight problems and then the solutions. The page lists only the years 2016 to 2020. The papers say "© Australian Mathematics Trust" and give no open licence, so the content only links to the papers. The robots.txt of the site asks for 10 s between requests, and the fetcher waits that long.

The Virginia Tech Regional Mathematics Contest works the same way, with the source `vtrmc-official`:

```sh
om papers -content olymmath vtrmc-official
om fetch -content olymmath vtrmc-official
om import -content olymmath -anchor -comp vtrmc
om anchor -content olymmath -missing vtrmc-official
```

The `om papers` command reads the VTRMC page of math.vt.edu and records the exam of each year that it lists. The year is the text of the link. The file of an exam is named by the number of the contest, so the command makes the URL from that number. Two links of the page are wrong: the link of 2009 goes to the exam of 2010, and the link of 2011 names a file that does not exist. The contest ran from 1979 to 2022 and was not held in 2020 and 2021. The page states no licence, so the content only links to the papers. The robots.txt of the site asks for 10 s between requests.

The International Mathematics Competition for University Students works the same way, with the source `imc-official`:

```sh
om papers -content olymmath imc-official
om fetch -content olymmath imc-official
om import -content olymmath -anchor -comp imc
om anchor -content olymmath -missing imc-official
```

The `om papers` command reads the problems page of each year on imc-math.org.uk, from 1994 to this year. It records one paper for each day, or one paper when the year has a single file. When a year has more than one file for a day, it takes the TeX "Questions" file first, then "Problems", then "Problems and Solutions", then "Solutions". The "Problems" files of 2015 to 2017 are prints of a web page and lose text at the right edge. From 2015 the second day numbers its problems 6 to 10, so a paper whose numbers run on from the paper before it is read with those numbers. The pages say "© IMC" and give no open licence, so the content only links to the papers. The site has no robots.txt.

The Pan-African Mathematics Olympiad works the same way, with the source `pamo-official`:

```sh
om papers -content olymmath pamo-official
om fetch -content olymmath pamo-official
om import -content olymmath -anchor -comp pamo
om anchor -content olymmath -missing pamo-official
```

The problems page of pamoofficial.org is HTML and holds all the years. The section of a year has the id `pamo-YEAR` and holds its six problems, numbered 1 to 6. So the paper of a year is the page with that id as the fragment, such as `https://pamoofficial.org/problems/#pamo-2025`. `om fetch` gets the page once, and `om anchor` reads the section of each paper from it. The page states no licence, so the content only links to it. The robots.txt of the site allows all paths for other agents and says `search=yes, ai-train=no, use=reference`.

The Tuymaada olympiad works the same way, with the source `tuymaada-official`:

```sh
om papers -content olymmath tuymaada-official
om fetch -content olymmath tuymaada-official
om import -content olymmath -anchor -comp tuymaada
om anchor -content olymmath -missing tuymaada-official
```

The archive page of tuymaada.lensky-kray.ru links one English booklet for each day of the mathematics olympiad, from 2018 and from 2020. A booklet holds the four problems of the senior league, then the four problems of the junior league, then the solutions. The first day has the problems 1 to 4 and the second day the problems 5 to 8. So a booklet is two papers with the same URL, `senior` and `junior`, and `om anchor` reads the part of each league from it. The booklets say "© Republic of Sakha (Yakutia) ministry of education and science" and give no licence, so the content only links to them.

The Georg Mohr Contest of Denmark works the same way, with the source `georgmohr`:

```sh
om papers -content olymmath georgmohr
om fetch -content olymmath georgmohr
om import -content olymmath -anchor -comp dk-georgmohr
om anchor -content olymmath -missing georgmohr
```

The page georgmohr.dk/gmopg/ lists the papers of the second round, which is the national olympiad, from 1991. A script on the page makes the links from the attributes of a table, so `om papers` reads those attributes. A year gets its paper in English when the page has one, from 2000, and its paper in Danish before that. The Danish papers number the problems "Opgave 1." and so on. The page says that the problems may not be used for commercial purposes, including the training of language models, so the content only links to the papers. The site has no robots.txt.

The Portuguese Mathematical Olympiad works the same way, with the source `spm-opm`:

```sh
om papers -content olymmath spm-opm
om fetch -content olymmath spm-opm
om import -content olymmath -anchor -comp pt-opm
om anchor -content olymmath -missing spm-opm
```

The archive page of olimpiadas.spm.pt links the papers of each olympiad, in a folder named by its school year, such as "XXXI OPM(2012-13)". The year of an edition is the year in which the school year ends. The positions of `pt-opm` are the problems of the final of category B, which has two days of three problems. The names of the files of that final change over the years, and `om papers` knows each form: XXXI_f1b.pdf, XXVIII_f2bp.pdf, XXIV_fdia1b.pdf, XX_fb2.pdf and catBfinaldia1.pdf for one day, and XXXIX_fb.pdf and XXVI_f12bps.pdf for both days. The papers number the problems with no word, and the second day starts at 4, so `om anchor` puts the word Problem before the numbers that follow each other. The pages state no licence, so the content only links to the papers. The site has no robots.txt.

The Czech Mathematical Olympiad works the same way, with the source `mo-cz`:

```sh
om papers -content olymmath mo-cz
om fetch -content olymmath mo-cz
om import -content olymmath -anchor -comp cz-sk-mo
om anchor -content olymmath -missing mo-cz
```

The site matematickaolympiada.cz has a page for each olympiad from the 43rd, which ended in 1994, but none for the 45th and the 46th. The nth olympiad ends in the year 1951 + n. The positions of `cz-sk-mo` are the problems of the final round of category A. Its file is a57iii.pdf, the third round, up to the 72nd olympiad, and a73iv.pdf, the fourth round, from the 73rd. The file is in Czech and has the solutions too. A solution or the header of a page hides the text up to the line with the next number, and a number with more than two spaces before it is part of a list. The pages state no licence, so the content only links to the papers. The robots.txt allows all paths.

The ELMO works the same way, with the source `elmo-official`:

```sh
om papers -content olymmath elmo-official
om fetch -content olymmath elmo-official
om import -content olymmath -anchor -comp elmo
om anchor -content olymmath -missing elmo-official
```

The problems page of the ELMO site, elmo.evanchen.cc, links the paper of each year from 2010, at web.evanchen.cc/exams/ELMO-YYYY.pdf. The paper of 2015 is only an attachment on AoPS, so `om papers` leaves it out. The files of the shortlist, the solutions and the ELSMO have other names. The papers number the problems 1 to 6, with or without the word Problem. The pages state no licence, so the content only links to the papers. The robots.txt of both hosts has no rule.

The Serbian Mathematical Olympiad works the same way, with the source `dms-rs`:

```
om papers -content olymmath dms-rs
om fetch -content olymmath dms-rs
om import -content olymmath -anchor -comp rs-smo
om anchor -content olymmath -missing dms-rs
```

The secondary school page of the Mathematical Society of Serbia, dms.rs/matematika-srednje-skole/, links the paper of the Serbian MO for 2008 to 2014, 2016 to 2019 and 2022 to 2024. The file names vary: zadaci_smo_2012.pdf, zadaci-smo-2014.pdf, 2017_smo.pdf, SMO2019.pdf, SMO_2022_ZADACI.pdf and SMO_2023_SVE.pdf. The files of the solutions (resenja) and the results (rezultati) are left out. The papers are in Serbian. The papers of 2008 to 2011 and 2013 number the problems of each day 1 to 3, so only the first day splits by number, and a row that AoPS places at a first-day position can be a second-day problem. Read each link against the paper. The pages state no licence, so the content only links to the papers. The robots.txt only disallows /wp-admin/.

With `-anchor`, `om import` also checks the position that a row gets from the number of its AoPS topic. When the edition has an English official paper, and the paper has another problem at that position, the row is skipped. Some AoPS collections hold the wrong contest: "2017 Australian MO" holds the problems of the Austrian olympiad.

### Link the human solutions

```sh
om solutions -content olymmath kedlaya-putnam
```

The Putnam archive also has the solutions of each year from 1995, by Kiran Kedlaya and his co-authors. Kedlaya asks for links, not copies, so these solutions are in zone Z3. The `om solutions` command gets each solution file into the raw store, finds the problems that it solves and writes one solution file for each problem, with the authors and a link to the PDF. A solution file in zone Z3 has no text. The problem page links to the solution. The command does not link a problem two times, so you can run it again each year.

```sh
om solutions -content olymmath imo-official
```

The ISL booklets on imo-official.org have the official solutions, from 2006. The solution part of a booklet repeats each problem and then gives its solution. For `imo-official`, the command finds the page where each solution starts and writes a link to that page of the booklet. These solutions are in zone Z2, but olymmath copies no excerpt.

```sh
om solutions -content olymmath mathse
```

Mathematics Stack Exchange is CC BY-SA 4.0, so olymmath can copy an answer when it names the author and links to the post. For `mathse`, the command searches the API for questions that name IMO, ISL, Putnam, USAMO or USAJMO and a year in the title. It keeps a question only when the problem of that edition occurs in the question. Then it takes at most 3 answers of each question: the accepted answer and the answers with a score of 3 or more. An answer must have at least 60 words. The command does not take an answer that speaks to the user ("you", "your"), a hint, a comment, or a copy of a text from another site. These solutions have the text of the answer and are not marked complete, because nobody has checked them yet. The API without a key gives 300 requests a day, so one run reads at most 10 pages for each search word.

### Set the topics

```sh
om topics -content olymmath
```

A shortlist gives the topic of a problem by the letter of its position: A3 of the IMO Shortlist is algebra, C is combinatorics, G is geometry and N is number theory. A competition with `topic_positions: true` in the register has such positions. The `om topics` command sets the topic of each problem that has no topic and appears at such a position. It does not change a topic that is there already. Use `-dry` to see the changes first.

## Layout

| Path | What it has |
|---|---|
| `cmd/om` | the main package of the binary |
| `cli` | the command tree, the flags and the exit codes |
| `content` | the loader of the registers, the editions and the problem files |
| `katex` | KaTeX in Go, for math on the server |
| `site` | the site generator, the templates and the style sheets |
| `spec` | the specification of the project |

## Contribute

Read [CONTRIBUTING.md](CONTRIBUTING.md). Report a security problem as [SECURITY.md](SECURITY.md) tells you.

## Licence

The code is under the Apache License 2.0. KaTeX is under the MIT licence. Its licence is in `katex/assets/LICENSE`.
