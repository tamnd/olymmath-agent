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

	labels := []Page{{N: 1, Text: "# Day II\n**IMO TST 4.** Let $n$ be a positive integer here.\n\n**IMO TST 5.** Let $m$ be a positive integer there."}}
	if got := Split(labels); len(got) != 2 || got[0].Number != 4 || got[1].Text != "Let $m$ be a positive integer there." {
		t.Errorf("labels: %+v", got)
	}

	more := []Page{{N: 1, Text: "Problem 1 Does there exist a sequence of integers?\nProblem T–2\nLet $a, b$ be positive.\n№3. Positive integer $d$ is not a square.\n№1. Алфавит состоит из $n$ букв."}}
	if got := Split(more); len(got) != 3 || got[0].Text != "Does there exist a sequence of integers?" || got[1].Number != 2 || got[1].Text != "Let $a, b$ be positive." || got[2].Number != 3 {
		t.Errorf("words with a space, a letter or №: %+v", got)
	}

	dashes := []Page{{N: 1, Text: "# Fase nacional 2008\n**1.-** Halla dos enteros positivos.\n# SOLUCIÓN:\nLa suma es par.\n**2.-** Prueba que $a<b$.\n**1.-** No es un problema."}}
	if got := Split(dashes); len(got) != 2 || got[0].Text != "Halla dos enteros positivos." || got[1].Number != 2 || got[1].Text != "Prueba que $a<b$." {
		t.Errorf("bold numbers with a dash: %+v", got)
	}
	if got := Split([]Page{{N: 1, Text: "1.- Halla $a$ y $b$.\n2.- Prueba que $a<b$."}}); len(got) != 2 || got[1].Text != "Prueba que $a<b$." {
		t.Errorf("bare numbers with a dash: %+v", got)
	}

	letters := []Page{{N: 2, Text: "# 2022 PAPER A: PROBLEMS\n**A1.** Let $ABCD$ be a unit square here.\n\nA2. Let $n$ be a positive integer there.\n**A3**. Show that $f$ has two roots."}}
	if got := Split(letters); len(got) != 3 || got[0].Number != 1 || got[0].Text != "Let $ABCD$ be a unit square here." || got[1].Text != "Let $n$ be a positive integer there." || got[2].Number != 3 {
		t.Errorf("letters: %+v", got)
	}
	if got := Split([]Page{{N: 1, Text: "1. Let $A_1$ be a point.\nA1. is a line of the problem.\n2. Show it."}}); len(got) != 2 || !strings.Contains(got[0].Text, "A1. is a line") {
		t.Errorf("letters after bare numbers: %+v", got)
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

func TestReadImages(t *testing.T) {
	dir := t.TempDir()
	var files []string
	for i, body := range []string{"crop 1", "crop 2"} {
		f := fmt.Sprintf("%s/row-%d.png", dir, i+1)
		if err := os.WriteFile(f, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	calls := 0
	ask := lane.AskerFunc(func(_ context.Context, q lane.Question) (lane.Answer, error) {
		calls++
		if q.Prompt != CropPrompt {
			t.Errorf("the prompt of a crop is not CropPrompt")
		}
		b, err := os.ReadFile(q.Images[0])
		if err != nil {
			return lane.Answer{}, err
		}
		return lane.Answer{Text: fmt.Sprintf("**%s.** Prove that $x \\ge 0$ for each real $x$.", strings.TrimPrefix(string(b), "crop ")), Lane: "claude", Version: "2.1"}, nil
	})
	cache := &Cache{Dir: t.TempDir()}
	r := &Reader{Asker: ask, Cache: cache}
	pages, err := r.ReadImages(context.Background(), files)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 || calls != 2 || pages[1].N != 2 || pages[1].File != files[1] || pages[0].Key != KeyFor(CropPromptVersion, []byte("crop 1")) {
		t.Fatalf("%d calls, pages %+v", calls, pages)
	}
	if probs := Split(pages); len(probs) != 2 || probs[1].Number != 2 || probs[1].Page != 2 {
		t.Errorf("split = %+v", probs)
	}
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("ReadImages deleted %s", f)
		}
	}
	// A crop and a page with the same bytes have two keys, so the cache never gives the answer of one prompt to the other.
	if KeyFor(CropPromptVersion, []byte("crop 1")) == Key([]byte("crop 1")) {
		t.Error("a crop and a page share a key")
	}
	if again, err := r.ReadImages(context.Background(), files); err != nil || calls != 2 || !again[0].Cached {
		t.Errorf("the second run: %d calls, %v", calls, err)
	}
	stop := &Reader{Asker: lane.AskerFunc(func(context.Context, lane.Question) (lane.Answer, error) {
		return lane.Answer{}, fmt.Errorf("no lane: %w", lane.ErrExhausted)
	}), Cache: &Cache{Dir: t.TempDir()}}
	if got, err := stop.ReadImages(context.Background(), files); !errors.Is(err, lane.ErrExhausted) || len(got) != 0 {
		t.Errorf("an exhausted chain: %d pages, %v", len(got), err)
	}
}

func TestCleanFrontMatter(t *testing.T) {
	// The block of empty keys that a server writes above the page, and the block that olmOCR writes.
	for _, in := range []string{
		"---\nrunning_head: \nfolio: \n---\n\n**1.** Prove $x$.\n",
		"---\nprimary_language: pl\nis_rotation_valid: True\nrotation_correction: 0\nis_table: False\nis_diagram: False\n---\n**1.** Prove $x$.",
	} {
		if got, flags := Clean(in); got != "**1.** Prove $x$." || len(flags) != 0 {
			t.Errorf("Clean(%q) = %q %v", in, got, flags)
		}
	}
	// A rule and a line with a capital letter are a part of the page.
	in := "---\nProblem 1: Prove $x$.\n---"
	if got, _ := Clean(in); got != in {
		t.Errorf("Clean took a part of the page: %q", got)
	}
	got, _ := Clean("**2.** See the figure.\n\n![A triangle ABC](page_10_20_300_200.png)")
	if got != "**2.** See the figure.\n\n[figure]" || !slices.Contains(Check(got), FlagFigure) {
		t.Errorf("image: %q", got)
	}
}

func TestReaderVariants(t *testing.T) {
	calls := 0
	model := "glm"
	ask := lane.AskerFunc(func(context.Context, lane.Question) (lane.Answer, error) {
		calls++
		return lane.Answer{Text: "**1.** Prove that $x \\ge 0$ for each real $x$.", Lane: "local", Model: model}, nil
	})
	cache := &Cache{Dir: t.TempDir()}
	// An answer of another lane is in the cache under the shared key. The local lane does not take it.
	img := []byte("image 1 200")
	if err := cache.Put(Key(img), Entry{Text: "**1.** codex", Lane: "codex", Prompt: PromptVersion}); err != nil {
		t.Fatal(err)
	}
	r := &Reader{Asker: ask, Cache: cache, PDF: pdf{}, Work: t.TempDir(), DPI: []int{200}, Variants: []string{"local/glm", "local/olm"}}
	pages, err := r.Read(t.Context(), "a.pdf", []int{1})
	if err != nil || calls != 1 || pages[0].Cached || pages[0].Key != KeyFor(PromptVersion+" local/glm", img) {
		t.Fatalf("%d calls, %+v, %v", calls, pages, err)
	}
	if got := pages[0].Extraction(); got != "vision/local (glm)" {
		t.Errorf("extraction %q", got)
	}
	// A page that the second model read is in the cache for the lane too.
	img2 := []byte("image 2 200")
	if err := cache.Put(KeyFor(PromptVersion+" local/olm", img2), Entry{Text: "**2.** olm text", Lane: "local", Model: "olm", Prompt: PromptVersion}); err != nil {
		t.Fatal(err)
	}
	pages, err = r.Read(t.Context(), "a.pdf", []int{1, 2})
	if err != nil || calls != 1 || !pages[0].Cached || !pages[1].Cached || pages[1].Model != "olm" {
		t.Fatalf("%d calls, %+v, %v", calls, pages, err)
	}
	// The shared key still holds the answer of codex.
	if e, ok, _ := cache.Get(Key(img)); !ok || e.Lane != "codex" {
		t.Errorf("shared entry %+v", e)
	}
}

func TestReaderRefusedPage(t *testing.T) {
	calls := 0
	ask := lane.AskerFunc(func(_ context.Context, q lane.Question) (lane.Answer, error) {
		calls++
		b, _ := os.ReadFile(q.Images[0])
		if strings.HasPrefix(string(b), "image 2 ") {
			return lane.Answer{}, fmt.Errorf("local: %w: the answer reached the token limit", lane.ErrRefused)
		}
		return lane.Answer{Text: "**1.** Prove that $x \\ge 0$ for each real $x$.", Lane: "local", Model: "m"}, nil
	})
	r := &Reader{Asker: ask, Cache: &Cache{Dir: t.TempDir()}, PDF: pdf{}, Work: t.TempDir()}
	pages, err := r.Read(t.Context(), "a.pdf", []int{1, 2, 3})
	if err != nil || len(pages) != 3 || calls != 4 {
		t.Fatalf("%d calls, %d pages, %v", calls, len(pages), err)
	}
	if pages[1].N != 2 || pages[1].Text != "" || !slices.Equal(pages[1].Flags, []string{FlagRefusal}) {
		t.Errorf("refused page %+v", pages[1])
	}
}
