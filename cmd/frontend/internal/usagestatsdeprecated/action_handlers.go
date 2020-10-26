package usagestatsdeprecated

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

var (
	searchOccurred   = false
	findRefsOccurred = false
)

// LogActivity logs any user activity (page view, integration usage, etc) to their "last active" time, and
// adds their unique ID to the set of active users
func LogActivity(isAuthenticated bool, userID int32, userCookieID, event string) error {
	// Setup our GC of active key goroutine
	gcOnce.Do(func() {
		go gc()
	})

	c := pool.Get()
	defer c.Close()

	uniqueID := userCookieID

	// If the user is authenticated, set uniqueID to their user ID, and store their "last active time" in the
	// appropriate user ID-keyed cache.
	if isAuthenticated {
		userIDStr := strconv.Itoa(int(userID))
		uniqueID = userIDStr
		key := keyPrefix + uniqueID

		// Set the user's last active time
		now := timeNow().UTC()
		if err := c.Send("HSET", key, fLastActive, now.Format(time.RFC3339)); err != nil {
			return err
		}
	}

	if uniqueID == "" {
		log15.Warn("usagestats.LogActivity: no user ID provided")
		return nil
	}

	// Regardless of authenicatation status, add the user's unique ID to the set of active users.
	if err := c.Send("SADD", usersActiveKeyFromDaysAgo(0), uniqueID); err != nil {
		return err
	}

	if handlers, ok := eventHandlers[event]; ok {
		for _, handler := range handlers {
			err := handler(userID, event, isAuthenticated)
			if err != nil {
				return err
			}
		}
		return nil
	}

	return fmt.Errorf("unknown user event %s", event)
}

// Custom event handlers
type eventHandler = func(userID int32, event string, isAuthenticated bool) error

var eventHandlers = map[string][]eventHandler{
	"SEARCHQUERY":              {logSiteSearchOccurred, logSearchQuery},
	"PAGEVIEW":                 {logPageView},
	"CODEINTEL":                {logCodeIntelAction},
	"CODEINTELREFS":            {logSiteFindRefsOccurred, logCodeIntelRefsAction},
	"CODEINTELINTEGRATION":     {logCodeHostIntegrationUsage, logCodeIntelAction},
	"CODEINTELINTEGRATIONREFS": {logSiteFindRefsOccurred, logCodeHostIntegrationUsage, logCodeIntelRefsAction},

	"STAGEMANAGE":    {logStageEvent},
	"STAGEPLAN":      {logStageEvent},
	"STAGECODE":      {logStageEvent},
	"STAGEREVIEW":    {logStageEvent},
	"STAGEVERIFY":    {logStageEvent},
	"STAGEPACKAGE":   {logStageEvent},
	"STAGEDEPLOY":    {logStageEvent},
	"STAGECONFIGURE": {logStageEvent},
	"STAGEMONITOR":   {logStageEvent},
	"STAGESECURE":    {logStageEvent},
	"STAGEAUTOMATE":  {logStageEvent},
}

// logSiteSearchOccurred records that a search has occurred on the Sourcegraph instance.
var logSiteSearchOccurred = func(_ int32, _ string, _ bool) error {
	if searchOccurred {
		return nil
	}
	key := keyPrefix + fSearchOccurred
	c := pool.Get()
	defer c.Close()
	searchOccurred = true
	return c.Send("SET", key, "true")
}

// logSiteFindRefsOccurred records that a search has occurred on the Sourcegraph instance.
var logSiteFindRefsOccurred = func(_ int32, _ string, _ bool) error {
	if findRefsOccurred {
		return nil
	}
	key := keyPrefix + fFindRefsOccurred
	c := pool.Get()
	defer c.Close()
	findRefsOccurred = true
	return c.Send("SET", key, "true")
}

// logSearchQuery increments a user's search query count.
var logSearchQuery = func(userID int32, _ string, isAuthenticated bool) error {
	return incrementUserCounter(userID, isAuthenticated, fSearchQueries)
}

// logPageView increments a user's pageview count.
var logPageView = func(userID int32, _ string, isAuthenticated bool) error {
	return incrementUserCounter(userID, isAuthenticated, fPageViews)
}

// logCodeIntelAction increments a user's code intelligence usage count.
var logCodeIntelAction = func(userID int32, _ string, isAuthenticated bool) error {
	return incrementUserCounter(userID, isAuthenticated, fCodeIntelActions)
}

// logCodeIntelRefsAction increments a user's code intelligence usage count.
// and their find refs action count.
var logCodeIntelRefsAction = func(userID int32, _ string, isAuthenticated bool) error {
	if err := incrementUserCounter(userID, isAuthenticated, fCodeIntelActions); err != nil {
		return err
	}
	return incrementUserCounter(userID, isAuthenticated, fFindRefsActions)
}

// logCodeHostIntegrationUsage logs the last time a user was active on a code host integration.
var logCodeHostIntegrationUsage = func(userID int32, _ string, isAuthenticated bool) error {
	if !isAuthenticated {
		return nil
	}
	key := keyPrefix + strconv.Itoa(int(userID))
	c := pool.Get()
	defer c.Close()

	now := timeNow().UTC()
	return c.Send("HSET", key, fLastActiveCodeHostIntegration, now.Format(time.RFC3339))
}

// logStageEvent logs the last time a user did an action from a specific stage.
var logStageEvent = func(userID int32, event string, isAuthenticated bool) error {
	if !isAuthenticated {
		return nil
	}
	key := keyPrefix + strconv.Itoa(int(userID))
	c := pool.Get()
	defer c.Close()

	now := timeNow().UTC()
	return c.Send("HSET", key, keyFromStage(event), now.Format(time.RFC3339))
}

// LogEvent logs users events.
func LogEvent(ctx context.Context, name, url string, userID int32, userCookieID, source string, argument json.RawMessage) error {
	info := &db.Event{
		Name:            name,
		URL:             url,
		UserID:          uint32(userID),
		AnonymousUserID: userCookieID,
		Source:          source,
		Argument:        argument,
	}
	return db.EventLogs.Insert(ctx, info)
}
