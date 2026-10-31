package official

import (
	"context"
	"io"

	"github.com/tamnd/olymmath-agent/store"
)

// Load reads the papers of a source from the captures of the store. It reads only the papers of the list, and it skips a paper that is not in the store or that has no readable text.
func Load(ctx context.Context, st *store.Store, source string, papers []Paper) (Papers, error) {
	out := Papers{}
	for _, p := range papers {
		c, ok, err := st.Latest(source, p.URL)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		rc, err := st.Get(ctx, c.Hash)
		if err != nil {
			continue
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		text, err := Text(ctx, b)
		if err != nil {
			continue
		}
		out[p.Edition] = p.Segments(text)
	}
	return out, nil
}
