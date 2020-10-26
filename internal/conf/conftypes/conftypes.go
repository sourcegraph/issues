package conftypes

import "reflect"

// ServiceConnections represents configuration about how the deployment
// internally connects to services. These are settings that need to be
// propagated from the frontend to other services, so that the frontend
// can be the source of truth for all configuration.
type ServiceConnections struct {
	// GitServers is the addresses of gitserver instances that should be
	// talked to.
	GitServers []string `json:"gitServers"`

	// PostgresDSN is the PostgreSQL DB data source name.
	// eg: "postgres://sg@pgsql/sourcegraph?sslmode=false"
	PostgresDSN string `json:"postgresDSN"`

	// CodeIntelPostgresDSN is the PostgreSQL DB data source name for the
	// code intel database.
	// eg: "postgres://sg@pgsql/sourcegraph_codeintel?sslmode=false"
	CodeIntelPostgresDSN string `json:"codeIntelPostgresDSN"`
}

// RawUnified is the unparsed variant of conf.Unified.
type RawUnified struct {
	Site               string
	ServiceConnections ServiceConnections
}

// Equal tells if the two configurations are equal or not.
func (r RawUnified) Equal(other RawUnified) bool {
	return r.Site == other.Site && reflect.DeepEqual(r.ServiceConnections, other.ServiceConnections)
}
