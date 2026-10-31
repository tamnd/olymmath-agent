package archive

import (
	"strings"
	"testing"
)

const putnamTeX = `\begin{document}
\begin{itemize}
\item[A--1]
Find the number of triples % a comment
that have the property that
\begin{enumerate}
\item[(i)] $A_1 \cup A_2 = \{1,2\}$, and
\item[(ii)] $A_1 \cap A_2 = \emptyset$.
\end{enumerate}

\item[A2]
A \emph{base $10$ over-expansion} of $N$ is {\em
good} if
\begin{align*}
x &= y \\
y &= z.
\end{align*}
Is it ` + "``good''?" + `

\item[B1]
\begin{tikzpicture}
\draw (0,0) -- (1,1);
\end{tikzpicture}
\end{itemize}
\end{document}
`

func TestParsePutnam(t *testing.T) {
	ps := ParsePutnam(putnamTeX)
	if len(ps) != 3 {
		t.Fatalf("%d problems", len(ps))
	}
	want := "Find the number of triples that have the property that\n\n(i) $A_1 \\cup A_2 = \\{1,2\\}$, and\n\n(ii) $A_1 \\cap A_2 = \\emptyset$."
	if ps[0].Number != "A1" || ps[0].Text != want {
		t.Errorf("A1 %q %q", ps[0].Number, ps[0].Text)
	}
	if !strings.Contains(ps[1].Text, "A *base $10$ over-expansion* of $N$ is *good* if") || !strings.Contains(ps[1].Text, `$$\begin{aligned}x &= y \\`) || !strings.HasSuffix(ps[1].Text, `Is it "good"?`) {
		t.Errorf("A2 %q", ps[1].Text)
	}
	if ps[2].Err == nil || ps[2].Text != "" {
		t.Errorf("B1 %+v", ps[2])
	}
}

func TestMarkdown(t *testing.T) {
	if _, err := Markdown(`See \cite{x}.`); err == nil {
		t.Error("an unknown macro was accepted")
	}
	if got, _ := Markdown("Let $x$ be real.\n\\begin{itemize}\n\\item one\n\\item two\n\\end{itemize}"); got != "Let $x$ be real.\n\n- one\n\n- two" {
		t.Errorf("itemize %q", got)
	}
	if got, _ := Markdown("\\begin{enumerate}\n\\item one\n\\item two\n\\end{enumerate}"); got != "1. one\n\n2. two" {
		t.Errorf("enumerate %q", got)
	}
}

func TestPutnam(t *testing.T) {
	ds := Putnam(1980, 1986)
	if len(ds) != 2 || ds[0].Year != 1985 || ds[1].URL != "https://kskedlaya.org/putnam-archive/1986.tex" {
		t.Errorf("%+v", ds)
	}
}

func TestExpand(t *testing.T) {
	tex := "\\begin{document}\n\\newcommand{\\RR}{\\mathbb{R}}\n$f: \\RR \\to \\RR^2$ and $x \\mbox{ for all $x$}$ \\RRR"
	if got := Expand(tex); got != "\\begin{document}\n$f: \\mathbb{R} \\to \\mathbb{R}^2$ and $x \\text{ for all $x$}$ \\RRR" {
		t.Errorf("%q", got)
	}
	if got, err := Markdown(`the set $\{k\colon \text{the $k$th turn}\}$ is odd`); err != nil || got != `the set $\{k\colon \text{the }{k}\text{th turn}\}$ is odd` {
		t.Errorf("%q %v", got, err)
	}
}

func TestSolutionFiles(t *testing.T) {
	tex := "\\author{Manjul Bhargava, Kiran Kedlaya, and\nLenny Ng}\n\\begin{itemize}\n\\item[A1] text\n\\begin{itemize}\\item[(a)] x\\end{itemize}\n\\item[A--2] more\n\\item[A1] again\n\\end{itemize}"
	if got := Labels(tex); len(got) != 2 || got[0] != "A1" || got[1] != "A2" {
		t.Errorf("labels %q", got)
	}
	if got := Authors(tex); got != "Manjul Bhargava, Kiran Kedlaya and Lenny Ng" {
		t.Errorf("authors %q", got)
	}
	if ds := PutnamSolutions(1990, 1995); len(ds) != 1 || ds[0].URL != "https://kskedlaya.org/putnam-archive/1995s.tex" {
		t.Errorf("%+v", ds)
	}
}

func TestTSTPapers(t *testing.T) {
	listing := `[{"name":"README.md","html_url":"r"},{"name":"TSTST2025Problems.pdf","html_url":"a"},{"name":"TSTST2025Solutions.pdf","html_url":"b"},{"name":"IMOTST2026Problems.pdf","html_url":"c"},{"name":"EGMOTST2026Problems.pdf","html_url":"d"}]`
	ps, err := TSTPapers([]byte(listing))
	if err != nil || len(ps) != 3 {
		t.Fatalf("%+v %v", ps, err)
	}
	if ps[0] != (TSTPaper{"tstst", 2025, "a"}) || ps[1].Competition != "us-egmo-tst" || ps[2] != (TSTPaper{"us-tst", 2026, "c"}) {
		t.Errorf("%+v", ps)
	}
}
