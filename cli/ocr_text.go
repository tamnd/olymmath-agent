package cli

import (
	"path"
	"strings"
)

// linkUnlisted makes each new problem whose position is not in keep a link only. keep is the value of -text, a comma-separated list of positions such as h1,h4. It is for a paper whose text layer was read by hand problem by problem: the text layer of some papers drops a formula with no sign, as "Let and for each , let .", so the checks pass and only a reader finds the gap. An empty keep changes nothing.
func linkUnlisted(news []newScan, keep string) {
	if keep == "" {
		return
	}
	set := map[string]bool{}
	for _, p := range strings.Split(keep, ",") {
		if p = strings.TrimSpace(p); p != "" {
			set[p] = true
		}
	}
	for i := range news {
		if news[i].scan.Text == "" || set[path.Base(news[i].at)] {
			continue
		}
		news[i].scan.Text, news[i].scan.Extraction, news[i].scan.Confidence = "", "", 0
		news[i].why = "a link only: the position is not in -text"
	}
}
