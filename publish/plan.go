package publish

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/site"
)

// Place is where a file of the built site goes.
type Place int

const (
	// Static is a file that stays in the static assets.
	Static Place = iota
	// Gone is a redirect page. The Worker answers its path, so it goes nowhere.
	Gone
	// Bucket is a file that goes to R2 in a pack.
	Bucket
)

// Plan tells where each file of a built site goes.
type Plan struct {
	// Places gives the place of each file by its slash path.
	Places map[string]Place
	// Deleted is the number of redirect pages, Moved the number of files that Split moves, and Fitted the number of files that Fit moves.
	Deleted, Moved, Fitted int
	// Fit are the competitions that Fit moves.
	Fit []string
}

// Split gives the files of a place, by their slash paths, with their data.
func (p Plan) Split(files map[string][]byte, place Place) map[string][]byte {
	out := map[string][]byte{}
	for f, data := range files {
		if p.Places[f] == place {
			out[f] = data
		}
	}
	return out
}

// PlanSplit tells where each file of a built site goes, with files by their slash paths. It takes the redirect pages from redirects.json in files. It has the rules of Split and then of Fit with limit, so a release in memory gives the same static assets and the same packs as om site build, split and pack on disk. It gives an error, with the plan, when more than limit files stay.
func PlanSplit(files map[string][]byte, corpus *content.Corpus, t Tiers, limit int) (Plan, error) {
	p := Plan{Places: make(map[string]Place, len(files))}
	data, ok := files["redirects.json"]
	if !ok {
		return p, fmt.Errorf("the site has no redirects.json")
	}
	redirects, err := redirectFiles(data)
	if err != nil {
		return p, err
	}
	for _, f := range redirects {
		if _, ok := files[f]; ok {
			p.Places[f] = Gone
			p.Deleted++
		}
	}
	rules := Rules(corpus, t)
	static := 0
	for f := range files {
		if p.Places[f] == Gone {
			continue
		}
		folder, _, ok := strings.Cut(f, "/")
		if ok && rules[folder].Moves(path.Base(f)) {
			p.Places[f] = Bucket
			p.Moved++
			continue
		}
		p.Places[f] = Static
		static++
	}
	if static <= limit {
		return p, nil
	}
	// Fit counts the files that stay in the folder of each competition.
	counts := map[string]*Folder{}
	var folders []*Folder
	for _, c := range corpus.Competitions {
		if counts[c.Code] == nil {
			counts[c.Code] = &Folder{Code: c.Code}
		}
		folders = append(folders, counts[c.Code])
	}
	for f, place := range p.Places {
		if place != Static {
			continue
		}
		folder, _, ok := strings.Cut(f, "/")
		if c := counts[folder]; ok && c != nil {
			c.Files++
			if All.Moves(path.Base(f)) {
				c.Moves++
			}
		}
	}
	list := make([]Folder, len(folders))
	for i, f := range folders {
		list[i] = *f
	}
	p.Fit, err = FitOrder(static, list, limit)
	fit := map[string]bool{}
	for _, code := range p.Fit {
		fit[code] = true
	}
	for f, place := range p.Places {
		folder, _, ok := strings.Cut(f, "/")
		if place == Static && ok && fit[folder] && All.Moves(path.Base(f)) {
			p.Places[f] = Bucket
			p.Fitted++
		}
	}
	return p, err
}

// Release writes the files of a site that site.Render built, by the plan: the static assets to out, the packs to pack and the index of each pack to out/_pack. It replaces both directories. It gives the same files as om site build, split and pack, and it writes no redirect page and no file of R2 apart from the packs.
func Release(files map[string][]byte, p Plan, out, pack string) (PackResult, error) {
	for _, d := range []string{out, pack} {
		if err := os.RemoveAll(d); err != nil {
			return PackResult{}, err
		}
	}
	// The index of the packs is under out, so out exists first, with the mode of a build.
	if err := os.MkdirAll(out, 0o750); err != nil {
		return PackResult{}, err
	}
	// The packs and the static assets are different files, so they write at the same time.
	var res PackResult
	var perr error
	var wg sync.WaitGroup
	wg.Go(func() {
		res, perr = PackFiles(p.Split(files, Bucket), pack, filepath.Join(out, "_pack"))
	})
	err := site.Write(out, p.Split(files, Static))
	wg.Wait()
	if err == nil {
		err = perr
	}
	return res, err
}
