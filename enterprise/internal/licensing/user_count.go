package licensing

import (
	"context"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/sourcegraph/sourcegraph/internal/redispool"

	"github.com/inconshreveable/log15"
)

var (
	pool      = redispool.Store
	keyPrefix = "license_user_count:"

	started bool
)

// A UsersStore captures the necessary methods for the licensing
// package to query Sourcegraph users. It allows decoupling this package
// from the OSS db package.
type UsersStore interface {
	// Count returns the total count of active Sourcegraph users.
	Count(context.Context) (int, error)
}

// setMaxUsers sets the max users associated with a license key if the new max count is greater than the previous max.
func setMaxUsers(key string, count int) error {
	c := pool.Get()
	defer c.Close()

	lastMax, _, err := getMaxUsers(c, key)
	if err != nil {
		return err
	}

	if count > lastMax {
		err = c.Send("HSET", maxUsersKey(), key, count)
		if err != nil {
			return err
		}
		return c.Send("HSET", maxUsersTimeKey(), key, time.Now().Format("2006-01-02 15:04:05 UTC"))
	}
	return nil
}

// GetMaxUsers gets the max users associated with a license key.
func GetMaxUsers(signature string) (int, string, error) {
	c := pool.Get()
	defer c.Close()

	if signature == "" {
		// No license key is in use.
		return 0, "", nil
	}

	return getMaxUsers(c, signature)
}

func getMaxUsers(c redis.Conn, key string) (int, string, error) {
	lastMax, err := redis.String(c.Do("HGET", maxUsersKey(), key))
	if err != nil && err != redis.ErrNil {
		return 0, "", err
	}
	lastMaxInt := 0
	if lastMax != "" {
		lastMaxInt, err = strconv.Atoi(lastMax)
		if err != nil {
			return 0, "", err
		}
	}
	lastMaxDate, err := redis.String(c.Do("HGET", maxUsersTimeKey(), key))
	if err != nil && err != redis.ErrNil {
		return 0, "", err
	}
	return lastMaxInt, lastMaxDate, nil
}

// checkMaxUsers runs periodically, and if a license key is in use, updates the
// record of maximum count of user accounts in use.
func checkMaxUsers(ctx context.Context, s UsersStore, signature string) error {
	if signature == "" {
		// No license key is in use.
		return nil
	}

	count, err := s.Count(ctx)
	if err != nil {
		log15.Error("licensing.checkMaxUsers: error getting user count", "error", err)
		return err
	}
	err = setMaxUsers(signature, int(count))
	if err != nil {
		log15.Error("licensing.checkMaxUsers: error setting new max users", "error", err)
		return err
	}
	return nil
}

func maxUsersKey() string {
	return keyPrefix + "max"
}

func maxUsersTimeKey() string {
	return keyPrefix + "max_time"
}

// ActualUserCount returns the actual max number of users that have had accounts on the
// Sourcegraph instance, under the current license.
func ActualUserCount(ctx context.Context) (int32, error) {
	_, signature, err := GetConfiguredProductLicenseInfoWithSignature()
	if err != nil || signature == "" {
		return 0, err
	}

	count, _, err := GetMaxUsers(signature)
	return int32(count), err
}

// ActualUserCountDate returns the timestamp when the actual max number of users that have
// had accounts on the Sourcegraph instance, under the current license, was reached.
func ActualUserCountDate(ctx context.Context) (string, error) {
	_, signature, err := GetConfiguredProductLicenseInfoWithSignature()
	if err != nil || signature == "" {
		return "", err
	}

	_, date, err := GetMaxUsers(signature)
	return date, err
}

// StartMaxUserCount starts checking for a new count of max user accounts periodically.
func StartMaxUserCount(s UsersStore) {
	if started {
		panic("already started")
	}
	started = true

	ctx := context.Background()
	const delay = 360 * time.Minute
	for {
		_, signature, err := GetConfiguredProductLicenseInfoWithSignature()
		if err != nil {
			log15.Error("licensing.startMaxUserCount: error getting configured license info")
		} else if signature != "" {
			ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
			_ = checkMaxUsers(ctx, s, signature) // updates global state on its own, can safely ignore return value
			cancel()
		}
		time.Sleep(delay)
	}
}

// NoLicenseMaximumAllowedUserCount is the maximum number of user accounts that may exist when
// running without a license. Exceeding this number of user accounts requires a license.
const NoLicenseMaximumAllowedUserCount int32 = 10

// NoLicenseWarningUserCount is the number of user accounts when all users are shown a warning (when running
// without a license).
const NoLicenseWarningUserCount int32 = 10
