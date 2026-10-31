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

The `om anchor` command gets the official English IMO papers into the raw store and reads their text with `pdftotext`. You must install Poppler to get `pdftotext`. The command compares each problem of a paper with the canonical text of the same problem in the content. When the words agree, the command adds a link to the paper and its page. When the words do not agree, the command adds a review item of kind `official` to the review queue. The command does not change the words of a canonical text. Use `-from` and `-to` to select the years.

The text layer of a PDF loses the layout of the formulas, so the text of a paper is never a statement. But a dataset row can be a faithful transcription of the paper. The command traces a transcription to the paper when two tests pass. First, the words of the two texts agree in their order: twice the longest common sequence of words, over the words of both texts, is 0.9 or more. Second, the two texts have the same set of numbers. The old papers set the point O so that the text layer gives 0, so a 0 that only the paper has does not count. A traced statement gets `traced: S3`, where S3 is the link to the paper, and it becomes the canonical statement. It has the class of the paper, H1 (spec 06.2). The site shows it as a transcription checked against the official paper.

Some datasets give the IMO problems of a year with no problem number. The command `om import -anchor -comp imo` compares each such text with the official papers in the raw store and gives it the position of the problem whose words agree best. The best signal must be 0.3 or more, and it must be at least 0.15 above the second best signal. Otherwise the row has no position, and the import skips it.

A text in zone Z3 is a link only, so the import does not take it. For example, the NuminaMath rows from the notes files have the texts of Evan Chen, and the register gives these texts the zone Z3. With `-anchor`, the import uses these texts only to find the position of a row with no number in other competitions, such as an AoPS row of a USAMO problem. The same thresholds apply. Two copies of one text do not compete with each other.

## Read a curated archive

Some curated sources give their problems as TeX files. For such a source, `om seed` gets each file into the raw store, converts each problem to Markdown with math and writes one seed row for each problem. The math does not change. A problem with TeX that has no Markdown form, for example a figure, gets no row. The command prints the reason for each such problem. The first archive is the Putnam archive of Kiran Kedlaya, from 1985:

```sh
om seed -content olymmath kedlaya-putnam
om import -content olymmath -comp putnam -promote
```

An import never changes the canonical text of an existing problem, unless you give `-promote`. With `-promote`, a candidate of class H1 or H2 becomes the canonical statement when it ranks above the canonical statement, its text agrees with the canonical text (the signal T is 0.5 or more) and the answers do not disagree. The old text stays in the file as another statement. Each other better candidate goes to the review queue.

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
