package official

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
)

// YearPage gives the page of imo-official.org that lists the papers of one year in all languages.
func YearPage(year int) string {
	return fmt.Sprintf("https://www.imo-official.org/problems/%d/", year)
}

// optionRE matches one paper in the language list of a year page: the option of the download widget with the path of the PDF, the language code and the name of the language.
var optionRE = regexp.MustCompile(`<option\s+value="(/assets/documents/problems/[0-9]{4}/[0-9]{4}_([a-z]{3})\.pdf)"\s+data-lang-code="[a-z]{3}"[^>]*>\s*([^<]+?)\s*</option>`)

// Languages gives the official papers of a year page, one for each language, in the order of the page. English comes first on the page.
func Languages(page string) []content.OfficialPaper {
	var out []content.OfficialPaper
	seen := map[string]bool{}
	for _, m := range optionRE.FindAllStringSubmatch(page, -1) {
		if seen[m[2]] {
			continue
		}
		seen[m[2]] = true
		out = append(out, content.OfficialPaper{Lang: strings.TrimSpace(html.UnescapeString(m[3])), Code: m[2], URL: "https://www.imo-official.org" + m[1]})
	}
	return out
}
