package bytes

import (
	"bytes"

	"github.com/alexfalkowski/go-sync"
)

// NewBufferPool for bytes.
func NewBufferPool() *BufferPool {
	return &BufferPool{sync.NewPool[bytes.Buffer]()}
}

// BufferPool for bytes.
type BufferPool struct {
	*sync.Pool[bytes.Buffer]
}

// Get a new buffer.
func (p *BufferPool) Get() *bytes.Buffer {
	return p.Pool.Get()
}

// Put the buffer back.
func (p *BufferPool) Put(buffer *bytes.Buffer) {
	buffer.Reset()
	p.Pool.Put(buffer)
}
