package cache

import (
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/viderstv/common/streaming/protocol/hls/item"
	"github.com/viderstv/common/utils/uid"
)

const (
	DefaultCacheSize = 10
)

type Cache struct {
	oldestIndex  int
	currentIndex int

	cache map[int]*item.Item
	size  int

	itemMtx sync.Mutex
	itemMap map[string]*item.Item

	purgeCh chan int

	itemEvents chan bool
	once       sync.Once
}

func New() *Cache {
	return NewWithSize(DefaultCacheSize)
}

func NewWithSize(size int) *Cache {
	c := &Cache{
		size:       size,
		cache:      map[int]*item.Item{},
		itemMap:    map[string]*item.Item{},
		purgeCh:    make(chan int),
		itemEvents: make(chan bool, 1),
	}

	go c.purge()

	return c
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

	c.cache[c.currentIndex] = i
	c.itemMap[i.Name()] = i

	select {
	case c.itemEvents <- true:
	default:
		logrus.Debug("dropping item event")
	}

	c.purgeCh <- c.oldestIndex
	c.oldestIndex++

	return i
}

func (c *Cache) GetItem(key string) *item.Item {
	c.itemMtx.Lock()
	defer c.itemMtx.Unlock()

	return c.itemMap[key]
}

func (c *Cache) NewItems() <-chan bool {
	return c.itemEvents
}

func (c *Cache) Items() []*item.Item {
	c.itemMtx.Lock()
	defer c.itemMtx.Unlock()

	list := make([]*item.Item, c.currentIndex-c.oldestIndex)
	for i := c.oldestIndex; i < c.currentIndex; i++ {
		list[c.currentIndex-i] = c.cache[i]
	}

	return list
}

func (c *Cache) Stop() {
	c.once.Do(func() {
		close(c.itemEvents)
		close(c.purgeCh)
	})
}

func (c *Cache) purge() {
	stack := make([]int, 10)
	n := 0
	for i := range c.purgeCh {
		stack[n] = i
		n++
		if n == len(stack) {
			c.itemMtx.Lock()
			for _, v := range stack {
				item := c.cache[v]
				delete(c.cache, v)
				if item != nil {
					delete(c.itemMap, item.Name())
				}
			}
			c.itemMtx.Unlock()
			n = 0
		}
	}
}
