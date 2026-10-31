package fetch

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"time"
)

// robots is the part of a robots.txt file that applies to one user agent.
type robots struct {
	allow    []string
	disallow []string
	delay    time.Duration
}

// parseRobots reads a robots.txt file and keeps the group of agent, or else the group of "*". The token of agent is the part before the first "/".
func parseRobots(r io.Reader, agent string) *robots {
	token := strings.ToLower(agent)
	if i := strings.IndexAny(token, "/ "); i >= 0 {
		token = token[:i]
	}
	type group struct {
		agents []string
		rules  robots
	}
	var groups []*group
	var cur *group
	inAgents := false
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k, v = strings.ToLower(strings.TrimSpace(k)), strings.TrimSpace(v)
		switch k {
		case "user-agent":
			if !inAgents || cur == nil {
				cur = &group{}
				groups = append(groups, cur)
			}
			cur.agents = append(cur.agents, strings.ToLower(v))
			inAgents = true
		case "allow", "disallow", "crawl-delay":
			inAgents = false
			if cur == nil {
				continue
			}
			switch k {
			case "allow":
				if v != "" {
					cur.rules.allow = append(cur.rules.allow, v)
				}
			case "disallow":
				if v != "" {
					cur.rules.disallow = append(cur.rules.disallow, v)
				}
			case "crawl-delay":
				if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
					cur.rules.delay = time.Duration(f * float64(time.Second))
				}
			}
		default:
			inAgents = false
		}
	}
	var star *robots
	for _, g := range groups {
		for _, a := range g.agents {
			if a == token {
				return &g.rules
			}
			if a == "*" && star == nil {
				star = &g.rules
			}
		}
	}
	if star != nil {
		return star
	}
	return &robots{}
}

// allowed tells if the rules let the agent get path. The longest rule that matches wins, and Allow wins a tie.
func (r *robots) allowed(path string) bool {
	best, ok := -1, true
	for _, p := range r.disallow {
		if n := match(p, path); n > best {
			best, ok = n, false
		}
	}
	for _, p := range r.allow {
		if n := match(p, path); n >= best && n >= 0 {
			best, ok = n, true
		}
	}
	return ok
}

// match returns the length of the pattern when it matches the start of path, or -1. A pattern can have "*" for any text and "$" at the end.
func match(pattern, path string) int {
	end := strings.HasSuffix(pattern, "$")
	p := strings.TrimSuffix(pattern, "$")
	parts := strings.Split(p, "*")
	if !strings.HasPrefix(path, parts[0]) {
		return -1
	}
	rest := path[len(parts[0]):]
	for _, part := range parts[1:] {
		i := strings.Index(rest, part)
		if i < 0 {
			return -1
		}
		rest = rest[i+len(part):]
	}
	if end && rest != "" && !strings.HasSuffix(p, "*") {
		return -1
	}
	return len(pattern)
}
