package locator

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		kind Kind
		out  string
	}{
		{"imo", Competition, "imo"},
		{"ru-allrussian", Competition, "ru-allrussian"},
		{"imo/2024", Edition, "imo/2024"},
		{"usamo/2021a", Edition, "usamo/2021a"},
		{"kmo/2021-spring", Edition, "kmo/2021-spring"},
		{"imo/2024/p1", Appearance, "imo/2024/p1"},
		{"/isl/2024/n1/", Appearance, "isl/2024/n1"},
		{"amc12/2021/a/p25", Appearance, "amc12/2021/a/p25"},
		{"ru-allrussian/2019/g10-d2/p3", Appearance, "ru-allrussian/2019/g10-d2/p3"},
		{"putnam/1985/b6", Appearance, "putnam/1985/b6"},
	}
	for _, c := range cases {
		l, err := Parse(c.in)
		if err != nil {
			t.Errorf("Parse(%q): %v", c.in, err)
			continue
		}
		if l.Kind() != c.kind || l.String() != c.out {
			t.Errorf("Parse(%q) = %s %q, want %s %q", c.in, l.Kind(), l, c.kind, c.out)
		}
	}
	for _, in := range []string{"", "IMO", "imo/24", "imo/2024/P1", "imo/2024/p1/x/y", "imo--x", "1imo", "imo/2024/day one/p1"} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) gave no error", in)
		}
	}
}

func TestPath(t *testing.T) {
	l, _ := Parse("imo/2024/p1")
	if got := l.Path(); got != "problems/imo/2024/p1.md" {
		t.Errorf("Path = %q", got)
	}
	if got := l.EditionOf().String(); got != "imo/2024" {
		t.Errorf("EditionOf = %q", got)
	}
}

func TestParseURI(t *testing.T) {
	cases := map[string]string{
		"om://7K3QX2M9R2":                     "om://7K3QX2M9R2",
		"om://7K3QX-2M9R2":                    "om://7K3QX2M9R2",
		"om://imo/2024/p1":                    "om://imo/2024/p1",
		"om://imo":                            "om://imo",
		"om://src/aops-collections/post/1234": "om://src/aops-collections/post/1234",
	}
	for in, want := range cases {
		u, err := ParseURI(in)
		if err != nil {
			t.Errorf("ParseURI(%q): %v", in, err)
			continue
		}
		if u.String() != want {
			t.Errorf("ParseURI(%q) = %q, want %q", in, u, want)
		}
	}
	for _, in := range []string{"https://x", "om://7K3XQ2M9R2", "om://src/x", "om://imo/24"} {
		if _, err := ParseURI(in); err == nil {
			t.Errorf("ParseURI(%q) gave no error", in)
		}
	}
}
