package rcache

import (
	"fmt"
	"os"
	"time"
	"unicode/utf8"

	"github.com/gomodule/redigo/redis"
	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/redispool"
)

// dataVersion is used for releases that change type structure for
// data that may already be cached. Increasing this number will
// change the key prefix that is used for all hash keys,
// effectively resetting the cache at the same time the new code
// is deployed.
const dataVersion = "v2"
const dataVersionToDelete = "v1"

// DeleteOldCacheData deletes the rcache data in the given Redis instance
// that's prefixed with dataVersionToDelete
func DeleteOldCacheData(c redis.Conn) error {
	return deleteKeysWithPrefix(c, dataVersionToDelete)
}

// Cache implements httpcache.Cache
type Cache struct {
	keyPrefix  string
	ttlSeconds int
}

// New creates a redis backed Cache
func New(keyPrefix string) *Cache {
	return &Cache{
		keyPrefix: keyPrefix,
	}
}

// NewWithTTL creates a redis backed Cache which expires values after
// ttlSeconds.
func NewWithTTL(keyPrefix string, ttlSeconds int) *Cache {
	return &Cache{
		keyPrefix:  keyPrefix,
		ttlSeconds: ttlSeconds,
	}
}

func (r *Cache) GetMulti(keys ...string) [][]byte {
	c := pool.Get()
	defer c.Close()

	if len(keys) == 0 {
		return nil
	}
	rkeys := make([]interface{}, len(keys))
	for i, key := range keys {
		rkeys[i] = r.rkeyPrefix() + key
	}

	vals, err := redis.Values(c.Do("MGET", rkeys...))
	if err != nil && err != redis.ErrNil {
		log15.Warn("failed to execute redis command", "cmd", "MGET", "error", err)
	}

	strVals := make([][]byte, len(vals))
	for i, val := range vals {
		// MGET returns nil as not found.
		if val == nil {
			continue
		}

		b, err := redis.Bytes(val, nil)
		if err != nil {
			log15.Warn("failed to parse bytes from Redis value", "value", val)
			continue
		}
		strVals[i] = b
	}
	return strVals
}

func (r *Cache) SetMulti(keyvals ...[2]string) {
	c := pool.Get()
	defer c.Close()

	if len(keyvals) == 0 {
		return
	}

	for _, kv := range keyvals {
		k, v := kv[0], kv[1]
		if !utf8.Valid([]byte(k)) {
			if conf.IsDev(conf.DeployType()) {
				panic(fmt.Sprintf("rcache: keys must be valid utf8 %v", []byte(k)))
			} else {
				log15.Error("rcache: keys must be valid utf8", "key", []byte(k))
			}
			continue
		}
		if r.ttlSeconds == 0 {
			if err := c.Send("SET", r.rkeyPrefix()+k, []byte(v)); err != nil {
				log15.Warn("failed to write redis command to client output buffer", "cmd", "SET", "error", err)
			}
		} else {
			if err := c.Send("SETEX", r.rkeyPrefix()+k, r.ttlSeconds, []byte(v)); err != nil {
				log15.Warn("failed to write redis command to client output buffer", "cmd", "SETEX", "error", err)
			}
		}
	}
	if err := c.Flush(); err != nil {
		log15.Warn("failed to flush Redis client", "error", err)
	}
}

// Get implements httpcache.Cache.Get
func (r *Cache) Get(key string) ([]byte, bool) {
	c := pool.Get()
	defer c.Close()

	b, err := redis.Bytes(c.Do("GET", r.rkeyPrefix()+key))
	if err != nil && err != redis.ErrNil {
		log15.Warn("failed to execute redis command", "cmd", "GET", "error", err)
	}

	return b, err == nil
}

// Set implements httpcache.Cache.Set
func (r *Cache) Set(key string, b []byte) {
	c := pool.Get()
	defer c.Close()

	if !utf8.Valid([]byte(key)) {
		if conf.IsDev(conf.DeployType()) {
			panic(fmt.Sprintf("rcache: keys must be valid utf8 %v", []byte(key)))
		} else {
			log15.Error("rcache: keys must be valid utf8", "key", []byte(key))
		}
	}

	if r.ttlSeconds == 0 {
		_, err := c.Do("SET", r.rkeyPrefix()+key, b)
		if err != nil {
			log15.Warn("failed to execute redis command", "cmd", "SET", "error", err)
		}
	} else {
		_, err := c.Do("SETEX", r.rkeyPrefix()+key, r.ttlSeconds, b)
		if err != nil {
			log15.Warn("failed to execute redis command", "cmd", "SETEX", "error", err)
		}
	}
}

// Delete implements httpcache.Cache.Delete
func (r *Cache) Delete(key string) {
	c := pool.Get()
	defer c.Close()

	_, err := c.Do("DEL", r.rkeyPrefix()+key)
	if err != nil {
		log15.Warn("failed to execute redis command", "cmd", "DEL", "error", err)
	}
}

// rkeyPrefix generates the actual key prefix we use on redis.
func (r *Cache) rkeyPrefix() string {
	return fmt.Sprintf("%s:%s:", globalPrefix, r.keyPrefix)
}

// TB is a subset of testing.TB
type TB interface {
	Name() string
	Skip(args ...interface{})
	Helper()
}

// SetupForTest adjusts the globalPrefix and clears it out. You will have
// conflicts if you do `t.Parallel()`
func SetupForTest(t TB) {
	t.Helper()

	pool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "127.0.0.1:6379")
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	globalPrefix = "__test__" + t.Name()
	c := pool.Get()
	defer c.Close()

	// If we are not on CI, skip the test if our redis connection fails.
	if os.Getenv("CI") == "" {
		_, err := c.Do("PING")
		if err != nil {
			t.Skip("could not connect to redis", err)
		}
	}

	err := deleteKeysWithPrefix(c, globalPrefix)
	if err != nil {
		log15.Error("Could not clear test prefix", "name", t.Name(), "globalPrefix", globalPrefix, "error", err)
	}
}

// The number of keys to delete per batch.
// The maximum number of keys that can be unpacked
// is determined by the Lua config LUAI_MAXCSTACK
// which is 8000 by default.
// See https://www.lua.org/source/5.1/luaconf.h.html
var deleteBatchSize = 5000

func deleteKeysWithPrefix(c redis.Conn, prefix string) error {
	const script = `
redis.replicate_commands()
local cursor = '0'
local prefix = ARGV[1]
local batchSize = ARGV[2]
local result = ''
repeat
	local keys = redis.call('SCAN', cursor, 'MATCH', prefix, 'COUNT', batchSize)
	if #keys[2] > 0
	then
		result = redis.call('DEL', unpack(keys[2]))
	end

	cursor = keys[1]
until cursor == '0'
return result
`

	_, err := c.Do("EVAL", script, 0, prefix+":*", deleteBatchSize)
	return err
}

var (
	pool         = redispool.Cache
	globalPrefix = dataVersion
)
