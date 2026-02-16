package sync

import "bytes"

// NewBufferPool returns an initialized [BufferPool].
//
// The returned pool is ready for use and is backed by a generic [Pool] of
// [bytes.Buffer] values.
func NewBufferPool() *BufferPool {
	return &BufferPool{pool: NewPool[bytes.Buffer]()}
}

// BufferPool provides pooled [bytes.Buffer] values.
//
// Buffers returned by [BufferPool.Get] should be considered temporarily borrowed
// by the caller. Return them to the pool via [BufferPool.Put] when finished to
// enable reuse and reduce allocations.
type BufferPool struct {
	pool *Pool[bytes.Buffer]
}

// Get returns a buffer from the pool.
//
// The returned buffer may contain previous contents; callers should call
// buffer.Reset() if they require an empty buffer. (Buffers returned by
// [BufferPool.Put] are reset before being returned to the pool.)
func (p *BufferPool) Get() *bytes.Buffer {
	return p.pool.Get()
}

// Put resets buffer and puts it back into the pool.
//
// If buffer is nil, Put is a no-op.
func (p *BufferPool) Put(buffer *bytes.Buffer) {
	if buffer == nil {
		return
	}
	buffer.Reset()
	p.pool.Put(buffer)
}

// Copy returns a copy of the buffer contents as a new byte slice.
//
// The returned slice does not alias the buffer's underlying array.
//
// If buffer is nil, Copy returns nil.
func (p *BufferPool) Copy(buffer *bytes.Buffer) []byte {
	if buffer == nil {
		return nil
	}
	return bytes.Clone(buffer.Bytes())
}
