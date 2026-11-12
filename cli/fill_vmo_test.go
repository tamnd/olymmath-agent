package cli

import (
	"context"
	"testing"
)

func TestFillLaneVmo(t *testing.T) {
	l, ok := ruLanes["ru-allrussian"]
	if !ok || l.source != "mccme-vmo" || !l.final || l.node != prAllRussian {
		t.Fatalf("lane = %+v, %v", l, ok)
	}
	// A year node, as 2004-2005, runs over a school year, and the final round is in its spring.
	for name, want := range map[string]bool{"2004-2005": true, "2005-2006": false, "2003-2004": false, "2005": false, "весна": false} {
		if got := l.holds(name, "2005"); got != want {
			t.Errorf("holds(%q, 2005) = %v, want %v", name, got, want)
		}
	}
	// A year with no paper in the archive gives no problems and reads nothing.
	if ps, err := vmoRead(context.Background(), nil, "mccme-vmo", "2000"); err != nil || ps != nil {
		t.Errorf("vmoRead 2000 = %v, %v", ps, err)
	}
	if _, err := vmoRead(context.Background(), nil, "mccme-vmo", "x"); err == nil {
		t.Error("vmoRead of a bad year gives no error")
	}
}
