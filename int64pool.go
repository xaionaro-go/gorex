package gorex

type int64Pool []*int64

func (pool *int64Pool) put(v *int64) {
	*v = 1
	*pool = append(*pool, v)
}

func (pool *int64Pool) get() *int64 {
	if len(*pool) == 0 {
		for i := 0; i < 100; i++ {
			pool.put(&[]int64{1}[0])
		}
	}

	idx := len(*pool) - 1
	v := (*pool)[idx]
	*pool = (*pool)[:idx]

	return v
}
