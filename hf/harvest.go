package hf

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/store"
)

// Harvest gets a dataset into the store.
type Harvest struct {
	Hub   *Client
	Fetch *fetch.Client
	// LocalMax is the largest file that goes to the local disk. A larger file goes to the remote store when the store has one, or else stays a reference.
	LocalMax int64
	// RemoteMax is the largest file that goes to the remote store. The R2 REST API has a limit on the size of a body, thus a larger file stays a reference. A file that R2 refuses with 413 also stays a reference. Zero means no limit.
	RemoteMax int64
	// MaxFiles is the largest number of data files that the harvest gets one by one. A dataset with more files uses the Parquet form of the hub. When MaxFiles is 0, the harvest always gets the files of the dataset.
	MaxFiles int
	// Log gets one line for each file. It can be nil.
	Log io.Writer
}

// Run harvests the dataset id for source and returns the manifest and its capture.
func (h *Harvest) Run(ctx context.Context, source, id string) (*Manifest, store.Capture, error) {
	info, err := h.Hub.Info(ctx, id)
	if err != nil {
		return nil, store.Capture{}, err
	}
	m := &Manifest{Source: source, Dataset: info.ID, Revision: info.SHA, LastModified: info.LastModified, Licence: info.Licence(), Fetched: time.Now().UTC()}
	files, err := h.Hub.Files(ctx, info.ID, info.SHA)
	if err != nil {
		return nil, store.Capture{}, err
	}
	data := DataFiles(files)
	rev := info.SHA
	// A dataset with no data files of its own has rows that a script makes. A dataset with very many files is slow to get one file at a time. The hub has a Parquet form of both, and the harvest pins the commit of that form.
	if len(data) == 0 || (h.MaxFiles > 0 && len(data) > h.MaxFiles) {
		conv, err := h.Hub.Revision(ctx, info.ID, ParquetRevision)
		if err != nil {
			return nil, store.Capture{}, fmt.Errorf("the dataset has %d data files and no Parquet form: %w", len(data), err)
		}
		files, err = h.Hub.Files(ctx, info.ID, conv)
		if err != nil {
			return nil, store.Capture{}, err
		}
		data, rev, m.Converted, m.ParquetRevision = DataFiles(files), conv, true, conv
	}
	if len(data) == 0 {
		return nil, store.Capture{}, errors.New("the dataset has no data files")
	}
	for _, f := range data {
		mf, err := h.file(ctx, source, info.ID, rev, f, m.Licence)
		if err != nil {
			return nil, store.Capture{}, err
		}
		m.Files = append(m.Files, mf)
		if h.Log != nil {
			fmt.Fprintf(h.Log, "%s\t%d\t%s\t%s\n", mf.Where, mf.Size, mf.SHA256, mf.Path)
		}
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, store.Capture{}, err
	}
	c, err := h.Fetch.Store.PutBytes(append(b, '\n'), store.Capture{Source: source, URL: Base + "/datasets/" + info.ID + "@" + info.SHA, Type: "application/json", Licence: m.Licence, Note: "manifest"})
	return m, c, err
}

func (h *Harvest) file(ctx context.Context, source, id, rev string, f File, licence string) (ManifestFile, error) {
	u := ResolveURL(id, rev, f.Path)
	mf := ManifestFile{Path: f.Path, Size: f.Size, SHA256: f.SHA256(), URL: u}
	st := h.Fetch.Store
	remote := f.Size > h.LocalMax && h.LocalMax > 0
	if remote && (st.Remote == nil || mf.SHA256 == "" || (h.RemoteMax > 0 && f.Size > h.RemoteMax)) {
		mf.Where = "ref"
		return mf, nil
	}
	r, err := h.Fetch.Get(ctx, source, u, fetch.Options{Hash: mf.SHA256, Remote: remote, Licence: licence})
	if remote && errors.Is(err, store.ErrTooLarge) {
		mf.Where = "ref"
		return mf, nil
	}
	if err != nil {
		return mf, err
	}
	mf.SHA256, mf.Size = r.Capture.Hash, r.Capture.Size
	switch {
	case r.Capture.Local:
		mf.Where = "local"
	case r.Capture.Remote:
		mf.Where = "r2"
	default:
		mf.Where = "ref"
	}
	return mf, nil
}

// LatestManifest returns the newest manifest of source in the store.
func LatestManifest(ctx context.Context, st *store.Store, source string) (*Manifest, error) {
	cs, err := st.Captures(source)
	if err != nil {
		return nil, err
	}
	for i := len(cs) - 1; i >= 0; i-- {
		if cs[i].Note != "manifest" {
			continue
		}
		rc, err := st.Get(ctx, cs[i].Hash)
		if err != nil {
			return nil, err
		}
		var m Manifest
		err = json.NewDecoder(rc).Decode(&m)
		rc.Close()
		if err != nil {
			return nil, err
		}
		return &m, nil
	}
	return nil, fmt.Errorf("%s: no manifest in the store", source)
}
