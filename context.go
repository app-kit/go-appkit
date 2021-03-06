package appkit

import (
	"errors"
	"fmt"
	"strconv"
)

type Context struct {
	Data map[string]interface{}
}

func NewContext(data ...map[string]interface{}) *Context {
	c := &Context{}

	if len(data) > 0 && data[0] != nil {
		c.Data = data[0]
	} else {
		c.Data = make(map[string]interface{})
	}

	return c
}

func (c Context) Has(key string) bool {
	_, ok := c.Data[key]
	return ok
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

func (c Context) Bool(key string) (bool, error) {
	val, ok := c.Get(key)
	if !ok {
		return false, errors.New("inexistant_key")
	}

	if boolVal, ok := val.(bool); ok {
		return boolVal, nil
	} else {
		return false, errors.New("type_mismatch")
	}
}

func (c Context) Map(key string) (map[string]interface{}, error) {
	val, ok := c.Get(key)
	if !ok {
		return nil, errors.New("inexistant_key")
	}

	if mapVal, ok := val.(map[string]interface{}); ok {
		return mapVal, nil
	} else {
		return nil, errors.New("type_mismatch")
	}
}

// Retrieve an int value from the context.
// Will auto-convert string values.
func (c *Context) Int(key string) (int, error) {
	val, ok := c.Get(key)
	if !ok {
		return 0, errors.New("inexistant_key")
	}

	if intVal, ok := val.(int); ok {
		return intVal, nil
	}

	if strVal, ok := val.(string); ok {
		intVal, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			return 0, err
		}

		return int(intVal), nil
	}

	return 0, errors.New("cant_convert_to_int")
}
