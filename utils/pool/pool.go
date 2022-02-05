package pool

type Pool struct {
	pos  int
	size int
	buf  []byte
}

const defaultPoolSize = 1024 * 1024

func (pool *Pool) Get(size int) []byte {
	if pool.size-pool.pos < size {
		pool.pos = 0
		if size > pool.size {
			pool.size = size
		}
		pool.buf = make([]byte, pool.size)
	}

	b := pool.buf[pool.pos : pool.pos+size]
	pool.pos += size
	return b
}

func NewPool() *Pool {
	return &Pool{
		size: defaultPoolSize,
		buf:  make([]byte, defaultPoolSize),
	}
}
