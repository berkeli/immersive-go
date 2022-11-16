package cache

import (
	"crypto/md5"
	"sync"
	"time"
)

// This package provides a very simple cache. It's designed to hide the values of the keys because
// they will be used for storing authentication information, so keys are hashed before being used.
//
// 	c := Cache[int]()
// 	k := c.Key("secret number")
// 	c.Put(k, 42)
// 	...
// 	if v, ok := c.Get(k); ok {
//		...
// 	}

type Key [16]byte

type Entry[Value any] struct {
	value  *Value
	expiry int64 // expiration time in seconds
}

type Cache[Value any] struct {
	entries    *sync.Map
	last_flush int64 // unix timestamp of last flush
}

func New[Value any]() *Cache[Value] {
	return &Cache[Value]{
		entries: &sync.Map{},
	}
}

func (c *Cache[V]) Key(k string) Key {
	return md5.Sum([]byte(k))
}

func (c *Cache[Value]) Get(k Key) (*Value, bool) {
	if value, ok := c.entries.Load(k); ok {
		if entry, ok := value.(Entry[Value]); ok && entry.expiry > time.Now().Unix() {
			return entry.value, true
		}
	}
	return nil, false
}

func (c *Cache[Value]) Put(k Key, v *Value) {
	c.entries.Store(k, Entry[Value]{
		value:  v,
		expiry: time.Now().Unix() + 3600,
	})
	if time.Now().Unix() > c.last_flush+3600 {
		go c.Flush()
	}
}

// flush will delete stale data every hour
func (c *Cache[Value]) Flush() {
	c.entries.Range(func(k, v interface{}) bool {
		if entry, ok := v.(Entry[Value]); ok {
			if time.Now().Unix() > entry.expiry {
				c.entries.Delete(k)
			}
		}
		return true
	})

}
