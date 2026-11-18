package official

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

// NabojVersion names the reading of the HTML pages of Náboj in the extraction field of a statement.
const NabojVersion = "om-naboj/1"

// NabojYear is a competition of the archive of a Náboj site: its year and the path of its page of problems, or "" when the archive has no problems for it.
type NabojYear struct {
	Year     int
	Problems string
}

var (
	// nabojYearRE is the heading of a competition of the archive, as "Náboj Math 2026" or "Náboj Junior 2021", with the links that follow it.
	nabojYearRE = regexp.MustCompile(`<h2[^>]*>\s*Náboj [A-Za-z]+ (\d{4})\s*</h2>`)
	// nabojProblemsRE is the link to the page of the problems and solutions of a competition.
	nabojProblemsRE = regexp.MustCompile(`href="?(/[a-z]{2}/[a-z]{2}/archive/\d+/problems/?)"?[ >]`)
	// nabojArticleRE is the heading of a problem of a page of problems, as "Problem 12" or "Úloha 12".
	nabojArticleRE = regexp.MustCompile(`<article[^>]*>\s*<h2[^>]*>\s*[^<\d]*?(\d+)\s*</h2>`)
	// nabojPDFRE is the link of a page of problems to the paper PDF.
	nabojPDFRE  = regexp.MustCompile(`href="?(/uploads/competitions/[^" >]+\.pdf)`)
	nabojMathRE = regexp.MustCompile(`(?s)<math\b([^>]*)>(.*?)</math>`)
	nabojImgRE  = regexp.MustCompile(`(?i)<img\b`)
	// nabojTabularRE is a table of data; a table of the class equation-star only sets a formula apart.
	nabojTabularRE = regexp.MustCompile(`<table\b[^>]*class="?tabular`)
	nabojParRE     = regexp.MustCompile(`(?i)<(?:p|div|br|hr|li|tr|h[1-6])\b[^>]*>`)
	nabojItemRE    = regexp.MustCompile(`(?i)<li\b[^>]*>`)
	nabojTagRE     = regexp.MustCompile(`<[^>]*>`)
	nabojSpaceRE   = regexp.MustCompile(`[ \t\r\f\v]+`)
	nabojLinesRE   = regexp.MustCompile(`\s*\n\s*`)
	// nabojTitleRE is the title in bold that the problems of the junior competition have before their text, as "Na vlásku": a span of the class ec-lmbx or of the weight 700.
	nabojTitleRE = regexp.MustCompile(`^\s*<p[^>]*>\s*<span (?:class="?ec-lmbx|style="?font-weight:\s*(?:700|bold))[^>]*>[^<]*</span>\s*(?:<br[^>]*>)?`)
)

// NabojArchive reads the competitions of an archive page of a Náboj site.
func NabojArchive(page string) []NabojYear {
	var out []NabojYear
	ms := nabojYearRE.FindAllStringSubmatchIndex(page, -1)
	for i, m := range ms {
		y, _ := strconv.Atoi(page[m[2]:m[3]])
		end := len(page)
		if i+1 < len(ms) {
			end = ms[i+1][0]
		}
		ny := NabojYear{Year: y}
		if p := nabojProblemsRE.FindStringSubmatch(page[m[1]:end]); p != nil {
			ny.Problems = p[1]
		}
		out = append(out, ny)
	}
	return out
}

// NabojPDF gives the path of the paper PDF that a page of problems links, or "".
func NabojPDF(page string) string {
	if m := nabojPDFRE.FindStringSubmatch(page); m != nil {
		return m[1]
	}
	return ""
}

// NabojProblems reads the problems of a page of problems and solutions of a Náboj site: the text of each problem before its solution, as Markdown with TeX math. A problem with a picture has Figure set, and a formula whose MathML has no TeX here gives the flag encoding.
func NabojProblems(page string) []HrProblem {
	var out []HrProblem
	ms := nabojArticleRE.FindAllStringSubmatchIndex(page, -1)
	for i, m := range ms {
		end := len(page)
		if i+1 < len(ms) {
			end = ms[i+1][0]
		}
		body := page[m[1]:end]
		if j := strings.Index(body, "<details"); j >= 0 {
			body = body[:j]
		} else if j := strings.Index(body, "</article>"); j >= 0 {
			body = body[:j]
		}
		body = strings.TrimPrefix(strings.TrimSpace(body), `<div class="prose prose-archive">`)
		body = nabojTitleRE.ReplaceAllString(body, "")
		n, _ := strconv.Atoi(page[m[2]:m[3]])
		p := HrProblem{N: n}
		p.Code = strconv.Itoa(n)
		p.Text, p.Figure, p.Flags = nabojText(body)
		out = append(out, p)
	}
	return out
}

// nabojText gives the Markdown of the HTML of a problem, whether it has a picture, and the flags of its reading.
func nabojText(body string) (string, bool, []string) {
	var flags []string
	figure := nabojImgRE.MatchString(body)
	if nabojTabularRE.MatchString(body) {
		// The cells of a table would come apart in the Markdown.
		flags = append(flags, "layout")
	}
	var maths []string
	body = nabojMathRE.ReplaceAllStringFunc(body, func(s string) string {
		m := nabojMathRE.FindStringSubmatch(s)
		tex, ok := MathMLTeX(s)
		if !ok && !containsStr(flags, "encoding") {
			flags = append(flags, "encoding")
		}
		if strings.TrimSpace(tex) == "" {
			// An empty formula, as of a cell of a table, would give the Markdown a bare $$.
			return " "
		}
		if strings.Contains(m[1], "display=block") || strings.Contains(m[1], `display="block"`) {
			tex = "\n\n$$" + tex + "$$\n\n"
		} else {
			tex = "$" + tex + "$"
		}
		maths = append(maths, tex)
		return "\x00" + strconv.Itoa(len(maths)-1) + "\x00"
	})
	body = nabojItemRE.ReplaceAllString(body, "\n\n- ")
	body = nabojParRE.ReplaceAllStringFunc(body, func(s string) string {
		if strings.HasPrefix(strings.ToLower(s), "<br") {
			return "\n"
		}
		return "\n\n"
	})
	body = nabojTagRE.ReplaceAllString(body, "")
	body = html.UnescapeString(body)
	// A $ of the text would open a formula of the Markdown.
	body = strings.ReplaceAll(body, "$", `\$`)
	for i, t := range maths {
		body = strings.Replace(body, "\x00"+strconv.Itoa(i)+"\x00", t, 1)
	}
	var paras []string
	for _, para := range strings.Split(body, "\n\n") {
		para = nabojSpaceRE.ReplaceAllString(para, " ")
		para = strings.TrimSpace(nabojLinesRE.ReplaceAllString(para, "\n"))
		if para != "" {
			paras = append(paras, para)
		}
	}
	return strings.Join(paras, "\n\n"), figure, flags
}

func containsStr(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
