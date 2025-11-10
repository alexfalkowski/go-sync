package sync

import "bytes"

// NewBufferPool for bytes.
func NewBufferPool() *BufferPool {
	return &BufferPool{NewPool[bytes.Buffer]()}
}

// BufferPool for bytes.
type BufferPool struct {
	*Pool[bytes.Buffer]
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
