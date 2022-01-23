package cache

import (
	"sync"

	"github.com/viderstv/common/streaming/protocol/hls/item"
	"github.com/viderstv/common/utils/uid"
)

const (
	DefaultCacheSize = 10
)

type Cache struct {
	oldestIndex  int
	currentIndex int

	cache []*item.Item

	itemMtx sync.Mutex
	itemMap map[string]*item.Item
}

func New() *Cache {
	return NewWithSize(DefaultCacheSize)
}

func NewWithSize(size int) *Cache {
	return &Cache{
		cache:   make([]*item.Item, size),
		itemMap: map[string]*item.Item{},
	}
}

func (c *Cache) NewItem() *item.Item {
	c.itemMtx.Lock()
	defer c.itemMtx.Unlock()

	old := c.cache[c.oldestIndex]
	if old != nil {
		delete(c.itemMap, old.Name())
	}

	i := item.New(uid.NewId(), c.currentIndex)
	c.currentIndex++

	c.cache[c.oldestIndex] = i
	c.itemMap[i.Name()] = i
	c.oldestIndex = (c.oldestIndex + 1) % len(c.cache)

	return i
}

func (c *Cache) GetItem(key string) *item.Item {
	c.itemMtx.Lock()
	defer c.itemMtx.Unlock()

	return c.itemMap[key]
}
