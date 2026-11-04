package model

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Cache keeps each response in a directory under the key of its request. A resumed run sends the same requests and gets the stored responses, so it pays only for the calls that did not end.
type Cache struct {
	Client Client
	Dir    string
}

// Complete answers from the cache and sends the other requests to the client.
func (c *Cache) Complete(ctx context.Context, reqs []Request) ([]Response, error) {
	out := make([]Response, len(reqs))
	var todo []Request
	var at []int
	for i, r := range reqs {
		if res, ok := c.get(r); ok {
			res.ID, res.Cached = r.ID, true
			out[i] = res
			continue
		}
		todo = append(todo, r)
		at = append(at, i)
	}
	if len(todo) == 0 {
		return out, nil
	}
	got, err := c.Client.Complete(ctx, todo)
	if err != nil {
		return nil, err
	}
	for j, res := range got {
		out[at[j]] = res
		if res.Err == nil {
			if err := c.put(todo[j], res); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

func (c *Cache) path(r Request) string {
	k := r.Key()
	return filepath.Join(c.Dir, k[:2], k+".json")
}

func (c *Cache) get(r Request) (Response, bool) {
	b, err := os.ReadFile(c.path(r))
	if err != nil {
		return Response{}, false
	}
	var res Response
	if json.Unmarshal(b, &res) != nil {
		return Response{}, false
	}
	return res, true
}

func (c *Cache) put(r Request, res Response) error {
	p := c.path(r)
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Has reports whether the cache holds the response of the request.
func (c *Cache) Has(r Request) bool {
	_, err := os.Stat(c.path(r))
	return !errors.Is(err, fs.ErrNotExist)
}

// Budget stops a run before its calls cost more than Limit US dollars. Before it sends a request it adds the most the request can cost (Estimate); a request that does not fit gets ErrBudget and is not sent. Responses from the cache cost nothing.
type Budget struct {
	Client Client
	Limit  float64
	// Batch is true when the client uses the Batch API, so the estimates take half the price.
	Batch bool

	mu    sync.Mutex
	spent float64
}

// Spent gives the cost of the calls so far.
func (b *Budget) Spent() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.spent
}

// Charge adds the cost of a call that was paid before, for example in an earlier part of a resumed run.
func (b *Budget) Charge(usd float64) {
	b.mu.Lock()
	b.spent += usd
	b.mu.Unlock()
}

// Complete sends the requests that fit in the budget, in order.
func (b *Budget) Complete(ctx context.Context, reqs []Request) ([]Response, error) {
	out := make([]Response, len(reqs))
	b.mu.Lock()
	left := b.Limit - b.spent
	b.mu.Unlock()
	var send []Request
	var at []int
	for i, r := range reqs {
		est, err := Estimate(r, b.Batch)
		if err != nil {
			return nil, err
		}
		if est > left {
			out[i] = Response{ID: r.ID, Err: ErrBudget}
			continue
		}
		left -= est
		send = append(send, r)
		at = append(at, i)
	}
	if len(send) == 0 {
		return out, nil
	}
	got, err := b.Client.Complete(ctx, send)
	if err != nil {
		return nil, err
	}
	for j, res := range got {
		out[at[j]] = res
		if res.Err == nil && !res.Cached {
			c, err := Cost(send[j].Model, res.Usage, res.Batch)
			if err != nil {
				return nil, err
			}
			b.Charge(c)
		}
	}
	return out, nil
}
