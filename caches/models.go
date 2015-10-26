package caches

import (
	"encoding/json"
	"time"

	"github.com/theduke/go-apperror"

	kit "github.com/app-kit/go-appkit"
)

type StrItem struct {
	Key       string    `db:"primary-key;max:10000"`
	Value     string    `db:"not-null;max:-1"`
	ExpiresAt time.Time `db:"ignore-zero"`
	Tags      []string  `db:"ignore-zero;marshal"`
}

// Ensure Item implements CacheItem
var _ kit.CacheItem = (*StrItem)(nil)

func (i *StrItem) Collection() string {
	return "caches"
}

func (i *StrItem) GetID() interface{} {
	return i.Key
}

func (i *StrItem) SetID(key interface{}) error {
	i.Key = key.(string)
	return nil
}

func (i *StrItem) GetStrID() string {
	return i.Key
}

func (i *StrItem) SetStrID(key string) error {
	i.Key = key
	return nil
}

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

func (i *StrItem) ToString() (string, apperror.Error) {
	return i.Value, nil
}

func (i *StrItem) FromString(x string) apperror.Error {
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
	Value map[string]interface{} `db:"not-null;max:-1;marshal"`
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

func (i *MapItem) ToString() (string, apperror.Error) {
	if i.Value == nil {
		return "", nil
	}

	js, err := json.Marshal(i.Value)
	if err != nil {
		return "", apperror.Wrap(err, "cache_mapitem_marshal_error")
	}

	return string(js), nil
}

func (i *MapItem) FromString(x string) apperror.Error {
	if err := json.Unmarshal([]byte(x), &i.Value); err != nil {
		return apperror.Wrap(err, "cache_mapitem_unmarshal_error")
	}
	return nil
}

type Item struct {
	StrItem
	Value interface{} `db:"not-null;max:-1;marshal"`
}

func (i *Item) GetValue() interface{} {
	return i.Value
}

func (i *Item) SetValue(x interface{}) {
	i.Value = x
}

func (i *Item) ToString() (string, apperror.Error) {
	if i.Value == nil {
		return "", nil
	}

	js, err := json.Marshal(i.Value)
	if err != nil {
		return "", apperror.Wrap(err, "cache_item_marshal_error")
	}

	return string(js), nil
}

func (i *Item) FromString(x string) apperror.Error {
	if i.Value == nil {
		return &apperror.Err{
			Code:    "cache_item_empty_value",
			Message: "When using a generic Item{} for caching, the value must already be set to an empty struct to hold the information",
		}
	}

	if err := json.Unmarshal([]byte(x), &i.Value); err != nil {
		return apperror.Wrap(err, "cache_item_unmarshal_error")
	}
	return nil
}
