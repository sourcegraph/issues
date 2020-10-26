package usagestatsdeprecated

import (
	"strconv"
	"time"
)

func keyFromDate(activity string, date time.Time) string {
	return keyPrefix + ":" + activity + ":" + time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

func usersActiveKeyFromDate(date time.Time) string {
	return keyFromDate(fUsersActive, date)
}

func usersActiveKeyFromDaysAgo(daysAgo int) string {
	now := timeNow().UTC()
	return keyFromDate(fUsersActive, now.AddDate(0, 0, -daysAgo))
}

func startOfWeek(weeksAgo int) time.Time {
	if weeksAgo > 0 {
		return startOfWeek(0).AddDate(0, 0, -7*weeksAgo)
	}

	// If weeksAgo == 0, start at timeNow(), and loop back by day until we hit a Sunday
	now := timeNow().UTC()
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for date.Weekday() != time.Sunday {
		date = date.AddDate(0, 0, -1)
	}
	return date
}

func startOfMonth(monthsAgo int) time.Time {
	now := timeNow().UTC()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -monthsAgo, 0)
}

func keys(m map[string]bool) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}

func minIntOrZero(a, b int) int {
	min := b
	if a < b {
		min = a
	}
	if min < 0 {
		return 0
	}
	return min
}

func incrementUserCounter(userID int32, isAuthenticated bool, counterKey string) error {
	if !isAuthenticated {
		return nil
	}

	key := keyPrefix + strconv.Itoa(int(userID))
	c := pool.Get()
	defer c.Close()

	return c.Send("HINCRBY", key, counterKey, 1)
}

func keyFromStage(stage string) string {
	return "user-last-active-" + stage
}
