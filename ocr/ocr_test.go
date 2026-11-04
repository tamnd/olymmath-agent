package ocr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tamnd/olymmath-agent/lane"
)

func TestClean(t *testing.T) {
	got, flags := Clean("```markdown\n**1.** Prove $x$.\n```")
	if got != "**1.** Prove $x$." || !slices.Equal(flags, []string{FlagFence}) {
		t.Errorf("fence: %q %v", got, flags)
	}
	if _, flags := Clean("I'm sorry, I cannot read this image."); !slices.Contains(flags, FlagRefusal) {
		t.Errorf("refusal: %v", flags)
	}
}

func TestCheck(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`Prove $a^{2} \ge 0$ and $$\{x\}$$, a cost of \$5.`, nil},
		{`Prove $a \ge 0.`, []string{FlagDollars}},
		{`Prove $$a$ b$.`, []string{FlagDollars}},
		{`Prove $\frac{1}{2$.`, []string{FlagBraces}},
		{`Prove $x$ [?] holds.`, []string{FlagUnreadable}},
		{"Look.\n[figure]", []string{FlagFigure}},
	}
	for _, c := range cases {
		if got := Check(c.in); !slices.Equal(got, c.want) {
			t.Errorf("Check(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestRepair(t *testing.T) {
	if got := Repair("Let \\(x>0\\).  \n\n\n\n\\[x^2\\]"); got != "Let $x>0$.\n\n$$x^2$$" {
		t.Errorf("Repair = %q", got)
	}
	if Confidence(nil) < Threshold || Confidence([]string{FlagDollars}) >= Threshold || Confidence([]string{FlagRefusal}) != 0 {
		t.Error("the confidence does not follow the flags")
	}
}

func TestSplit(t *testing.T) {
	pages := []Page{
		{N: 1, Text: "# XLVIII Olimpiada Matematyczna\n\n# Zadania na dzień 4 kwietnia\n\n**1.** Liczby $x_1$, $x_2$ spełniają warunki:\n\n1. pierwszy,\n2. drugi.\n\n**2.** Znaleźć wszystkie trójki liczb rzeczywistych.\n\n# Zadania na dzień 5 kwietnia\n\n**3.** Ciąg $a_n$ jest określony wzorami"},
		{N: 2, Text: "$$a_1 = 0.$$\n\n---\n\n# Rozwiązania\n\n**1.** Rozwiązanie zadania pierwszego, które jest długie."},
	}
	got := Split(pages)
	if len(got) != 3 {
		t.Fatalf("Split gave %d problems: %+v", len(got), got)
	}
	if got[0].Number != 1 || !strings.Contains(got[0].Text, "2. drugi.") || strings.HasPrefix(got[0].Text, "**") {
		t.Errorf("p1 = %+v", got[0])
	}
	if got[1].Text != "Znaleźć wszystkie trójki liczb rzeczywistych." {
		t.Errorf("p2 = %q", got[1].Text)
	}
	if got[2].Page != 1 || got[2].Text != "Ciąg $a_n$ jest określony wzorami\n$$a_1 = 0.$$" {
		t.Errorf("p3 across the pages = %+v", got[2])
	}

	words := []Page{{N: 1, Text: "Problem 1. Show that $n$ is even, which is short.\nSolution. Easy.\nProblem 2: Find all $f$ such that $f(x)=x$ holds."}}
	got = Split(words)
	if len(got) != 2 || got[0].Text != "Show that $n$ is even, which is short." || got[1].Number != 2 {
		t.Errorf("word markers: %+v", got)
	}

	days := []Page{{N: 1, Text: "# Day 1\n**1.** First problem of the first day.\n# Day 2\n**1.** First problem of the second day."}}
	if got := Sequence(days); len(got) != 2 || got[1].Number != 2 || !strings.Contains(got[1].Text, "second day") {
		t.Errorf("Sequence = %+v", got)
	}
	if got := Split([]Page{{N: 1, Text: "**1.** Hi."}}); !slices.Contains(got[0].Flags, FlagShort) {
		t.Errorf("a short text has no flag: %+v", got)
	}
}

// pdf is a Runner that gives a PDF of three pages. The image of page n at d dpi is "image n d".
type pdf struct{}

func (pdf) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	switch name {
	case "pdfinfo":
		return []byte("Pages: 3\nPage size: 595 x 842 pts (A4)\n"), nil
	case "pdftoppm":
		img := fmt.Sprintf("image %s %s", args[4], args[1])
		return nil, os.WriteFile(args[len(args)-1]+".png", []byte(img), 0o644)
	}
	return nil, errors.New("no " + name)
}

func TestReader(t *testing.T) {
	now = func() time.Time { return time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC) }
	calls := 0
	ask := lane.AskerFunc(func(_ context.Context, q lane.Question) (lane.Answer, error) {
		calls++
		b, err := os.ReadFile(q.Images[0])
		if err != nil {
			return lane.Answer{}, err
		}
		text := "**1.** Prove that $x \\ge 0$ for each real $x$."
		switch string(b) {
		case "image 2 200":
			text = "**2.** Prove that $x [?] 0$."
		case "image 2 300":
			text = "**2.** Prove that $x < 1$ for each $x$ here."
		case "image 3 200":
			text = Empty
		}
		return lane.Answer{Text: text, Lane: "claude", Model: "m", Version: "2.1"}, nil
	})
	work := t.TempDir()
	r := &Reader{Asker: ask, Cache: &Cache{Dir: t.TempDir()}, PDF: pdf{}, Work: work}
	pages, err := r.Read(context.Background(), "a.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 3 || calls != 4 {
		t.Fatalf("%d pages, %d calls", len(pages), calls)
	}
	if pages[1].DPI != 300 || len(pages[1].Flags) != 0 || pages[2].Text != "" {
		t.Errorf("the ladder: %+v", pages)
	}
	if got := pages[0].Extraction(); got != "vision/claude@2.1 (m)" {
		t.Errorf("extraction = %q", got)
	}
	if left, _ := os.ReadDir(work); len(left) != 0 {
		t.Errorf("the images are still in the work dir: %v", left)
	}
	// A second run reads the cache and calls no model.
	again, err := r.Read(context.Background(), "a.pdf", []int{1, 2})
	if err != nil || calls != 4 || !again[0].Cached || again[1].Text != pages[1].Text {
		t.Errorf("the second run: %d calls, %+v, %v", calls, again, err)
	}
}

