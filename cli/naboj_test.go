package cli

import (
	"reflect"
	"testing"

	"github.com/tamnd/olymmath-agent/official"
)

func TestNabojPlace(t *testing.T) {
	ats, names := nabojPlace(hrPaper{kind: "naboj-junior", year: 2023}, []official.HrProblem{{N: 5}, {N: 0}})
	if want := []string{"naboj-junior/2023/p5", ""}; !reflect.DeepEqual(ats, want) {
		t.Errorf("nabojPlace places at %q, want %q", ats, want)
	}
	if names[0] != "Náboj Junior 2023, problems and solutions" {
		t.Errorf("nabojPlace names the paper %q", names[0])
	}
}
