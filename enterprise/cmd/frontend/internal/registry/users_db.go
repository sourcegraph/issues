package registry

import (
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/redispool"
)

var (
	pool      = redispool.Store
	keyPrefix = "extensions_user_count:"

	started bool
)

func extensionUsersKey(id string) string {
	return keyPrefix + "extension:" + id
}

func downloadedExtensionsKey() string {
	return keyPrefix + "downloaded_extensions"
}

func IncrementExtensionUserCount(extensionID string, anonUserID string) error {
	c := pool.Get()
	defer c.Close()

	if err := c.Send("SADD", extensionUsersKey(extensionID), anonUserID); err != nil {
		return err
	}
	if err := c.Send("SADD", downloadedExtensionsKey(), extensionID); err != nil {
		return err
	}

	return nil
}

func ComputeExtensionsUserCounts() (map[string][]string, error) {
	c := pool.Get()
	defer c.Close()

	values, err := redis.Values(c.Do("SMEMBERS", downloadedExtensionsKey()))
	if err == redis.ErrNil {
		err = nil
	}
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, nil
	}

	extensionIDs := make([]string, len(values))

	for i, id := range values {
		extensionID := string(id.([]byte))
		extensionIDs[i] = extensionID
	}

	uidsByExtensionId := make(map[string][]string)

	for _, id := range extensionIDs {
		key := extensionUsersKey(id)
		values, err := redis.Values(c.Do("SMEMBERS", key))
		if err == redis.ErrNil {
			err = nil
		}
		if err != nil {
			return nil, err
		}

		uids := make([]string, len(values))
		for i, id := range values {
			uid := string(id.([]byte))
			uids[i] = uid
		}
		uidsByExtensionId[id] = uids

		_, err = c.Do("DEL", key)
		if err != nil {
			return nil, err
		}
	}

	_, err = c.Do("DEL", downloadedExtensionsKey())
	if err != nil {
		return nil, err
	}

	return uidsByExtensionId, nil
}

func startExtensionsUserCount() {
	if started {
		panic("already started")
	}
	started = true

	const delay = 5 * time.Second
	for {
		usersByExtension, err := ComputeExtensionsUserCounts()
		// add to db

		time.Sleep(delay)
	}
}

func init() {
	goroutine.Go(func() {
		startExtensionsUserCount()
	})
}
