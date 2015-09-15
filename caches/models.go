package caches

import(
	"time"

	. "github.com/theduke/go-appkit/error"
)

type StrItem struct {
	Key string
	Value string
	ExpiresAt time.Time
	Tags []string
}

// Ensure Item implements CacheItem
var _ CacheItem = (*StrItem)(nil)

func(i *StrItem) GetKey() string {
	return i.Key
}

func(i *StrItem) SetKey(x string) {
	i.Key = x
}

func(i *StrItem) GetValue() interface{} {
	return i.Value
}

func(i *StrItem) SetValue(x interface{}) {
	i.Value = x.(string)
}

func(i *StrItem) ToString() (string, Error) {
	return i.Value, nil
}

func(i *StrItem) FromString(x string) Error {
	i.Value = x
	return nil
}

func(i *StrItem) GetExpiresAt() time.Time {
	return i.ExpiresAt
}

func(i *StrItem) SetExpiresAt(x time.Time) {
	i.ExpiresAt = x
}

func(i *StrItem) GetTags() []string {
	return i.Tags
}

func(i *StrItem) SetTags(x []string) {
	i.Tags = x
}
