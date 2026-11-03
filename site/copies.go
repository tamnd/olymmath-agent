package site

import (
	"encoding/json"
	"fmt"
	"strings"
)

// problemCopy is the JSON copy of a problem page. It has the same facts and texts as the HTML page.
type problemCopy struct {
	Tag         string        `json:"tag"`
	Locator     string        `json:"locator,omitempty"`
	URL         string        `json:"url"`
	Title       string        `json:"title"`
	Competition string        `json:"competition,omitempty"`
	Year        int           `json:"year,omitempty"`
	Position    string        `json:"position,omitempty"`
	Type        string        `json:"type,omitempty"`
	Topics      []string      `json:"topics,omitempty"`
	Band        int           `json:"band,omitempty"`
	Statement   string        `json:"statement"`
	Source      SourceRef     `json:"source"`
	Official    *SourceRef    `json:"official,omitempty"`
	Answer      string        `json:"answer,omitempty"`
	Solutions   []SolutionRef `json:"solutions"`
	Status      string        `json:"status,omitempty"`
}

// copies writes the machine copies of a problem page: the same path with .md and .json. The JSON copy has the fields of the API records of the problem.
func (b *builder) copies(v *problemView, p string) error {
	r, sol := b.records.Problem(v.Problem), b.records.Solutions(v.Problem)
	c := problemCopy{
		Tag: r.Tag, Locator: r.Locator, URL: r.URL, Title: r.Title,
		Competition: r.Competition, Year: r.Year, Position: r.Position, Type: r.Type,
		Topics: r.Topics, Band: r.Band, Statement: r.Statement,
		Source: r.Source, Official: r.Official, Answer: sol.Answer, Status: r.Status,
		Solutions: sol.Solutions,
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	file := strings.TrimPrefix(p, "/")
	if err := b.write(file+".json", append(data, '\n')); err != nil {
		return err
	}
	return b.write(file+".md", []byte(problemMarkdown(c)))
}

// problemMarkdown gives the Markdown copy of a problem page.
func problemMarkdown(c problemCopy) string {
	var m strings.Builder
	fmt.Fprintf(&m, "# %s\n\n", c.Title)
	var facts []string
	if c.Locator != "" {
		facts = append(facts, "Locator: "+c.Locator)
	}
	facts = append(facts, "Tag: "+c.Tag)
	if len(c.Topics) > 0 {
		names := make([]string, len(c.Topics))
		for i, t := range c.Topics {
			names[i] = topicName(t)
		}
		facts = append(facts, "Topics: "+strings.Join(names, ", "))
	}
	if c.Band > 0 {
		facts = append(facts, fmt.Sprintf("Difficulty band: %d of 10", c.Band))
	}
	fmt.Fprintf(&m, "%s\n\n%s\n\n", strings.Join(facts, " · "), strings.TrimSpace(c.Statement))
	fmt.Fprintf(&m, "Source: %s.\n", sourceLine(c.Source))
	if c.Official != nil {
		fmt.Fprintf(&m, "Official text: %s.\n", sourceLine(*c.Official))
	}
	if c.Answer != "" {
		fmt.Fprintf(&m, "\n## Answer\n\n%s\n", strings.TrimSpace(c.Answer))
	}
	m.WriteString("\n## Solutions\n")
	if len(c.Solutions) == 0 {
		m.WriteString("\nNo solution is on olymmath yet.\n")
	}
	for i, s := range c.Solutions {
		fmt.Fprintf(&m, "\n### Solution %d", i+1)
		if s.Author != "" {
			fmt.Fprintf(&m, " by %s", s.Author)
		}
		fmt.Fprintf(&m, " (%s)\n\n", s.Origin)
		if s.Text != "" {
			fmt.Fprintf(&m, "%s\n\n", strings.TrimSpace(s.Text))
		}
		switch {
		case s.URL != "" && s.Text == "":
			fmt.Fprintf(&m, "The source allows a link only: [%s](%s).\n", s.Source, s.URL)
		case s.URL != "":
			fmt.Fprintf(&m, "Source: [%s](%s), zone %s.\n", s.Source, s.URL, s.Zone)
		}
	}
	fmt.Fprintf(&m, "\nPage: %s\n", c.URL)
	return m.String()
}

func sourceLine(s SourceRef) string {
	line := s.Name
	if s.URL != "" {
		line = "[" + s.Name + "](" + s.URL + ")"
	}
	if s.Page > 0 {
		line += fmt.Sprintf(", page %d", s.Page)
	}
	if s.Origin != "" {
		line += ", origin " + s.Origin
	}
	if s.Zone != "" {
		line += ", zone " + s.Zone
	}
	return line
}
