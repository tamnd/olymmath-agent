package content

import (
	"fmt"
	"maps"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/tamnd/olymmath-agent/locator"
)

// Permission is one row of registers/permissions.yaml (spec 11.4): one rights holder that olymmath asks, or will ask, for leave to show its texts in zone Z1. The row has no zone. A grant moves a source to Z1 only through its own change of registers/sources.yaml, which names the row in the licence of the source as permission:<id>.
type Permission struct {
	ID     string `yaml:"id"`
	Holder string `yaml:"holder"`
	// Country is the ISO 3166-1 alpha-2 code of the holder, or international.
	Country string `yaml:"country"`
	// Competitions and Sources are the codes and source ids that the request covers. Each must be in its register.
	Competitions []string `yaml:"competitions,omitempty"`
	Sources      []string `yaml:"sources,omitempty"`
	// Languages are the ISO 639-1 codes of the languages of the letter. The English letter is always prepared; the others are translated for the row.
	Languages []string          `yaml:"languages,omitempty"`
	Contact   PermissionContact `yaml:"contact,omitempty"`
	// Scope is what olymmath asks for. Granted is what the holder allowed, which can be less.
	Scope    PermissionScope  `yaml:"scope"`
	Granted  *PermissionScope `yaml:"granted,omitempty"`
	Priority string           `yaml:"priority"`
	State    string           `yaml:"state"`
	Dates    PermissionDates  `yaml:"dates,omitempty"`
	// Letter is the sha256:<hex> hash of the letter as it was sent.
	Letter string `yaml:"letter,omitempty"`
	// Answer is a short summary of the answer, with the hash of the answer mail when there is one.
	Answer string `yaml:"answer,omitempty"`
	Notes  string `yaml:"notes,omitempty"`
}

// PermissionContact is where the request goes. Every field comes from the official site of the holder. Email is a role address of the holder, never a personal or free-mail address.
type PermissionContact struct {
	URL     string `yaml:"url,omitempty"`
	Email   string `yaml:"email,omitempty"`
	Address string `yaml:"address,omitempty"`
}

// PermissionScope is the scope of a request or of a grant.
type PermissionScope struct {
	Statements bool `yaml:"statements,omitempty"`
	Solutions  bool `yaml:"solutions,omitempty"`
	// Years is all, YYYY, YYYY- or YYYY-YYYY.
	Years string `yaml:"years"`
	// Uses are the uses asked or granted, from PermissionUses.
	Uses []string `yaml:"uses"`
}

// PermissionDates are the dates of a request, as YYYY-MM-DD.
type PermissionDates struct {
	Drafted  string `yaml:"drafted,omitempty"`
	Sent     string `yaml:"sent,omitempty"`
	Answered string `yaml:"answered,omitempty"`
}

// Priorities are the priorities of a request, most urgent first (spec 11.4).
var Priorities = []string{"P0", "P1", "P2", "P3"}

// PermissionStates are the states of a request (spec 11.4).
var PermissionStates = []string{"draft", "sent", "granted", "refused", "no-answer"}

// PermissionUses are the uses that a request can ask for: site is the text on olymmath.org with attribution and a link back, export is the text in the open data releases (spec 11.5), and licence is an open licence such as CC BY 4.0 on the text.
var PermissionUses = []string{"site", "export", "licence"}

// freeMail are mail domains of persons. A request goes to a role address of the holder, so an address there is an error.
var freeMail = []string{"gmail.com", "googlemail.com", "yahoo.", "hotmail.", "outlook.", "live.", "msn.com", "aol.com", "icloud.com", "me.com", "mail.ru", "yandex.", "ya.ru", "bk.ru", "list.ru", "inbox.ru", "rambler.ru", "qq.com", "163.com", "126.com", "sina.com", "naver.com", "daum.net", "proton.me", "protonmail.", "gmx.", "web.de", "abv.bg", "seznam.cz", "wp.pl", "o2.pl", "interia.pl", "libero.it", "orange.fr", "free.fr", "laposte.net", "ukr.net", "rediffmail.com"}

