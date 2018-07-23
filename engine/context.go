package engine

import (
	"fmt"
	"math/rand"
)

type Context interface {
	Get(key string) interface{}
	Set(key string, val interface{})
	NewChild(kvs map[string]interface{}) Context

	Protect() Context
	Top() Context
}

type context struct {
	data   map[string]interface{}
	parent Context
	top    Context
}

func (c *context) Get(key string) interface{} {
	val, ok := c.data[key]
	if ok {
		return val
	}
	if c.parent != nil {
		return c.parent.Get(key)
	}
	return nil
}

func (c *context) Set(key string, val interface{}) {
	if val == nil {
		delete(c.data, key)
		return
	}

	c.data[key] = val
}

func (c *context) NewChild(kvs map[string]interface{}) Context {
	if kvs == nil {
		kvs = make(map[string]interface{})
	}
	return &context{
		data:   kvs,
		parent: c,
		top:    c.top,
	}
}

func (c *context) Protect() Context {
	result := &context{
		data:   make(map[string]interface{}),
		parent: c,
	}

	result.top = result
	return result
}

func (c *context) Top() Context {
	return c.top
}

func NewContext(kvs map[string]interface{}) Context {
	if kvs == nil {
		kvs = make(map[string]interface{})
	}

	result := &context{
		data:   kvs,
		parent: nil,
	}
	result.top = result
	return result
}

func FreshKey(ctx Context) string {
	key := fmt.Sprintf("_GKey%d", rand.Uint64())

	for val := ctx.Get(key); val != nil; {
		key = fmt.Sprintf("_GKey%d", rand.Uint64())
		val = ctx.Get(key)
	}

	return key
}
