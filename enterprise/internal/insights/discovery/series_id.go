package discovery

import (
	"crypto/sha256"
	"fmt"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/insights"

	"github.com/sourcegraph/sourcegraph/schema"
)

// EncodeSeriesID hashes the hashes the input series to return a string which uniquely identifies
// the data series being described. It is possible the same series is described in multiple user's
// settings, e.g. if multiple users declare an insight with the same search query - in which case
// we have an opportunity to deduplicate them.
//
// Note that since the series ID hash is stored in the database, it must remain stable or else past
// data will not be queryable.
func EncodeSeriesID(series *schema.InsightSeries) (string, error) {
	switch {
	case series.Search != "":
		return fmt.Sprintf("s:%s", sha256String(series.Search)), nil
	case series.Webhook != "":
		return fmt.Sprintf("w:%s", sha256String(series.Webhook)), nil
	default:
		return "", errors.Errorf("invalid series %+v", series)
	}
}

func Encode(series insights.TimeSeries) string {
	return fmt.Sprintf("s:%s", sha256String(series.Query))
}

func sha256String(s string) string {
	return fmt.Sprintf("%X", sha256.Sum256([]byte(s)))
}
