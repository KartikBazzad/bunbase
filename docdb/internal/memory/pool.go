package memory

import (
	"sync"
)

var defaultBufferSizes = []uint64{1024, 4096, 16384, 65536, 262144}

type BufferPool struct {
	pools []*sync.Pool
	sizes []uint64
}

func NewBufferPool(sizes []uint64) *BufferPool {
	if len(sizes) == 0 {
		sizes = defaultBufferSizes
	}

	pool := &BufferPool{
		pools: make([]*sync.Pool, len(sizes)),
		sizes: make([]uint64, len(sizes)),
	}

	for i, size := range sizes {
		pool.sizes[i] = size
		pool.pools[i] = &sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		}
	}

	return pool
}

func (p *BufferPool) Get(size uint64) []byte {
	idx := p.findBucket(size)
	if idx >= 0 {
		buf := p.pools[idx].Get().([]byte)
		return buf[:size]
	}
	return make([]byte, size)
}

func (p *BufferPool) Put(buf []byte) {
	capacity := uint64(cap(buf))
	idx := p.findBucket(capacity)
	if idx >= 0 && capacity == p.sizes[idx] {
		p.pools[idx].Put(buf)
	}
}

func (p *BufferPool) findBucket(size uint64) int {
	for i, bucketSize := range p.sizes {
		if size <= bucketSize {
			return i
		}
	}
	return -1
}

func (p *BufferPool) Sizes() []uint64 {
	return p.sizes
}
