package fs

import (
	"encoding/json"
	"os"
	"path"
	"strings"
	"time"

	"github.com/theduke/go-apperror"

	kit "github.com/app-kit/go-appkit"
	. "github.com/app-kit/go-appkit/caches"
	"github.com/app-kit/go-appkit/utils"
)

type Fs struct {
	name string
	path string
}

// Ensure redis implements the Cache interface.
var _ kit.Cache = (*Fs)(nil)

func fsErr(err error) apperror.Error {
	return apperror.Wrap(err, "fs_error")
}

func New(path string) (*Fs, apperror.Error) {
	if path == "" {
		return nil, apperror.New("empty_path")
	}

	if err := os.MkdirAll(path, 0777); err != nil {
		return nil, apperror.Wrap(err, "root_path_unwritable")
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
func (fs *Fs) Set(item kit.CacheItem) apperror.Error {
	key := fs.key(item.GetKey())
	if key == "" {
		return apperror.New("empty_key")
	}

	if item.IsExpired() {
		return apperror.New("item_expired")
	}

	value, err := item.ToString()
	if err != nil {
		return apperror.Wrap(err, "cacheitem_tostring_error")
	}
	if value == "" {
		return apperror.New("empty_value")
	}

	// Marshal metadata.
	tmpVal := item.GetValue()
	item.SetValue(nil)

	js, err2 := json.Marshal(item)
	if err2 != nil {
		return apperror.Wrap(err2, "json_marshal_error")
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

func (fs *Fs) SetString(key string, value string, expiresAt *time.Time, tags []string) apperror.Error {
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
func (fs *Fs) Get(key string, items ...kit.CacheItem) (kit.CacheItem, apperror.Error) {
	var item kit.CacheItem = &StrItem{}
	if items != nil {
		if len(items) != 1 {
			return nil, &apperror.Err{
				Code:    "invalid_item",
				Message: "You must specify one item only",
			}
		}
		item = items[0]
	}

	key = fs.key(key)
	if key == "" {
		return nil, apperror.New("empty_key")
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
		return nil, apperror.Wrap(err, "metadata_unmarshal_error")
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
		return nil, apperror.Wrap(err, "cacheitem_fromstring_error")
	}

	// Return nil if item is expired.
	if item.IsExpired() {
		return nil, nil
	}

	return item, nil
}

func (fs *Fs) GetString(key string) (string, apperror.Error) {
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
func (fs *Fs) Delete(keys ...string) apperror.Error {
	for _, rawKey := range keys {
		key := fs.key(rawKey)
		if key == "" {
			return apperror.New("empty_key")
		}

		exists, err := utils.FileExists(fs.keyPath(key))
		if err != nil {
			return err
		}
		if exists {
			if err := os.Remove(fs.keyPath(key)); err != nil {
				return apperror.Wrap(err, "file_delete_error")
			}
		}

		exists, err = utils.FileExists(fs.keyMetaPath(key))
		if err != nil {
			return err
		}
		if exists {
			if err := os.Remove(fs.keyMetaPath(key)); err != nil {
				return apperror.Wrap(err, "file_delete_error")
			}
		}
	}

	return nil
}

func (fs *Fs) Keys() ([]string, apperror.Error) {
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

func (fs *Fs) KeysByTags(tags ...string) ([]string, apperror.Error) {
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
func (fs *Fs) Clear() apperror.Error {
	keys, err := fs.Keys()
	if err != nil {
		return err
	}

	return fs.Delete(keys...)
}

// Clean up all expired entries.
func (fs *Fs) Cleanup() apperror.Error {
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
func (fs *Fs) ClearTag(tag string) apperror.Error {
	keys, err := fs.KeysByTags(tag)
	if err != nil {
		return err
	}

	return fs.Delete(keys...)
}
