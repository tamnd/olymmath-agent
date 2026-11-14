package cli

import (
	"os"
	"runtime/metrics"
	"runtime/pprof"
	"sync"
	"time"
)

// heapProfile writes a heap profile to path near the peak of the heap. It checks the heap every half second and writes the profile again each time the heap is 5% above the heap of the last profile. The function it gives stops the checks and writes a last profile when the heap is at a new peak. A path of "" does nothing.
func heapProfile(path string) (stop func() error) {
	if path == "" {
		return func() error { return nil }
	}
	sample := []metrics.Sample{{Name: "/memory/classes/heap/objects:bytes"}}
	var mu sync.Mutex
	var peak uint64
	var werr error
	check := func() {
		metrics.Read(sample)
		heap := sample[0].Value.Uint64()
		mu.Lock()
		defer mu.Unlock()
		if heap <= peak+peak/20 {
			return
		}
		peak = heap
		f, err := os.Create(path) // #nosec G304 -- the path comes from a flag of the user.
		if err == nil {
			err = pprof.Lookup("heap").WriteTo(f, 0)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
		}
		werr = err
	}
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		t := time.NewTicker(500 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				check()
			}
		}
	})
	return func() error {
		close(done)
		wg.Wait()
		check()
		mu.Lock()
		defer mu.Unlock()
		return werr
	}
}
