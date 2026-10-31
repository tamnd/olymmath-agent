package seed

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Ranged is an io.ReaderAt over a URL. It reads blocks with HTTP range requests and keeps the last blocks in memory. Thus a reader of a few columns of a large Parquet file gets only the bytes of these columns.
type Ranged struct {
	URL    string
	Size   int64
	Client *http.Client
	Agent  string
	Token  string
	Block  int64
	Keep   int
	Ctx    context.Context

	mu    sync.Mutex
	cache map[int64][]byte
	order []int64
	// Fetched counts the bytes that came over the network.
	Fetched int64
}

// NewRanged returns a reader with blocks of 8 MB and a cache of 16 blocks.
func NewRanged(ctx context.Context, url string, size int64) *Ranged {
	return &Ranged{URL: url, Size: size, Client: http.DefaultClient, Block: 8 << 20, Keep: 16, Ctx: ctx}
}

// ReadAt reads len(p) bytes at off. It gets the blocks that it does not have with HTTP range requests and keeps the newest blocks in the cache.
func (r *Ranged) ReadAt(p []byte, off int64) (int, error) {
	if off >= r.Size {
		return 0, io.EOF
	}
	n := 0
	for n < len(p) && off+int64(n) < r.Size {
		at := off + int64(n)
		start := at - at%r.Block
		b, err := r.block(start)
		if err != nil {
			return n, err
		}
		n += copy(p[n:], b[at-start:])
	}
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (r *Ranged) block(start int64) ([]byte, error) {
	r.mu.Lock()
	if b, ok := r.cache[start]; ok {
		r.mu.Unlock()
		return b, nil
	}
	r.mu.Unlock()
	end := min(start+r.Block, r.Size) - 1
	b, err := r.get(start, end)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cache == nil {
		r.cache = map[int64][]byte{}
	}
	r.cache[start] = b
	r.order = append(r.order, start)
	r.Fetched += int64(len(b))
	for len(r.order) > r.Keep {
		delete(r.cache, r.order[0])
		r.order = r.order[1:]
	}
	return b, nil
}

// get reads one range. It tries again after a 429 or a 5xx, with back-off.
func (r *Ranged) get(start, end int64) ([]byte, error) {
	ctx := r.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	wait := 2 * time.Second
	for try := 0; ; try++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.URL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Range", "bytes="+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10))
		if r.Agent != "" {
			req.Header.Set("User-Agent", r.Agent)
		}
		if r.Token != "" {
			req.Header.Set("Authorization", "Bearer "+r.Token)
		}
		resp, err := r.Client.Do(req)
		if err == nil && resp.StatusCode == http.StatusPartialContent {
			b, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err == nil && int64(len(b)) == end-start+1 {
				return b, nil
			}
		} else if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
				return nil, fmt.Errorf("GET %s range %d-%d: %s", r.URL, start, end, resp.Status)
			}
			if s, e := strconv.Atoi(resp.Header.Get("Retry-After")); e == nil && s > 0 {
				wait = time.Duration(s) * time.Second
			}
		}
		if try >= 6 {
			if err == nil {
				err = fmt.Errorf("GET %s range %d-%d: too many tries", r.URL, start, end)
			}
			return nil, err
		}
		d := wait + time.Duration(rand.Int64N(int64(wait/2)+1)) // #nosec G404 -- the jitter of a retry needs no strong random numbers.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(d):
		}
		wait = min(wait*2, 2*time.Minute)
	}
}
