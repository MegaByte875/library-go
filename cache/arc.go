package cache

// ARCCache is Adaptive Replacement Cache (ARC).
type ARCCache struct {
	size int // Size is the total capacity of the cache
	p    int // P is the dynamic preference towards T1 or T2

	t1 *LRU // T1 is the LRU for recently accessed items
	b1 *LRU // B1 is the LRU for evictions from t1

	t2 *LRU // T2 is the LRU for frequently accessed items
	b2 *LRU // B2 is the LRU for evictions from t2
}

func NewARC(size int) (*ARCCache, error) {
	t1, err := NewLRU(size)
	if err != nil {
		return nil, err
	}
	b1, err := NewLRU(size)
	if err != nil {
		return nil, err
	}
	t2, err := NewLRU(size)
	if err != nil {
		return nil, err
	}
	b2, err := NewLRU(size)
	if err != nil {
		return nil, err
	}
	return &ARCCache{
		size: size,
		p:    0,
		t1:   t1,
		b1:   b1,
		t2:   t2,
		b2:   b2,
	}, nil
}

func (c *ARCCache) Get(key uint64) (any, bool) {
	// If the value is contained in T1 (recent), then
	// promote it to T2 (frequent)
	if value, ok := c.t1.Peek(key); ok {
		c.t1.Remove(key)
		c.t2.Put(key, value)
		return value, ok
	}

	// Check if the value is contained in T2 (frequent)
	if value, ok := c.t2.Get(key); ok {
		return value, ok
	}

	return nil, false
}

func (c *ARCCache) Put(key uint64, value any) {
	// Check if the value is contained in T1 (recent), and potentially
	// promote it to frequent T2
	if c.t1.Contains(key) {
		c.t1.Remove(key)
		c.t2.Put(key, value)
		return
	}

	// Check if the value is already in T2 (frequent) and update it
	if c.t2.Contains(key) {
		c.t2.Put(key, value)
		return
	}

	// Check if this value was recently evicted as part of the
	// recently used list
	if c.b1.Contains(key) {
		// T1 set is too small, increase P appropriately
		delta := 1
		b1Len := c.b1.Len()
		b2Len := c.b2.Len()
		if b2Len > b1Len {
			delta = b2Len / b1Len
		}
		if c.p+delta >= c.size {
			c.p = c.size
		} else {
			c.p += delta
		}

		// Potentially need to make room in the cache
		if c.t1.Len()+c.t2.Len() >= c.size {
			c.replace(false)
		}

		// Remove from B1
		c.b1.Remove(key)

		// Add the key to the frequently used list
		c.t2.Put(key, value)
		return
	}

	// Check if this value was recently evicted as part of the
	// frequently used list
	if c.b2.Contains(key) {
		// T2 set is too small, decrease P appropriately
		delta := 1
		b1Len := c.b1.Len()
		b2Len := c.b2.Len()
		if b1Len > b2Len {
			delta = b1Len / b2Len
		}
		if delta >= c.p {
			c.p = 0
		} else {
			c.p -= delta
		}

		// Potentially need to make room in the cache
		if c.t1.Len()+c.t2.Len() >= c.size {
			c.replace(true)
		}

		// Remove from B2
		c.b2.Remove(key)

		// Add the key to the frequently used list
		c.t2.Put(key, value)
		return
	}

	// Potentially need to make room in the cache
	if c.t1.Len()+c.t2.Len() >= c.size {
		c.replace(false)
	}

	// Keep the size of the ghost buffers trim
	if c.b1.Len() > c.size-c.p {
		c.b1.RemoveOldest()
	}
	if c.b2.Len() > c.p {
		c.b2.RemoveOldest()
	}

	// Add to the recently seen list
	c.t1.Put(key, value)
}

// replace is used to adaptively evict from either T1 or T2
// based on the current learned value of P
func (c *ARCCache) replace(b2ContainsKey bool) {
	if c.t1.Len() > 0 && (c.t1.Len() > c.p || (c.t1.Len() == c.p && b2ContainsKey)) {
		k, _, ok := c.t1.GetAndRemoveOldest()
		if ok {
			c.b1.Put(k, nil)
		}
	} else {
		k, _, ok := c.t2.GetAndRemoveOldest()
		if ok {
			c.b2.Put(k, nil)
		}
	}
}

func (c *ARCCache) Peek(key uint64) (any, bool) {
	if value, ok := c.t1.Peek(key); ok {
		return value, ok
	}
	return c.t2.Peek(key)
}

func (c *ARCCache) Remove(key uint64) {
	if c.t1.Remove(key) {
		return
	}
	if c.t2.Remove(key) {
		return
	}
	if c.b1.Remove(key) {
		return
	}
	if c.b2.Remove(key) {
		return
	}
}

func (c *ARCCache) Purge() {
	c.t1.Purge()
	c.t2.Purge()
	c.b1.Purge()
	c.b2.Purge()
}

func (c *ARCCache) Items() []*Item {
	elems := make([]*Item, 0, c.Len())
	elems = append(elems, c.t1.Items()...)
	elems = append(elems, c.t2.Items()...)
	return elems
}

func (c *ARCCache) Len() int {
	return c.t1.Len() + c.t2.Len()
}
