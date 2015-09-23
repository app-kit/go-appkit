package redis

import (
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"

	kit "github.com/theduke/go-appkit"
	. "github.com/theduke/go-appkit/caches"
	"github.com/theduke/go-appkit/utils"
)

type Config struct {
	// Redis network address, like localhost:6379
	Address string

	DialOptions []redis.DialOption

	// Prefix under which all keys will be stored.
	Prefix string

	// Maximum number of idle connections in the connection pool.
	MaxIdleConnections int

	// Timeout for idle connections in the pool in seconds.
	IdleConnectionTimeout int

	// Number of maximum active connections.
	MaxActiveConnections int

	// If true, wait until a connection becomes available instead of returning an
	// error.
	WaitForConnection bool

	// Password for redis authentication.
	Password string
}

type Redis struct {
	name   string
	config Config
	pool   *redis.Pool
}

// Ensure redis implements the Cache interface.
var _ kit.Cache = (*Redis)(nil)

func redisErr(err error) Error {
	return kit.AppError{
		Code:    "redis_error",
		Message: err.Error(),
		Errors:  []error{err},
	}
}

func New(conf Config) (*Redis, Error) {
	if conf.Address == "" {
		conf.Address = "localhost:6379"
	}
	if conf.Prefix == "" {
		conf.Prefix = "cache"
	}
	if conf.MaxIdleConnections == 0 {
		conf.MaxIdleConnections = 3
	}
	if conf.IdleConnectionTimeout == 0 {
		conf.IdleConnectionTimeout = 120
	}

	r := &Redis{
		name:   "redis",
		config: conf,
	}

	r.buildPool()

	// Test connection.

	return r, nil
}

func (r *Redis) Name() string {
	return r.name
}

