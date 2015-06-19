package chunks

import (
	"bytes"
	"crypto/sha1"
	"io"
	"io/ioutil"

	"github.com/attic-labs/noms/ref"
)

// An in-memory implementation of store.ChunkStore. Useful mainly for tests.
type MemoryStore struct {
	data map[ref.Ref][]byte
	root ref.Ref
}

func (ms *MemoryStore) Root() ref.Ref {
	return ms.root
}

func (ms *MemoryStore) UpdateRoot(current, last ref.Ref) bool {
	if last != ms.root {
		return false
	}

	ms.root = current
	return true
}

func (ms *MemoryStore) Get(ref ref.Ref) (io.ReadCloser, error) {
	if b, ok := ms.data[ref]; ok {
		return ioutil.NopCloser(bytes.NewReader(b)), nil
	} else {
		return nil, nil
	}
}

func (ms *MemoryStore) Put() ChunkWriter {
	return &memoryChunkWriter{ms, &bytes.Buffer{}}
}

func (ms *MemoryStore) Len() int {
	return len(ms.data)
}

type memoryChunkWriter struct {
	ms  *MemoryStore
	buf *bytes.Buffer
}

func (w *memoryChunkWriter) Write(data []byte) (int, error) {
	return w.buf.Write(data)
}

func (w *memoryChunkWriter) Ref() (ref.Ref, error) {
	r := ref.New(sha1.Sum(w.buf.Bytes()))
	if w.ms.data == nil {
		w.ms.data = map[ref.Ref][]byte{}
	}
	w.ms.data[r] = w.buf.Bytes()
	w.Close()
	return r, nil
}

func (w *memoryChunkWriter) Close() error {
	// Not really necessary, but this will at least free memory and cause subsequent operations to crash.
	// BUG 17: Make this method consistent with other ChunkStore implementations.
	*w = memoryChunkWriter{}
	return nil
}