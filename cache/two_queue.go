package cache

import "errors"

const (
	// defaultRecentRatio is the ratio of the 2Q cache dedicated
	// to recently added entries that have only been accessed once.
	defaultRecentRatio = 0.25

	// defaultGhostEntriesRatio is the default ratio of ghost
	// entries kept to track entries recently evicted
	defaultGhostRatio = 0.50
)

type TwoQueueCache struct {
	size       int
	recentSize int

	recent   *LRU
	frequent *LRU
	evict    *LRU
}

func NewTwoQueueCache(size int) (*TwoQueueCache, error) {
	return newTowQueueParams(size, defaultRecentRatio, defaultGhostRatio)
}

func newTowQueueParams(size int, recentRatio, ghostRatio float64) (*TwoQueueCache, error) {
	if size <= 0 {
		return nil, errors.New("must provide a positive size")
	}
	if recentRatio < 0.0 || recentRatio > 1.0 {
		return nil, errors.New("invalid recent ratio")
	}
	if ghostRatio < 0.0 || ghostRatio > 1.0 {
		return nil, errors.New("invalid ghost ratio")
	}

	recentSize := int(float64(size) * recentRatio)
	evictSize := int(float64(size) * ghostRatio)

	recent, err := NewLRU(size)
	if err != nil {
		return nil, err
	}
	frequent, err := NewLRU(size)
	if err != nil {
		return nil, err
	}
	evict, err := NewLRU(evictSize)
	if err != nil {
		return nil, err
	}
	return &TwoQueueCache{
		size:       size,
		recentSize: recentSize,
		recent:     recent,
		frequent:   frequent,
		evict:      evict,
	}, nil
}

func (c *TwoQueueCache) Get(key uint64) (any, bool) {
	if val, ok := c.frequent.Get(key); ok {
		return val, ok
	}

	if val, ok := c.recent.Peek(key); ok {
		c.recent.Remove(key)
		c.frequent.Put(key, val)
		return val, ok
	}

	return nil, false
}

func (c *TwoQueueCache) Put(key uint64, value any) {
	if c.frequent.contains(key) {
		c.frequent.Put(key, value)
		return
	}

	if c.recent.contains(key) {
		c.recent.Remove(key)
		c.frequent.Put(key, value)
		return
	}

	if c.evict.contains(key) {
		c.ensureSpace(true)
		c.evict.Remove(key)
		c.frequent.Put(key, value)
		return
	}

	c.ensureSpace(false)
	c.recent.Put(key, value)
}

func (c *TwoQueueCache) ensureSpace(recentEvict bool) {
	// If we have space, nothing to do
	if c.recent.Len()+c.frequent.Len() < c.size {
		return
	}

	// If the recent buffer is larger than the target, evict from there
	if c.recent.Len() > 0 && (c.recent.Len() > c.recentSize || (c.recent.Len() == c.recentSize && !recentEvict)) {
		k, _, _ := c.recent.getAndRemoveOldest()
		c.evict.Put(k, nil)
		return
	}

	// Remove from the frequent list otherwise
	c.frequent.removeOldest()
}

func (c *TwoQueueCache) Peek(key uint64) (any, bool) {
	if val, ok := c.frequent.Peek(key); ok {
		return val, ok
	}
	return c.recent.Peek(key)
}

func (c *TwoQueueCache) Remove(key uint64) {
	if c.frequent.removeIfExist(key) {
		return
	}
	if c.recent.removeIfExist(key) {
		return
	}
	if c.evict.removeIfExist(key) {
		return
	}
}

func (c *TwoQueueCache) Purge() {
	c.recent.Purge()
	c.frequent.Purge()
	c.evict.Purge()
}

func (c *TwoQueueCache) Items() []*Item {
	elems := make([]*Item, 0, c.Len())
	elems = append(elems, c.recent.Items()...)
	elems = append(elems, c.frequent.Items()...)
	return elems
}

func (c *TwoQueueCache) Len() int {
	return c.recent.Len() + c.frequent.Len()
}
