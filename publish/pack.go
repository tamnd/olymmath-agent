package publish

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PackDir is the folder of the packs in the upload directory and in the bucket.
const PackDir = "pack"

// ShardFiles is the number of files above which a folder gets Shards packs and not one. A big folder such as api or p changes at each deploy, and one changed file must not send the whole folder again.
const ShardFiles = 2000

// Shards is the number of packs of a big folder. It is fixed, so a file stays in the same shard from one build to the next.
const Shards = 32

// PackIndex is the index of the packs of one folder. The Worker reads it from the static assets, and it reads a page from a pack with one range request.
type PackIndex struct {
	// Pack is the key of the pack of a small folder in the bucket, for example pack/aime.3f2a9c1d0b7e.pack. The name has the hash of the pack, so a changed pack has a new key and an old Worker still reads its old pack.
	Pack string `json:"pack,omitempty"`
	// Packs are the keys of the shards of a big folder, for example pack/api-07.3f2a9c1d0b7e.pack. The entry of a file gives its shard.
	Packs []string `json:"packs,omitempty"`
	// Files maps the path of a file in its folder, for example 2020/p1.html, to its offset and length in the pack and the hash of its bytes.
	Files map[string]PackEntry `json:"files,omitempty"`
	// IndexShards is set in the index of a big folder, which has no Files: the entries are in IndexShards files FOLDER/NN.json, and a file is in the one of its shard (shardOf). The Worker then reads a small part of the index for a page, not the index of the whole folder.
	IndexShards int `json:"shards,omitempty"`
}

// PackEntry is the place of one file in a pack. It is a JSON array [offset, length, hash], or [offset, length, hash, shard] in a big folder, to keep the index small.
type PackEntry struct {
	Offset, Length int
	Hash           string
	// Shard is the index of the pack in Packs. It is -1 in a small folder.
	Shard int
}

// MarshalJSON writes the entry as [offset, length, hash] or [offset, length, hash, shard].
func (e PackEntry) MarshalJSON() ([]byte, error) {
	if e.Shard < 0 {
		return json.Marshal([]any{e.Offset, e.Length, e.Hash})
	}
	return json.Marshal([]any{e.Offset, e.Length, e.Hash, e.Shard})
}

// UnmarshalJSON reads the entry from [offset, length, hash] or [offset, length, hash, shard].
func (e *PackEntry) UnmarshalJSON(b []byte) error {
	var a []json.RawMessage
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	if len(a) != 3 && len(a) != 4 {
		return fmt.Errorf("a pack entry has %d values, want 3 or 4", len(a))
	}
	e.Shard = -1
	if len(a) == 4 {
		if err := json.Unmarshal(a[3], &e.Shard); err != nil {
			return err
		}
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

// Pack joins the files of the directory in into one pack for each top folder, such as aime, and into Shards packs for a folder with more than ShardFiles files. It writes the packs to out/pack and the index of each folder to index/FOLDER.json. A file that is not in a folder is an error. The packs and the indexes do not change when the files do not change, so an upload sends only the packs of the folders that changed.
func Pack(in, out, index string) (PackResult, error) {
	groups := map[string][]string{}
	err := filepath.WalkDir(in, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(in, p)
		if err != nil {
			return err
		}
		return group(groups, Key(rel))
	})
	if err != nil {
		return PackResult{}, err
	}
	return packGroups(groups, out, index, func(f string) ([]byte, error) {
		return os.ReadFile(filepath.Join(in, filepath.FromSlash(f))) // #nosec G304 -- the path comes from the walk of the directory.
	})
}

// PackFiles packs files, by their slash paths, as Pack packs a directory with these files. It gives the same packs and indexes, and it reads no file.
func PackFiles(files map[string][]byte, out, index string) (PackResult, error) {
	groups := map[string][]string{}
	for rel := range files {
		if err := group(groups, rel); err != nil {
			return PackResult{}, err
		}
	}
	return packGroups(groups, out, index, func(f string) ([]byte, error) { return files[f], nil })
}

// group adds the file at the slash path rel to the files of its top folder.
func group(groups map[string][]string, rel string) error {
	folder, rest, ok := strings.Cut(rel, "/")
	if !ok {
		return fmt.Errorf("pack: %s is not in a folder", rel)
	}
	groups[folder] = append(groups[folder], rest)
	return nil
}

// packGroups writes the packs of each folder of groups and their indexes. read gives the data of a file by its slash path.
func packGroups(groups map[string][]string, out, index string, read func(f string) ([]byte, error)) (PackResult, error) {
	var res PackResult
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
	// Each folder gives its own pack, so the workers share the folders.
	err := each(len(folders), func(i int) error {
		return packFolder(out, index, folders[i], groups[folders[i]], read)
	})
	if err != nil {
		return res, err
	}
	for _, f := range folders {
		res.Files += len(groups[f])
		res.Packs += shards(len(groups[f]))
	}
	return res, nil
}

// shards gives the number of packs of a folder with n files.
func shards(n int) int {
	if n > ShardFiles {
		return Shards
	}
	return 1
}

// shardOf gives the shard of a file. It depends only on the path, so a change of one file changes one shard.
func shardOf(f string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(f))
	return int(h.Sum32() % Shards)
}

