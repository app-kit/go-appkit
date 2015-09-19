package caches

import (
	"encoding/json"
	"time"

	kit "github.com/theduke/go-appkit"
)

type StrItem struct {
	Key       string
	Value     string
	ExpiresAt time.Time
	Tags      []string
}

// Ensure Item implements CacheItem
var _ kit.CacheItem = (*StrItem)(nil)

func (i *StrItem) GetKey() string {
	return i.Key
}

func (i *StrItem) SetKey(x string) {
	i.Key = x
}

func (i *StrItem) GetValue() interface{} {
	return i.Value
}

func (i *StrItem) SetValue(x interface{}) {
	if x == nil {
		i.Value = ""
	} else {
		i.Value = x.(string)
	}
}

func (i *StrItem) ToString() (string, kit.Error) {
	return i.Value, nil
}

func (i *StrItem) FromString(x string) kit.Error {
	i.Value = x
	return nil
}

func (i *StrItem) GetExpiresAt() time.Time {
	return i.ExpiresAt
}

func (i *StrItem) SetExpiresAt(x time.Time) {
	i.ExpiresAt = x
}

func (i *StrItem) IsExpired() bool {
	if i.ExpiresAt.IsZero() {
		return false
	}
	return i.ExpiresAt.Sub(time.Now()).Seconds() < 0
}

func (i *StrItem) GetTags() []string {
	return i.Tags
}

func (i *StrItem) SetTags(x []string) {
	i.Tags = x
}

type MapItem struct {
	StrItem
	Value map[string]interface{}
}

func (i *MapItem) GetValue() interface{} {
	return i.Value
}

func (i *MapItem) SetValue(x interface{}) {
	if x == nil {
		i.Value = nil
	} else {
		i.Value = x.(map[string]interface{})
	}
}

func (i *MapItem) ToString() (string, kit.Error) {
	if i.Value == nil {
		return "", nil
	}

	js, err := json.Marshal(i.Value)
	if err != nil {
		return "", kit.AppError{
			Code:     "cache_mapitem_marshal_error",
			Message:  err.Error(),
			Errors:   []error{err},
			Internal: true,
		}
	}

	return string(js), nil
}

func (i *MapItem) FromString(x string) kit.Error {
	if err := json.Unmarshal([]byte(x), &i.Value); err != nil {
		return kit.AppError{
			Code:     "cache_mapitem_unmarshal_error",
			Message:  err.Error(),
			Errors:   []error{err},
			Internal: true,
		}
	}
	return nil
}

type Item struct {
	StrItem
	Value interface{}
}

func (i *Item) GetValue() interface{} {
	return i.Value
}

func (i *Item) SetValue(x interface{}) {
	i.Value = x
}

func (i *Item) ToString() (string, kit.Error) {
	if i.Value == nil {
		return "", nil
	}

	js, err := json.Marshal(i.Value)
	if err != nil {
		return "", kit.AppError{
			Code:     "cache_item_marshal_error",
			Message:  err.Error(),
			Errors:   []error{err},
			Internal: true,
		}
	}

	return string(js), nil
}

func (i *Item) FromString(x string) kit.Error {
	if i.Value == nil {
		return kit.AppError{
			Code:     "cache_item_empty_value",
			Message:  "When using a generic Item{} for caching, the value must already be set to an empty struct to hold the information",
			Internal: true,
		}
	}

	if err := json.Unmarshal([]byte(x), &i.Value); err != nil {
		return kit.AppError{
			Code:     "cache_item_unmarshal_error",
			Message:  err.Error(),
			Errors:   []error{err},
			Internal: true,
		}
	}
	return nil
}
