package content

import (
	"strings"
	"testing"
)

func TestPositionTopic(t *testing.T) {
	isl := &Competition{Code: "isl", TopicPositions: true}
	putnam := &Competition{Code: "putnam"}
	for _, c := range []struct {
		comp *Competition
		pos  string
		want string
	}{{isl, "a3", "A"}, {isl, "g7", "G"}, {isl, "p1", ""}, {putnam, "a1", ""}, {nil, "a1", ""}, {isl, "", ""}} {
		if got := PositionTopic(c.comp, c.pos); got != c.want {
			t.Errorf("%v %q: %q", c.comp, c.pos, got)
		}
	}
}

func TestSetTopics(t *testing.T) {
	in := "---\ntag: 73NDBAMJC4\nkind: problem\nappearances:\n  - at: isl/2009/c8\n    primary: true\ntype: proof\nstatus:\n  statement: sourced\n---\nText.\n"
	out, err := SetTopics([]byte(in), []string{"C"})
	if err != nil {
		t.Fatal(err)
	}
	want := "type: proof\ntopics: [C]\n"
	if got := string(out); !strings.Contains(got, want) || !strings.HasSuffix(got, "---\nText.\n") {
		t.Errorf("%s", got)
	}
}
