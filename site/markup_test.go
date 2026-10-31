package site

import (
	"strings"
	"testing"
)

func TestSplitMath(t *testing.T) {
	text, spans := splitMath(`Let $a$ be $\$5$ and $$x^2$$ and \(y\) and \[z\] cost \$3.`)
	if len(spans) != 5 {
		t.Fatalf("spans: %+v", spans)
	}
	if spans[1].tex != `\$5` || !spans[2].display || spans[3].display || !spans[4].display {
		t.Fatalf("spans: %+v", spans)
	}
	if !strings.HasSuffix(text, `cost \$3.`) {
		t.Fatalf("text: %q", text)
	}
}

func TestSplitMathUnclosed(t *testing.T) {
	text, spans := splitMath("It costs $5 only.")
	if len(spans) != 0 || text != "It costs $5 only." {
		t.Fatalf("%q %+v", text, spans)
	}
}

func TestMarkupHTML(t *testing.T) {
	m, err := NewMarkup(nil)
	if err != nil {
		t.Fatal(err)
	}
	out, errs := m.HTML("Find all $n$ such that\n$$n^2 = 4.$$\n\n- one\n- two")
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	for _, want := range []string{`class="katex"`, `katex-display`, "<li>one</li>", "<p>Find all"} {
		if !strings.Contains(out, want) {
			t.Errorf("no %q in %s", want, out)
		}
	}
	_, errs = m.HTML(`Bad $\frac{1}$.`)
	if len(errs) != 1 {
		t.Fatalf("want one error, got %v", errs)
	}
}

func TestMarkupEscapesHTML(t *testing.T) {
	m, err := NewMarkup(nil)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := m.HTML("<script>alert(1)</script> and $a<b$")
	if strings.Contains(out, "<script>") {
		t.Fatalf("raw HTML is in the output: %s", out)
	}
}

func TestPlain(t *testing.T) {
	got := Plain("Find **all** $n$\nsuch that $n^2=4$.")
	if got != "Find all n such that n^2=4." {
		t.Fatalf("%q", got)
	}
}
