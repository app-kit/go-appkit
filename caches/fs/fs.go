package fs

import (
	"encoding/json"
	"os"
	"path"
	"strings"
	"time"

	kit "github.com/theduke/go-appkit"
	. "github.com/theduke/go-appkit/caches"
	"github.com/theduke/go-appkit/utils"
)

type Fs struct {
	name string
	path string
}

// Ensure redis implements the Cache interface.
var _ kit.Cache = (*Fs)(nil)

func fsErr(err error) kit.Error {
	return kit.AppError{
		Code:     "fs_error",
		Message:  err.Error(),
		Errors:   []error{err},
		Internal: true,
	}
}

func New(path string) (*Fs, kit.Error) {
	if path == "" {
		return nil, kit.AppError{
			Code:     "empty_path",
			Internal: true,
		}
	}

	if err := os.MkdirAll(path, 0777); err != nil {
		return nil, kit.AppError{
			Code:     "root_path_unwritable",
			Message:  err.Error(),
			Errors:   []error{err},
			Internal: true,
		}
	}

	fs := &Fs{
		name: "fs",
		path: path,
	}

	return fs, nil
}

func (fs *Fs) Name() string {
	return fs.name
}

func (fs *Fs) SetName(x string) {
	fs.name = x
}

func (fs *Fs) key(rawKey string) string {
	return utils.Canonicalize(rawKey)
}

func (fs *Fs) keyPath(key string) string {
	return path.Join(fs.path, key)
}

func (fs *Fs) keyMetaPath(key string) string {
	return path.Join(fs.path, key+".meta")
}

// Save a new item into the cache.
func (fs *Fs) Set(item kit.CacheItem) kit.Error {
	key := fs.key(item.GetKey())
	if key == "" {
		return kit.AppError{Code: "empty_key"}
	}

	if item.IsExpired() {
		return kit.AppError{Code: "item_expired"}
	}

	value, err := item.ToString()
	if err != nil {
		return kit.AppError{
			Code:     "cacheitem_tostring_error",
			Message:  err.Error(),
			Errors:   []error{err},
			Internal: true,
		}
	}
	if value == "" {
		return kit.AppError{Code: "empty_value"}
	}

	// Marshal metadata.
	tmpVal := item.GetValue()
	item.SetValue(nil)

	js, err2 := json.Marshal(item)
	if err2 != nil {
		return kit.AppError{
			Code:     "json_marshal_error",
			Message:  err2.Error(),
			Internal: true,
		}
	}
	item.SetValue(tmpVal)

	if err := utils.WriteFile(fs.keyPath(key), []byte(value), false); err != nil {
		return err
	}
	if err := utils.WriteFile(fs.keyMetaPath(key), js, false); err != nil {
		return err
	}

	return nil
}

func (fs *Fs) SetString(key string, value string, expiresAt *time.Time, tags []string) kit.Error {
	item := &StrItem{
		Key:   key,
		Value: value,
		Tags:  tags,
	}
	if expiresAt != nil {
		item.ExpiresAt = *expiresAt
	}

	return fs.Set(item)
}

// Retrieve a cache item from the cache.
func (fs *Fs) Get(key string, items ...kit.CacheItem) (kit.CacheItem, kit.Error) {
	var item kit.CacheItem = &StrItem{}
	if items != nil {
		if len(items) != 1 {
			return nil, kit.AppError{
				Code:     "invalid_item",
				Message:  "You must specify one item only",
				Internal: true,
			}
		}
		item = items[0]
	}

	key = fs.key(key)
	if key == "" {
		return nil, kit.AppError{Code: "empty_key"}
	}

	exists, err := utils.FileExists(fs.keyPath(key))
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	metaContent, err := utils.ReadFile(fs.keyMetaPath(key))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(metaContent, &item); err != nil {
		return nil, kit.AppError{
			Code:     "metadata_unmarshal_error",
			Message:  err.Error(),
			Internal: true,
		}
	}

	// Reset ExpiresAt if it is zero, since json.Unmarshal produces
	// a time.Time which is not exactly equal to the zero value.
	if item.GetExpiresAt().IsZero() {
		item.SetExpiresAt(time.Time{})
	}

	content, err := utils.ReadFile(fs.keyPath(key))
	if err != nil {
		return nil, err
	}

	if err := item.FromString(string(content)); err != nil {
		return nil, kit.AppError{
			Code:     "cacheitem_fromstring_error",
			Message:  err.Error(),
			Errors:   []error{err},
			Internal: true,
		}
	}

	// Return nil if item is expired.
	if item.IsExpired() {
		return nil, nil
	}

	return item, nil
}

func (fs *Fs) GetString(key string) (string, kit.Error) {
	item, err := fs.Get(key)
	if err != nil {
		return "", err
	}
	if item == nil {
		return "", nil
	}

	value, err := item.ToString()
	if err != nil {
		return "", err
	}
	return value, nil
}

// Delete item from the cache.
func (fs *Fs) Delete(keys ...string) kit.Error {
	for _, rawKey := range keys {
		key := fs.key(rawKey)
		if key == "" {
			return kit.AppError{Code: "empty_key"}
		}

		exists, err := utils.FileExists(fs.keyPath(key))
		if err != nil {
			return err
		}
		if exists {
			if err := os.Remove(fs.keyPath(key)); err != nil {
				return kit.AppError{
					Code:     "file_delete_error",
					Message:  err.Error(),
					Internal: true,
				}
			}
		}

		exists, err = utils.FileExists(fs.keyMetaPath(key))
		if err != nil {
			return err
		}
		if exists {
			if err := os.Remove(fs.keyMetaPath(key)); err != nil {
				return kit.AppError{
					Code:     "file_delete_error",
					Message:  err.Error(),
					Internal: true,
				}
			}
		}
	}

	return nil
}

func (fs *Fs) Keys() ([]string, kit.Error) {
	files, err := utils.ListFiles(fs.path)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0)
	for _, file := range files {
		if !strings.HasSuffix(file, ".meta") {
			keys = append(keys, file)
		}
	}

	return keys, nil
}

func (fs *Fs) KeysByTags(tags ...string) ([]string, kit.Error) {
	allKeys, err := fs.Keys()
	if err != nil {
		return nil, err
	}

	matchedKeys := make([]string, 0)
	for _, key := range allKeys {
		item, err := fs.Get(key)
		if err != nil {
			return nil, err
		}

		for _, tag := range tags {
			if utils.StrIn(item.GetTags(), tag) {
				matchedKeys = append(matchedKeys, key)
			}
		}
	}

	return matchedKeys, nil
}

// Clear all items from the cache.
func (fs *Fs) Clear() kit.Error {
	keys, err := fs.Keys()
	if err != nil {
		return err
	}

	return fs.Delete(keys...)
}

// Clean up all expired entries.
func (fs *Fs) Cleanup() kit.Error {
	keys, err := fs.Keys()
	if err != nil {
		return err
	}

	for _, key := range keys {
		item, err := fs.Get(key)
		if err != nil {
			return err
		}

		if item.GetExpiresAt().IsZero() {
			continue
		}

		if item.IsExpired() {
			if err := fs.Delete(key); err != nil {
				return err
			}
		}
	}

	return nil
}

// Clear all items with the specified tags.
func (fs *Fs) ClearTag(tag string) kit.Error {
	keys, err := fs.KeysByTags(tag)
	if err != nil {
		return err
	}

	return fs.Delete(keys...)
}
