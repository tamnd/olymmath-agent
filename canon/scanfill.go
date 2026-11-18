package canon

import (
	"fmt"
	"math"

	"github.com/tamnd/olymmath-agent/content"
)

// AddScan gives the addition of one statement that a model read from the page of a scan, for an existing problem that has only links: the text of c at the page of s, with the extraction and the confidence of s (spec 05.7). Apply adds it to the file and Fill gives it the text. The statement is new, so the links of the problem stay as they are.
func AddScan(p *content.Problem, c *Candidate, s Scan) (Additions, error) {
	if s.Extraction == "" {
		return Additions{}, fmt.Errorf("a scanned text needs its extraction")
	}
	st := statementOf(fmt.Sprintf("S%d", len(p.Statements)+1), c)
	st.Page = s.Page
	st.Extraction = s.Extraction
	st.Confidence = math.Round(s.Confidence*100) / 100
	return Additions{Statements: []outStatement{st}}, nil
}
