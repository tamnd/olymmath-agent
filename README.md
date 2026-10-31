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

Use `om help` to see all the commands.

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
