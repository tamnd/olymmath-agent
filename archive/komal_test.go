package archive

import (
	"strings"
	"testing"
)

const komalPage = `<p id="C804" align="justify"><b>C.&nbsp;804.</b> The second term is <IMG SRC="/kep/a.gif" ALIGN="ABSCENTER" ALT="\frac{5}{2}">. Find this geometric progression.</P>
<p align="left">(5&nbsp;pont)</p>
<hr>
<p id="B3803" align="justify"><b>B.&nbsp;3803.</b> Are there three cubes whose sum of volumes is 10<SUP>20</SUP>+30?</P>
<p align="left">(3&nbsp;pont)</p>
<hr>
<p id="B3804" align="justify"><b>B.&nbsp;3804.</b> In the figure <IMG SRC="/kep/fig.gif">, find x.</P>
<hr>
<p id="A912" align="justify"><b>A.&nbsp;912.</b> Let \(\displaystyle  ABC \) be a triangle. Prove that these points lie on a line: </p>
<ul>
<li>the reflection of \(\displaystyle A\) over \(\displaystyle BC\);     </li>
<li>the midpoint of \(\displaystyle AA'\). </li>
</ul>
<p align="right">Proposed by: <i>Ábel Szakács</i>, Budapest</p>
<hr>
<p id="A412" align="justify"><b>A.&nbsp;412.</b> Let <I>t</I> be irrational and <I>f</I>(<I>x</I>,<I>y</I>)=<I>u</I><sub>1</sub>, as <i>the paper</i> says.</P>
<P ALIGN=JUSTIFY>(Proposed by <I>Tamás Keleti,</I> Budapest)</P>
<hr>
<p id="C1" align="justify"><b>C.&nbsp;1.</b> Subscribers can reach the text of the problem after signing in. The text will be public from September 01, 1984.]</P>
<hr>
<p id="P5100" align="justify"><b>P.&nbsp;5100.</b> A physics problem.</P>`

func TestParseKomal(t *testing.T) {
	ps := ParseKomal(komalPage)
	var nums []string
	for _, p := range ps {
		nums = append(nums, p.Number)
	}
	if got := strings.Join(nums, " "); got != "C804 B3803 B3804 A912 A412 C1" {
		t.Fatalf("problems %s", got)
	}
	want := []string{
		"The second term is $\\frac{5}{2}$. Find this geometric progression.",
		"Are there three cubes whose sum of volumes is $10^{20}$+30?",
		"",
		"Let $ABC$ be a triangle. Prove that these points lie on a line:\n- the reflection of $A$ over $BC$;\n- the midpoint of $AA'$.",
		"Let $t$ be irrational and $f$($x$,$y$)=$u_1$, as the paper says.",
	}
	for i, w := range want {
		if ps[i].Text != w {
			t.Errorf("%s:\n got %q\nwant %q", ps[i].Number, ps[i].Text, w)
		}
	}
	if ps[2].Err == nil {
		t.Error("B3804 has a figure and no error")
	}
	if ps[5].Err == nil || ps[5].Text != "" {
		t.Errorf("C1 is only for subscribers: %+v", ps[5])
	}
}

func TestKomal(t *testing.T) {
	docs := Komal(2005, 2005)
	if len(docs) != 9 || docs[0].Year != 2005 || docs[4].Year != 2006 || !strings.HasSuffix(docs[4].URL, "h=200601&t=200601") {
		t.Fatalf("docs %+v", docs)
	}
}
