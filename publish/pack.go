package publish

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PackDir is the folder of the packs in the upload directory and in the bucket.
const PackDir = "pack"

// PackIndex is the index of one pack. The Worker reads it from the static assets, and it reads a page from the pack with one range request.
type PackIndex struct {
	// Pack is the key of the pack in the bucket, for example pack/aime.3f2a9c1d0b7e.pack. The name has the hash of the pack, so a changed pack has a new key and an old Worker still reads its old pack.
	Pack string `json:"pack"`
	// Files maps the path of a file in its folder, for example 2020/p1.html, to its offset and length in the pack and the hash of its bytes.
	Files map[string]PackEntry `json:"files"`
}

// PackEntry is the place of one file in a pack. It is a JSON array [offset, length, hash], to keep the index small.
type PackEntry struct {
	Offset, Length int
	Hash           string
}

// MarshalJSON writes the entry as [offset, length, hash].
func (e PackEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{e.Offset, e.Length, e.Hash})
}

// UnmarshalJSON reads the entry from [offset, length, hash].
func (e *PackEntry) UnmarshalJSON(b []byte) error {
	var a []json.RawMessage
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	if len(a) != 3 {
		return fmt.Errorf("a pack entry has %d values, want 3", len(a))
	}
	if err := json.Unmarshal(a[0], &e.Offset); err != nil {
		return err
	}
	if err := json.Unmarshal(a[1], &e.Length); err != nil {
		return err
	}
	return json.Unmarshal(a[2], &e.Hash)
}

// PackResult tells what Pack did.
type PackResult struct {
	Files, Packs int
}

// Pack joins the files of the directory in into one pack for each top folder, such as aime. It writes the packs to out/pack and the index of each folder to index/FOLDER.json. A file that is not in a folder is an error. The packs and the indexes do not change when the files do not change, so an upload sends only the packs of the folders that changed.
func Pack(in, out, index string) (PackResult, error) {
	var res PackResult
	groups := map[string][]string{}
	err := filepath.WalkDir(in, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(in, p)
		if err != nil {
			return err
		}
		folder, rest, ok := strings.Cut(Key(rel), "/")
		if !ok {
			return fmt.Errorf("pack: %s is not in a folder", rel)
		}
		groups[folder] = append(groups[folder], rest)
		return nil
	})
	if err != nil {
		return res, err
	}
	for _, d := range []string{filepath.Join(out, PackDir), index} {
		if err := os.MkdirAll(d, 0o755); err != nil { // #nosec G301 -- the site is public.
			return res, err
		}
	}
	folders := make([]string, 0, len(groups))
	for f := range groups {
		folders = append(folders, f)
	}
	sort.Strings(folders)
	for _, folder := range folders {
		files := groups[folder]
		sort.Strings(files)
		var buf bytes.Buffer
		ix := PackIndex{Files: make(map[string]PackEntry, len(files))}
		for _, f := range files {
			b, err := os.ReadFile(filepath.Join(in, folder, filepath.FromSlash(f))) // #nosec G304 -- the path comes from the walk of the directory.
			if err != nil {
				return res, err
			}
			sum := sha256.Sum256(b)
			ix.Files[f] = PackEntry{Offset: buf.Len(), Length: len(b), Hash: hex.EncodeToString(sum[:8])}
			buf.Write(b)
		}
		sum := sha256.Sum256(buf.Bytes())
		ix.Pack = PackDir + "/" + folder + "." + hex.EncodeToString(sum[:6]) + ".pack"
		if err := os.WriteFile(filepath.Join(out, filepath.FromSlash(ix.Pack)), buf.Bytes(), 0o644); err != nil { // #nosec G306 -- the site is public.
			return res, err
		}
		js, err := json.Marshal(ix)
		if err != nil {
			return res, err
		}
		if err := os.WriteFile(filepath.Join(index, folder+".json"), js, 0o644); err != nil { // #nosec G306 G703 -- the site is public.
			return res, err
		}
		res.Files += len(files)
		res.Packs++
	}
	return res, nil
}