var (
	yearsRE  = regexp.MustCompile(`^(all|\d{4}|\d{4}-|\d{4}-\d{4})$`)
	dateRE   = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])$`)
	letterRE = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	countryR = regexp.MustCompile(`^[A-Z]{2}$`)
)

// permissions reads registers/permissions.yaml. It runs after the sources and the competitions, so that it can check the codes and the source ids of each row.
func (l *loader) permissions() {
	const rel = "registers/permissions.yaml"
	if !l.exists(rel) {
		return
	}
	b, ok := l.read(rel)
	if !ok {
		return
	}
	var ps []*Permission
	if err := strictYAML(b, &ps); err != nil {
		l.add(rel, "", "%v", err)
		return
	}
	for _, p := range ps {
		switch {
		case !locator.ValidCode(p.ID):
			l.add(rel, "", "the permission id %q is not valid. Use lower case letters, digits and hyphens", p.ID)
			continue
		case l.corpus.Permissions[p.ID] != nil:
			l.add(rel, "", "the permission %q is in the register two times", p.ID)
			continue
		}
		for _, msg := range l.permissionErrors(p) {
			l.add(rel, "", "the permission %q: %s", p.ID, msg)
		}
		l.corpus.Permissions[p.ID] = p
	}
}

func (l *loader) permissionErrors(p *Permission) []string {
	var errs []string
	bad := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }
	if strings.TrimSpace(p.Holder) == "" {
		bad("it has no holder")
	}
	if p.Country != "international" && !countryR.MatchString(p.Country) {
		bad("the country %q is not an ISO 3166-1 alpha-2 code or international", p.Country)
	}
	if len(p.Competitions)+len(p.Sources) == 0 {
		bad("it covers no competition and no source")
	}
	for i, c := range p.Competitions {
		switch {
		case l.corpus.byCode[c] == nil:
			bad("the competition %q is not in registers/competitions.yaml", c)
		case slices.Contains(p.Competitions[:i], c):
			bad("the competition %q is in the row two times", c)
		}
	}
	for i, s := range p.Sources {
		switch {
		case l.corpus.bySource[s] == nil:
			bad("the source %q is not in registers/sources.yaml", s)
		case slices.Contains(p.Sources[:i], s):
			bad("the source %q is in the row two times", s)
		}
	}
	if !slices.Contains(Priorities, p.Priority) {
		bad("the priority %q is not one of %s", p.Priority, strings.Join(Priorities, ", "))
	}
	if !slices.Contains(PermissionStates, p.State) {
		bad("the state %q is not one of %s", p.State, strings.Join(PermissionStates, ", "))
	}
	if msg := scopeError(p.Scope); msg != "" {
		bad("the scope %s", msg)
	}
	if u := p.Contact.URL; u != "" {
		if v, err := url.Parse(u); err != nil || (v.Scheme != "https" && v.Scheme != "http") || v.Host == "" {
			bad("the contact url %q is not an http or https URL", u)
		}
	}
	if e := p.Contact.Email; e != "" {
		a, err := mail.ParseAddress(e)
		if err != nil || a.Address != e || a.Name != "" {
			bad("the contact email %q is not a bare mail address", e)
		} else if domain := strings.ToLower(e[strings.LastIndex(e, "@")+1:]); isFreeMail(domain) {
			bad("the contact email %q is at a free-mail domain. Use a role address of the holder from its official site", e)
		}
	}
	d := p.Dates
	for _, x := range []struct{ name, v string }{{"drafted", d.Drafted}, {"sent", d.Sent}, {"answered", d.Answered}} {
		if x.v != "" && !dateRE.MatchString(x.v) {
			bad("the date %s %q is not YYYY-MM-DD", x.name, x.v)
		}
	}
	if d.Drafted != "" && d.Sent != "" && d.Sent < d.Drafted {
		bad("it was sent before it was drafted")
	}
	if d.Sent != "" && d.Answered != "" && d.Answered < d.Sent {
		bad("it was answered before it was sent")
	}
	if p.Letter != "" && !letterRE.MatchString(p.Letter) {
		bad("the letter hash %q is not sha256:<64 hex digits>", p.Letter)
	}
	switch p.State {
	case "draft":
		if d.Sent != "" || d.Answered != "" || p.Answer != "" || p.Granted != nil {
			bad("a draft has no sent date, no answer and no grant")
		}
	case "sent", "no-answer":
		if d.Answered != "" || p.Answer != "" || p.Granted != nil {
			bad("a request in state %s has no answer date, no answer and no grant", p.State)
		}
	case "granted", "refused":
		if d.Answered == "" {
			bad("a request in state %s needs the date answered", p.State)
		}
		if strings.TrimSpace(p.Answer) == "" {
			bad("a request in state %s needs the answer", p.State)
		}
	}
	if p.State != "draft" && slices.Contains(PermissionStates, p.State) {
		if d.Sent == "" {
			bad("a request in state %s needs the date sent", p.State)
		}
		if p.Letter == "" {
			bad("a request in state %s needs the hash of the letter", p.State)
		}
		if p.Contact.URL == "" && p.Contact.Email == "" && p.Contact.Address == "" {
			bad("a request in state %s needs the contact it was sent to", p.State)
		}
	}
	switch {
	case p.State == "granted" && p.Granted == nil:
		bad("a granted request needs the granted scope")
	case p.State == "granted":
		if msg := scopeError(*p.Granted); msg != "" {
			bad("the granted scope %s", msg)
		}
	case p.State == "refused" && p.Granted != nil:
		bad("a refused request has no granted scope")
	}
	return errs
}

func scopeError(s PermissionScope) string {
	switch {
	case !s.Statements && !s.Solutions:
		return "covers neither statements nor solutions"
	case !yearsRE.MatchString(s.Years):
		return fmt.Sprintf("has the years %q, not all, YYYY, YYYY- or YYYY-YYYY", s.Years)
	case len(s.Years) == 9 && s.Years[5:] < s.Years[:4]:
		return fmt.Sprintf("has the years %q, which end before they start", s.Years)
	case len(s.Uses) == 0:
		return "has no uses. Give some of " + strings.Join(PermissionUses, ", ")
	}
	for _, u := range s.Uses {
		if !slices.Contains(PermissionUses, u) {
			return fmt.Sprintf("has the use %q, not one of %s", u, strings.Join(PermissionUses, ", "))
		}
	}
	return ""
}

func isFreeMail(domain string) bool {
	for _, f := range freeMail {
		if strings.HasSuffix(f, ".") {
			if strings.HasPrefix(domain, f) {
				return true
			}
		} else if domain == f {
			return true
		}
	}
	return false
}

// zoneBasis checks that each source in zone Z1 has a basis (spec 11.4 and 11.9): a licence on the source, or a granted row of registers/permissions.yaml named in its licence as permission:<id> that covers the source. It runs after the permissions.
func (l *loader) zoneBasis() {
	const rel = "registers/sources.yaml"
	for _, s := range l.corpus.Sources {
		lic := strings.TrimSpace(s.Licence)
		if id, ok := strings.CutPrefix(lic, "permission:"); ok {
			p := l.corpus.Permissions[id]
			switch {
			case p == nil:
				l.add(rel, "", "the source %q names the permission %q, which is not in registers/permissions.yaml", s.ID, id)
			case p.State != "granted":
				l.add(rel, "", "the source %q names the permission %q, which is in state %s, not granted", s.ID, id, p.State)
			case !slices.Contains(p.Sources, s.ID):
				l.add(rel, "", "the source %q names the permission %q, which does not cover it", s.ID, id)
			}
			continue
		}
		if lic == "" && (s.Zone.Statement == "Z1" || s.Zone.Solution == "Z1") {
			l.add(rel, "", "the source %q is in zone Z1 with no licence. Give the licence of the source, public-domain, or permission:<id> of a granted row of registers/permissions.yaml", s.ID)
		}
	}
}

// PermissionWarnings gives a warning for each granted row whose sources are not yet in zone Z1. The register does not change zones; a change of registers/sources.yaml applies the grant.
func PermissionWarnings(c *Corpus) Findings {
	var fs Findings
	for _, id := range slices.Sorted(maps.Keys(c.Permissions)) {
		p := c.Permissions[id]
		if p.State != "granted" {
			continue
		}
		for _, sid := range p.Sources {
			if s := c.Source(sid); s != nil && s.Zone.Statement != "Z1" {
				fs = append(fs, Finding{Path: "registers/permissions.yaml", Level: Warning, Message: fmt.Sprintf("the permission %q is granted, but the source %q is still in zone %s. Apply the grant in registers/sources.yaml with the licence permission:%s", id, sid, s.Zone.Statement, id)})
			}
		}
	}
	return fs
}
