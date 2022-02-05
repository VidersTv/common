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

	itemCh chan *item.Item

	itemMtx sync.Mutex
	itemMap map[string]*item.Item

	purgeCh chan int

	itemEvents chan bool
	once       sync.Once
	done       chan struct{}
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
		itemCh:     make(chan *item.Item, 3),
		itemEvents: make(chan bool, 1),
		done:       make(chan struct{}),
	}

	go c.purge()
	go c.fill()

	return c
}

func (c *Cache) fill() {
	for {
		c.itemMtx.Lock()
		item := item.New(uid.NewId(), c.currentIndex)
		c.cache[item.SeqNum()] = item
		c.itemMap[item.Name()] = item
		c.itemMtx.Unlock()
		c.currentIndex++
		select {
		case <-c.done:
			return
		case c.itemCh <- item:
		}

	}
}

func (c *Cache) NewItem() *item.Item {
	i := <-c.itemCh

	select {
	case c.itemEvents <- true:
	default:
		logrus.Debug("dropping item event")
	}

	if c.currentIndex > c.size {
		c.purgeCh <- c.oldestIndex
		c.oldestIndex++
	}

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

	list := []*item.Item{}
	for i := c.oldestIndex; i < c.currentIndex; i++ {
		list = append(list, c.cache[i])
	}

	return list
}

func (c *Cache) Stop() {
	c.once.Do(func() {
		close(c.itemEvents)
		close(c.purgeCh)
		close(c.done)
		close(c.itemCh)
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

func (c *Cache) Wait() <-chan struct{} {
	return c.done
}

func (c *Cache) Done() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}
