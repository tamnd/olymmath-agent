package site

import (
	"hash/fnv"
	"os"
	"path/filepath"
	"sync"
)

// writer writes the files of a build with several workers. All the writes of one path go to the same worker in the order of the calls, so a path that a build writes twice keeps its last data, as with one worker.
type writer struct {
	queues []chan file
	wg     sync.WaitGroup
	dirs   sync.Map // the directories that exist

	mu  sync.Mutex
	err error
}

type file struct {
	path string
	data []byte
}

func newWriter(workers int) *writer {
	w := &writer{queues: make([]chan file, workers)}
	for i := range w.queues {
		q := make(chan file, 256)
		w.queues[i] = q
		w.wg.Go(func() {
			for f := range q {
				if err := w.save(f); err != nil {
					w.mu.Lock()
					if w.err == nil {
						w.err = err
					}
					w.mu.Unlock()
				}
			}
		})
	}
	return w
}

// write queues a file. It returns the first error of an earlier write, so a build stops soon after a write fails.
func (w *writer) write(path string, data []byte) error {
	w.mu.Lock()
	err := w.err
	w.mu.Unlock()
	if err != nil {
		return err
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(path))
	w.queues[int(h.Sum32())%len(w.queues)] <- file{path, data}
	return nil
}

func (w *writer) save(f file) error {
	dir := filepath.Dir(f.path)
	if _, ok := w.dirs.Load(dir); !ok {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return err
		}
		w.dirs.Store(dir, true)
	}
	return os.WriteFile(f.path, f.data, 0o600)
}

// close waits for the queued files and returns the first error.
func (w *writer) close() error {
	for _, q := range w.queues {
		close(q)
	}
	w.wg.Wait()
	return w.err
}