func TestReaderStops(t *testing.T) {
	calls := 0
	ask := lane.AskerFunc(func(context.Context, lane.Question) (lane.Answer, error) {
		calls++
		return lane.Answer{}, fmt.Errorf("no lane: %w", lane.ErrExhausted)
	})
	r := &Reader{Asker: ask, Cache: &Cache{Dir: t.TempDir()}, PDF: pdf{}, Work: t.TempDir()}
	pages, err := r.Read(context.Background(), "a.pdf", []int{1, 2, 3})
	if !errors.Is(err, lane.ErrExhausted) || len(pages) != 0 || calls != 1 {
		t.Errorf("%d pages, %d calls, %v", len(pages), calls, err)
	}
}

func TestCache(t *testing.T) {
	c := &Cache{Dir: t.TempDir()}
	k := Key([]byte("png"))
	if _, ok, err := c.Get(k); ok || err != nil {
		t.Fatalf("an empty cache has %s: %v", k, err)
	}
	if err := c.Put(k, Entry{Text: "x", Lane: "claude", Prompt: PromptVersion}); err != nil {
		t.Fatal(err)
	}
	if e, ok, err := c.Get(k); !ok || err != nil || e.Text != "x" {
		t.Errorf("get = %+v %v %v", e, ok, err)
	}
	if Key([]byte("png")) == Key([]byte("png2")) {
		t.Error("two images have one key")
	}
}
