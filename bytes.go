package sync

import "bytes"

// NewBufferPool returns a BufferPool that reuses bytes.Buffer instances.
func NewBufferPool() *BufferPool {
	return &BufferPool{pool: NewPool[bytes.Buffer]()}
}

// BufferPool provides pooled *bytes.Buffer values.
type BufferPool struct {
	pool *Pool[bytes.Buffer]
}

// Get returns a buffer from the pool.
func (p *BufferPool) Get() *bytes.Buffer {
	return p.pool.Get()
}

// Put resets buffer and returns it to the pool.
func (p *BufferPool) Put(buffer *bytes.Buffer) {
	if buffer == nil {
		return
	}
	buffer.Reset()
	p.pool.Put(buffer)
}

// Copy returns a copy of the buffer contents as a new byte slice.
func (p *BufferPool) Copy(buffer *bytes.Buffer) []byte {
	if buffer == nil {
		return nil
	}
	return bytes.Clone(buffer.Bytes())
}
