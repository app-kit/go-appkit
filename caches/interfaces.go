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

	ToString() (string, Error) 
	FromString(string) Error

	GetExpiresAt() time.Time
	SetExpiresAt(time.Time)
	IsExpired() bool

	GetTags() []string
	SetTags([]string)
}

type Cache interface {
	// Save a new item into the cache.
	Set(CacheItem) Error
	SetString(key string, value string, expiresAt *time.Time, tags []string) Error

	// Retrieve a cache item from the cache.
	Get(key string, item ...CacheItem) (CacheItem, Error)
	GetString(key string) (string, Error)

	// Delete item from the cache.
	Delete(key ...string) Error

	// Get all keys stored in the cache.
	Keys() ([]string, Error)

	// Return all keys that have a certain tag.
	KeysByTags(tag ...string) ([]string, Error)

	// Clear all items from the cache.
	Clear() Error

	// Clear all items with the specified tags.
	ClearTag(tag string) Error

	// Clean up all expired entries.
	Cleanup() Error

}
