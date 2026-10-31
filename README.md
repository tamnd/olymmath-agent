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

The `om anchor` command gets the official English IMO papers into the raw store and reads their text with `pdftotext`. You must install Poppler to get `pdftotext`. The command compares each problem of a paper with the canonical text of the same problem in the content. When the words agree, the command adds a link to the paper and its page. When the words do not agree, the command adds a review item of kind `official` to the review queue. The command does not change a canonical text. Use `-from` and `-to` to select the years.

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
