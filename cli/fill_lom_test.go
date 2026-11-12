package cli

import (
	"context"
	"testing"
)

func TestFillLanesLomSharygin(t *testing.T) {
	for comp, src := range map[string]string{"ru-lomonosov": "turlom", "sharygin": "geometry-ru"} {
		l, ok := ruLanes[comp]
		if !ok || l.source != src || l.final || !l.holds("2013 год", "2013") {
			t.Errorf("lane %s = %+v, %v", comp, l, ok)
		}
	}
	// A year with no official paper gives no problems and reads nothing.
	if ps, err := lomRead(context.Background(), nil, "turlom", "2016"); err != nil || ps != nil {
		t.Errorf("lomRead 2016 = %v, %v", ps, err)
	}
	if ps, err := sharyginRead(context.Background(), nil, "geometry-ru", "2014"); err != nil || ps != nil {
		t.Errorf("sharyginRead 2014 = %v, %v", ps, err)
	}
}