// packFolder writes the packs of one folder and its index.
func packFolder(out, index, folder string, files []string, read func(f string) ([]byte, error)) error {
	sort.Strings(files)
	n := shards(len(files))
	bufs := make([]bytes.Buffer, n)
	ix := PackIndex{Files: make(map[string]PackEntry, len(files))}
	for _, f := range files {
		b, err := read(folder + "/" + f)
		if err != nil {
			return err
		}
		s := -1
		if n > 1 {
			s = shardOf(f)
		}
		buf := &bufs[max(s, 0)]
		sum := sha256.Sum256(b)
		ix.Files[f] = PackEntry{Offset: buf.Len(), Length: len(b), Hash: hex.EncodeToString(sum[:8]), Shard: s}
		buf.Write(b)
	}
	for i := range bufs {
		name := folder
		if n > 1 {
			name = fmt.Sprintf("%s-%02d", folder, i)
		}
		sum := sha256.Sum256(bufs[i].Bytes())
		key := PackDir + "/" + name + "." + hex.EncodeToString(sum[:6]) + ".pack"
		if err := os.WriteFile(filepath.Join(out, filepath.FromSlash(key)), bufs[i].Bytes(), 0o644); err != nil { // #nosec G306 -- the site is public.
			return err
		}
		if n > 1 {
			ix.Packs = append(ix.Packs, key)
		} else {
			ix.Pack = key
		}
	}
	if n > 1 {
		return writeShards(index, folder, ix)
	}
	return writeJSON(filepath.Join(index, folder+".json"), ix)
}

// writeShards writes the index of a big folder as a head with the packs and one file for the entries of each shard.
func writeShards(index, folder string, ix PackIndex) error {
	parts := make([]PackIndex, Shards)
	for f, e := range ix.Files {
		if parts[e.Shard].Files == nil {
			parts[e.Shard].Files = map[string]PackEntry{}
		}
		parts[e.Shard].Files[f] = e
	}
	dir := filepath.Join(index, folder)
	if err := os.MkdirAll(dir, 0o755); err != nil { // #nosec G301 -- the site is public.
		return err
	}
	for i, part := range parts {
		if part.Files == nil {
			part.Files = map[string]PackEntry{}
		}
		if err := writeJSON(filepath.Join(dir, fmt.Sprintf("%02d.json", i)), part); err != nil {
			return err
		}
	}
	return writeJSON(filepath.Join(index, folder+".json"), PackIndex{Packs: ix.Packs, IndexShards: Shards})
}

func writeJSON(name string, v any) error {
	js, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(name, js, 0o644) // #nosec G306 G703 -- the site is public.
}
