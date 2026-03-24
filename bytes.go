package sync

import "bytes"

// NewBufferPool returns an initialized [BufferPool].
//
// The returned pool is ready for use and is backed by a generic [Pool] of
// [bytes.Buffer] values.
//
// The zero value of [BufferPool] is not ready for use; construct one with
// NewBufferPool.
func NewBufferPool() *BufferPool {
	return &BufferPool{pool: NewPool[bytes.Buffer]()}
}

// BufferPool provides pooled [bytes.Buffer] values.
//
// Buffers returned by [BufferPool.Get] should be considered temporarily borrowed
// by the caller. Return them to the pool via [BufferPool.Put] when finished to
// enable reuse and reduce allocations.
//
// The zero value is not ready for use.
type BufferPool struct {
	pool *Pool[bytes.Buffer]
}

// Get returns a buffer from the pool.
//
// The returned buffer is empty. New buffers start zeroed, and [BufferPool.Put]
// resets buffers before returning them to the pool.
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
// The returned slice does not alias the buffer's underlying array, so it is safe
// to keep after the buffer is returned to the pool.
//
// If buffer is nil, Copy returns nil.
func (p *BufferPool) Copy(buffer *bytes.Buffer) []byte {
	if buffer == nil {
		return nil
	}
	return bytes.Clone(buffer.Bytes())
}
