// Package locator reads and writes the locators and the om:// URIs of spec 04.2.
//
// A locator is a readable path. A competition is `imo`. An edition is `imo/2024`. An appearance is `imo/2024/p1`, or `amc12/2021/a/p25` when the edition has more than one paper. A locator can change when a curator fixes a mistake. A tag never changes.
package locator

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/tamnd/olymmath-agent/tag"
)

// Kind is the kind of record that a locator names.
type Kind int

// The kinds of locator.
const (
	Competition Kind = iota + 1
	Edition
	Appearance
)

func (k Kind) String() string {
	switch k {
	case Competition:
		return "competition"
	case Edition:
		return "edition"
	case Appearance:
		return "appearance"
	}
	return "unknown"
}

var (
	codeRE     = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	editionRE  = regexp.MustCompile(`^[0-9]{4}([a-z]|-[a-z0-9]+)?$`)
	paperRE    = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	positionRE = regexp.MustCompile(`^[a-z]{0,3}[0-9]{1,3}[a-z]?$`)
)

// Locator names a competition, an edition or an appearance.
type Locator struct {
	Comp     string
	Edition  string
	Paper    string
	Position string
}

// Kind returns the kind of record that l names.
func (l Locator) Kind() Kind {
	switch {
	case l.Position != "":
		return Appearance
	case l.Edition != "":
		return Edition
	}
	return Competition
}

// String returns the locator as a path.
func (l Locator) String() string {
	parts := []string{l.Comp}
	for _, p := range []string{l.Edition, l.Paper, l.Position} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "/")
}

// EditionOf returns the locator of the edition of l.
func (l Locator) EditionOf() Locator { return Locator{Comp: l.Comp, Edition: l.Edition} }

// Path returns the path of the problem file of an appearance, for example problems/imo/2024/p1.md.
func (l Locator) Path() string { return "problems/" + l.String() + ".md" }

// ValidCode reports whether s is a valid competition code: lowercase letters, digits and single hyphens, with a letter first.
func ValidCode(s string) bool { return codeRE.MatchString(s) }

// ValidEdition reports whether s is a valid edition: a year, then an optional letter or a hyphen and a word, for example 2024, 2021a or 2021-spring.
func ValidEdition(s string) bool { return editionRE.MatchString(s) }

// ValidPosition reports whether s is a valid position, for example p1, n1, b6 or 25.
func ValidPosition(s string) bool { return positionRE.MatchString(s) }

// ErrEmpty means that the locator is empty.
var ErrEmpty = errors.New("locator: empty")

// Parse reads a locator. It accepts one part (a competition), two parts (an edition), three parts (an appearance) or four parts (an appearance in a paper).
func Parse(s string) (Locator, error) {
	s = strings.Trim(s, "/")
	if s == "" {
		return Locator{}, ErrEmpty
	}
	parts := strings.Split(s, "/")
	var l Locator
	switch len(parts) {
	case 4:
		l.Paper = parts[2]
		l.Position = parts[3]
	case 3:
		l.Position = parts[2]
	case 1, 2:
	default:
		return Locator{}, fmt.Errorf("locator %q: want 1 to 4 parts, got %d", s, len(parts))
	}
	l.Comp = parts[0]
	if len(parts) > 1 {
		l.Edition = parts[1]
	}
	if !ValidCode(l.Comp) {
		return Locator{}, fmt.Errorf("locator %q: %q is not a valid competition code", s, l.Comp)
	}
	if l.Edition != "" && !ValidEdition(l.Edition) {
		return Locator{}, fmt.Errorf("locator %q: %q is not a valid edition", s, l.Edition)
	}
	if l.Paper != "" && !paperRE.MatchString(l.Paper) {
		return Locator{}, fmt.Errorf("locator %q: %q is not a valid paper", s, l.Paper)
	}
	if l.Position != "" && !ValidPosition(l.Position) {
		return Locator{}, fmt.Errorf("locator %q: %q is not a valid position", s, l.Position)
	}
	return l, nil
}

// Scheme is the scheme of olymmath URIs.
const Scheme = "om://"

// URI is a parsed om:// URI. Exactly one of Tag, Locator and Source is set.
type URI struct {
	Tag     string
	Locator *Locator
	Source  string
	Key     string
}

// String returns the URI in its canonical form.
func (u URI) String() string {
	switch {
	case u.Tag != "":
		return Scheme + u.Tag
	case u.Locator != nil:
		return Scheme + u.Locator.String()
	}
	return Scheme + "src/" + u.Source + "/" + u.Key
}

// ParseURI reads an om:// URI: om://<tag>, om://<locator> or om://src/<source>/<key>. A tag is normalized, and a tag with a wrong check character is an error.
func ParseURI(s string) (URI, error) {
	rest, ok := strings.CutPrefix(s, Scheme)
	if !ok {
		return URI{}, fmt.Errorf("uri %q: want the prefix %s", s, Scheme)
	}
	if src, ok := strings.CutPrefix(rest, "src/"); ok {
		name, key, _ := strings.Cut(src, "/")
		if !ValidCode(name) || key == "" {
			return URI{}, fmt.Errorf("uri %q: want om://src/<source>/<key>", s)
		}
		return URI{Source: name, Key: key}, nil
	}
	// A competition code is lowercase and starts with a letter. Anything else with one part is read as a tag.
	if !strings.Contains(rest, "/") && (rest == strings.ToUpper(rest) || !ValidCode(rest)) {
		t, err := tag.Normalize(rest)
		if err != nil {
			return URI{}, fmt.Errorf("uri %q: %w", s, err)
		}
		return URI{Tag: t}, nil
	}
	l, err := Parse(rest)
	if err != nil {
		return URI{}, fmt.Errorf("uri %q: %w", s, err)
	}
	return URI{Locator: &l}, nil
}
