package usagestats

import (
	"context"
	"sync/atomic"

	"github.com/gomodule/redigo/redis"
	"github.com/sourcegraph/sourcegraph/internal/redispool"
)

const (
	keyPrefix         = "user_activity:"
	fSearchOccurred   = "searchoccurred"
	fFindRefsOccurred = "findrefsoccurred"
)

var (
	pool             = redispool.Store
	searchOccurred   int32
	findRefsOccurred int32
)

// logSiteSearchOccurred records that a search has occurred on the Sourcegraph instance.
func logSiteSearchOccurred() error {
	if !atomic.CompareAndSwapInt32(&searchOccurred, 0, 1) {
		return nil
	}
	key := keyPrefix + fSearchOccurred
	c := pool.Get()
	defer c.Close()
	return c.Send("SET", key, "true")
}

// logSiteFindRefsOccurred records that a search has occurred on the Sourcegraph instance.
func logSiteFindRefsOccurred() error {
	if !atomic.CompareAndSwapInt32(&findRefsOccurred, 0, 1) {
		return nil
	}
	key := keyPrefix + fFindRefsOccurred
	c := pool.Get()
	defer c.Close()
	return c.Send("SET", key, "true")
}

// HasSearchOccurred indicates whether a search has ever occurred on this instance.
func HasSearchOccurred(ctx context.Context) (bool, error) {
	c, err := pool.GetContext(ctx)
	if err != nil {
		return false, err
	}
	defer c.Close()
	s, err := redis.Bool(c.Do("GET", keyPrefix+fSearchOccurred))
	if err != nil && err != redis.ErrNil {
		return s, err
	}
	return s, nil
}

// HasFindRefsOccurred indicates whether a find-refs has ever occurred on this instance.
func HasFindRefsOccurred(ctx context.Context) (bool, error) {
	c, err := pool.GetContext(ctx)
	if err != nil {
		return false, err
	}
	defer c.Close()
	r, err := redis.Bool(c.Do("GET", keyPrefix+fFindRefsOccurred))
	if err != nil && err != redis.ErrNil {
		return r, err
	}
	return r, nil
}