func (r *Redis) buildPool() {
	r.pool = &redis.Pool{
		MaxIdle:     r.config.MaxIdleConnections,
		IdleTimeout: time.Duration(r.config.IdleConnectionTimeout) * time.Second,
		MaxActive:   r.config.MaxActiveConnections,
		Wait:        r.config.WaitForConnection,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", r.config.Address, r.config.DialOptions...)
			if err != nil {
				return nil, err
			}
			if r.config.Password != "" {
				if _, err := c.Do("AUTH", r.config.Password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func (r *Redis) key(key string) string {
	return r.config.Prefix + ":" + key
}

func (r *Redis) keys(keys []string) []string {
	rawKeys := make([]string, 0)
	for _, key := range keys {
		rawKeys = append(rawKeys, r.config.Prefix+":"+key)
	}
	return rawKeys
}

func (r *Redis) tagKey(key string) string {
	return key + ":tags"
}

func (r *Redis) tagKeys(keys []string) []string {
	rawKeys := make([]string, 0)
	for _, key := range keys {
		rawKeys = append(rawKeys, key+":tags")
	}
	return rawKeys
}

func (r *Redis) cleanKey(rawKey string) string {
	return strings.Replace(rawKey, r.config.Prefix+":", "", 1)
}

func (r *Redis) cleanKeys(rawKeys []string) []string {
	keys := make([]string, 0)
	for _, key := range rawKeys {
		keys = append(keys, strings.Replace(key, r.config.Prefix+":", "", 1))
	}

	return keys
}

// Save a new item into the cache.
func (r *Redis) Set(item kit.CacheItem) Error {
	key := item.GetKey()
	if key == "" {
		return kit.AppError{Code: "empty_key"}
	}
	key = r.key(key)

	if item.IsExpired() {
		return kit.AppError{Code: "item_expired"}
	}

	expireSeconds := 0
	if expires := item.GetExpiresAt(); !expires.IsZero() {
		seconds := expires.Sub(time.Now()).Seconds()
		if seconds > 0 && seconds < 1 {
			return kit.AppError{Code: "item_expired"}
		}
		expireSeconds = int(seconds)
	}

	conn := r.pool.Get()
	defer conn.Close()

	value, err := item.ToString()
	if err != nil {
		return kit.AppError{
			Code:    "cacheitem_tostring_error",
			Message: err.Error(),
			Errors:  []error{err},
		}
	}
	if value == "" {
		return kit.AppError{Code: "empty_value"}
	}

	conn.Send("SET", key, value)
	if expireSeconds > 0 {
		conn.Send("EXPIRE", key, expireSeconds)
	}

	if tags := item.GetTags(); tags != nil {
		tagKey := r.tagKey(key)
		conn.Send("SET", tagKey, strings.Join(tags, ";"))
		if expireSeconds > 0 {
			conn.Send("EXPIRE", tagKey, expireSeconds)
		}
	}

	if err := conn.Flush(); err != nil {
		return redisErr(err)
	}

	return nil
}

func (r *Redis) SetString(key string, value string, expiresAt *time.Time, tags []string) Error {
	item := &StrItem{
		Key:   key,
		Value: value,
		Tags:  tags,
	}
	if expiresAt != nil {
		item.ExpiresAt = *expiresAt
	}

	return r.Set(item)
}

// Retrieve a cache item from the cache.
func (r *Redis) Get(key string, items ...kit.CacheItem) (kit.CacheItem, Error) {
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

	if key == "" {
		return nil, kit.AppError{Code: "empty_key"}
	}
	item.SetKey(key)
	key = r.key(key)

	conn := r.pool.Get()
	defer conn.Close()

	result, err := redis.Strings(conn.Do("MGET", key, r.tagKey(key)))
	if err != nil {
		return nil, redisErr(err)
	}

	// Retrieve the value.
	if result[0] == "" {
		return nil, nil
	}
	if err := item.FromString(result[0]); err != nil {
		return nil, kit.AppError{
			Code:    "cacheitem_fromstring_error",
			Message: err.Error(),
			Errors:  []error{err},
		}
	}

	// Parse and set tags.
	if result[1] != "" {
		item.SetTags(strings.Split(result[1], ";"))
	}

	// Now get the ttl.
	ttl, err := redis.Int(conn.Do("TTL", key))
	if err == nil && ttl > 0 {
		expires := time.Now().Add(time.Duration(ttl) * time.Second)
		item.SetExpiresAt(expires)
	} else if err != nil && err != redis.ErrNil {
		return nil, redisErr(err)
	}

	// Return nil if item is expired.
	if item.IsExpired() {
		return nil, nil
	}

	return item, nil
}

func (r *Redis) GetString(key string) (string, Error) {
	item, err := r.Get(key)
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
func (r *Redis) Delete(keys ...string) Error {
	ifKeys := make([]interface{}, 0)
	for _, key := range keys {
		if key == "" {
			return kit.AppError{Code: "empty_key"}
		}

		key = r.key(key)
		ifKeys = append(ifKeys, interface{}(key))
		ifKeys = append(ifKeys, interface{}(r.tagKey(key)))
	}

	conn := r.pool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", ifKeys...)
	if err != nil {
		return redisErr(err)
	}

	return nil
}

func (r *Redis) getRawKeys(conn redis.Conn) ([]string, Error) {
	rawKeys, err := redis.Strings(conn.Do("KEYS", r.key("")+"*"))
	if err != nil {
		return nil, redisErr(err)
	}

	// Remove tags keys.
	cleanedKeys := make([]string, 0)
	for _, key := range rawKeys {
		if !strings.HasSuffix(key, ":tags") {
			cleanedKeys = append(cleanedKeys, key)
		}
	}

	return cleanedKeys, nil
}

func (r *Redis) Keys() ([]string, Error) {
	conn := r.pool.Get()
	defer conn.Close()

	rawKeys, err := r.getRawKeys(conn)
	if err != nil {
		return nil, err
	}

	return r.cleanKeys(rawKeys), nil
}

func (r *Redis) KeysByTags(matchTags ...string) ([]string, Error) {
	conn := r.pool.Get()
	defer conn.Close()

	keys, err := r.getRawKeys(conn)
	if err != nil {
		return nil, err
	}

	tagKeys := make([]interface{}, 0)
	for _, key := range keys {
		tagKeys = append(tagKeys, interface{}(r.tagKey(key)))
	}

	tags, err2 := redis.Strings(conn.Do("MGET", tagKeys...))
	if err2 != nil {
		return nil, redisErr(err2)
	}

	matchedKeys := make([]string, 0)
	for index, tags := range tags {
		tagList := strings.Split(tags, ";")

		for _, matchTag := range matchTags {
			if utils.StrIn(tagList, matchTag) {
				matchedKeys = append(matchedKeys, keys[index])
			}
		}
	}

	return r.cleanKeys(matchedKeys), nil
}

// Clear all items from the cache.
func (r *Redis) Clear() Error {
	conn := r.pool.Get()
	defer conn.Close()

	keys, err := r.getRawKeys(conn)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	interfaceKeys := make([]interface{}, 0)
	for _, key := range keys {
		interfaceKeys = append(interfaceKeys, interface{}(key))
	}

	if _, err := conn.Do("DEL", interfaceKeys...); err != nil {
		return redisErr(err)
	}

	return nil
}

// Clean up all expired entries.
func (r *Redis) Cleanup() Error {
	// Nothing to do here with redis.
	return nil
}

// Clear all items with the specified tags.
func (r *Redis) ClearTag(tag string) Error {
	conn := r.pool.Get()
	defer conn.Close()

	keys, err := r.KeysByTags(tag)
	if err != nil {
		return err
	}

	keysToDelete := make([]interface{}, 0)
	for _, key := range keys {
		keysToDelete = append(keysToDelete, interface{}(r.key(key)))
	}

	if len(keysToDelete) > 0 {
		if _, err := conn.Do("DEL", keysToDelete...); err != nil {
			return redisErr(err)
		}
	}

	return nil
}
