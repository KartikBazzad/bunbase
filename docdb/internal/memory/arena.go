package memory

type Arena struct {
	buffers [][]byte
	pool    *BufferPool
}

func NewArena(pool *BufferPool) *Arena {
	return &Arena{
		buffers: make([][]byte, 0),
		pool:    pool,
	}
}

func (a *Arena) Alloc(size uint64) []byte {
	buf := a.pool.Get(size)
	a.buffers = append(a.buffers, buf)
	return buf
}

func (a *Arena) Release() {
	for _, buf := range a.buffers {
		a.pool.Put(buf)
	}
	a.buffers = nil
}

func (a *Arena) Size() int {
	return len(a.buffers)
}
