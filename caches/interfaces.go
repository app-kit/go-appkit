package caches

import(
	"time"

	. "github.com/theduke/go-appkit/error"
)

type CacheItem interface {
	GetKey() string
	SetKey(string)

	GetValue() interface{}
	SetValue(interface{})

	GetString() string 
	SetString(string)

	GetExpiresAt() *time.Time
	SetExpiresAt(*time.Time)

	GetTags() []string
	SetTags([]string)
}

type Cache interface {
	// Save a new item into the cache.
	Set(CacheItem) Error
	SetValue(key, value interface{}, expiresAt *time.Time, tags []string) Error
	SetString(key, value string, expiresAt *time.Time, tags []string) Error

	// Retrieve a cache item from the cache.
	Get(key string) (CacheItem, Error)
	GetValue(key string) (interface{}, Error)
	GetString(key string) (string, Error)

	// Delete item from the cache.
	Delete(key string) Error
	DeleteItem(CacheItem) Error

	// Clear all items from the cache.
	Clear() Error

	// Clean up all expired entries.
	Cleanup() Error

	// Clear all items with the specified tags.
	ClearTag(tag ...string) Error
}
