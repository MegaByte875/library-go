package cache

import (
	"container/list"
	"errors"
)

// LRU is 'Least-Recently-Used' cache.
type LRU struct {
	size      int
	evictList *list.List
	items     map[uint64]*list.Element
}

type Item struct {
	Key   uint64
	Value any
}

func NewLRU(size int) (*LRU, error) {
	if size <= 0 {
		return nil, errors.New("must provide a positive size")
	}
	return &LRU{
		size:      size,
		evictList: list.New(),
		items:     make(map[uint64]*list.Element),
	}, nil
}

func (c *LRU) Get(key uint64) (any, bool) {
	if ele, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ele)
		return ele.Value.(*Item).Value, true
	}
	return nil, false
}

func (c *LRU) Put(key uint64, value any) {
	if ele, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ele)
		ele.Value.(*Item).Value = value
		return
	}

	item := &Item{Key: key, Value: value}
	ele := c.evictList.PushFront(item)
	c.items[key] = ele
	if c.Len() > c.size {
		c.removeOldest()
	}
}

func (c *LRU) Peek(key uint64) (any, bool) {
	if ele, ok := c.items[key]; ok {
		return ele.Value.(*Item).Value, ok
	}
	return nil, false
}

func (c *LRU) Remove(key uint64) {
	c.removeIfExist(key)
}

func (c *LRU) Purge() {
	for k := range c.items {
		delete(c.items, k)
	}
	c.evictList.Init()
}

func (c *LRU) Items() []*Item {
	items := make([]*Item, 0, c.evictList.Len())
	for ele := c.evictList.Front(); ele != nil; ele = ele.Next() {
		clone := *ele.Value.(*Item)
		items = append(items, &clone)
	}
	return items
}

func (c *LRU) contains(key uint64) bool {
	_, ok := c.items[key]
	return ok
}

func (c *LRU) removeOldest() {
	ele := c.evictList.Back()
	if ele != nil {
		c.removeElement(ele)
	}
}

func (c *LRU) getAndRemoveOldest() (uint64, any, bool) {
	ele := c.evictList.Back()
	if ele != nil {
		c.removeElement(ele)
		return ele.Value.(*Item).Key, ele.Value.(*Item).Value, true
	}
	return 0, nil, false
}

func (c *LRU) removeElement(ele *list.Element) {
	c.evictList.Remove(ele)
	item := ele.Value.(*Item)
	delete(c.items, item.Key)
}

func (c *LRU) removeIfExist(key uint64) bool {
	if ele, ok := c.items[key]; ok {
		c.removeElement(ele)
		return ok
	}
	return false
}

func (c *LRU) Len() int {
	return c.evictList.Len()
}
