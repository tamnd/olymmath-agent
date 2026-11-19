package official

import (
	"reflect"
	"testing"
)

func TestChiuchangPapers(t *testing.T) {
	page := `<a href="https://chiuchang.org/imc/wp-content/uploads/sites/2/2024/08/InIMC-2024_Keystage-3_Individual.x17381.pdf">Individual</a>
<a href="https://chiuchang.org/imc/wp-content/uploads/sites/2/2024/08/2024-IWYMIC-answer.x17381.pdf">Answer</a>
<a href="https://chiuchang.org/imc/wp-content/uploads/sites/2/2024/08/InIMC-2024_Keystage-2_Team.x17381.pdf">Team</a>
<a href="/wp-content/uploads/sites/2/2018/01/2017InIMC_Keystage_III_Team_ENG.x17381.pdf">Team</a>
<a href="/wp-content/uploads/sites/2/2018/02/2005-PEMIC-Individual.x17381.pdf">Individual</a>
<a href="/wp-content/uploads/sites/2/2018/02/2004-IWYMIC-Team.x17381.pdf">Team</a>
<a href="/wp-content/uploads/sites/2/2018/02/2008-TEMIC-Team.x17381.pdf">Team</a>
<a href="https://chiuchang.org/imc/wp-content/uploads/sites/2/2018/02/2008-TEMIC-Team.x17381.pdf">Team</a>`
	u := "https://chiuchang.org/imc/wp-content/uploads/sites/2/"
	want := []National{
		{Competition: "iwymic", Year: 2024, Paper: "section-a", URL: u + "2024/08/InIMC-2024_Keystage-3_Individual.x17381.pdf"},
		{Competition: "iwymic", Year: 2024, Paper: "section-b", URL: u + "2024/08/InIMC-2024_Keystage-3_Individual.x17381.pdf"},
		{Competition: "emic", Year: 2024, Paper: "team", URL: u + "2024/08/InIMC-2024_Keystage-2_Team.x17381.pdf"},
		{Competition: "iwymic", Year: 2017, Paper: "team", URL: u + "2018/01/2017InIMC_Keystage_III_Team_ENG.x17381.pdf"},
		{Competition: "emic", Year: 2005, Paper: "individual", URL: u + "2018/02/2005-PEMIC-Individual.x17381.pdf"},
		{Competition: "emic", Year: 2008, Paper: "team", URL: u + "2018/02/2008-TEMIC-Team.x17381.pdf"},
	}
	if got := ChiuchangPapers(page); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	if n := len(ChiuchangListings); n != 27 {
		t.Errorf("%d listings, want 27", n)
	}
}
