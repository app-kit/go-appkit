package appkit

import (
	"fmt"
)

type Context struct {
	Data map[string]interface{}
}

func NewContext() Context {
	c := Context{}
	c.Data = make(map[string]interface{})

	return c
}

func (c Context) Get(key string) (interface{}, bool) {
	x, ok := c.Data[key]
	return x, ok
}

func (c Context) MustGet(key string) interface{} {
	x, ok := c.Data[key]
	if !ok {
		panic("Context does not have key " + key)
	}

	return x
}

func (c *Context) Set(key string, data interface{}) {
	c.Data[key] = data
}

func (c Context) String(key string) string {
	x, ok := c.Data[key]
	if !ok {
		return ""
	}

	str, ok := x.(string)
	if !ok {
		return ""
	}
	return str
}

func (c Context) MustString(key string) string {
	val := c.MustGet(key)
	str, ok := val.(string)
	if !ok {
		panic(fmt.Sprintf("Context key %v is not a string", key))
	}

	return str
}

func (c *Context) SetString(key, val string) {
	c.Data[key] = val
}
