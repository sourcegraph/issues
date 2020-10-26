package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"
	"github.com/mxk/go-flowrate/flowrate"
	"github.com/opentracing/opentracing-go/ext"
	pkgerrors "github.com/pkg/errors"
	"github.com/sourcegraph/codeintelutils"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-bundle-manager/internal/paths"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/database"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence"
	postgresreader "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/postgres"
	sqlitereader "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/sqlite"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
)

const DefaultMonikerResultPageSize = 100
const DefaultDiagnosticResultPageSize = 100

func (s *Server) setupRoutes(router *mux.Router) {
	router.Path("/uploads/{id:[0-9]+}").Methods("GET").HandlerFunc(s.handleGetUpload)
	router.Path("/uploads/{id:[0-9]+}").Methods("POST").HandlerFunc(s.handlePostUpload)
	router.Path("/uploads/{id:[0-9]+}/{index:[0-9]+}").Methods("POST").HandlerFunc(s.handlePostUploadPart)
	router.Path("/uploads/{id:[0-9]+}/stitch").Methods("POST").HandlerFunc(s.handlePostUploadStitch)
	router.Path("/uploads/{id:[0-9]+}").Methods("DELETE").HandlerFunc(s.handleDeleteUpload)
	router.Path("/dbs/{id:[0-9]+}/{index:[0-9]+}").Methods("POST").HandlerFunc(s.handlePostDatabasePart)
	router.Path("/dbs/{id:[0-9]+}/stitch").Methods("POST").HandlerFunc(s.handlePostDatabaseStitch)
	router.Path("/dbs/{id:[0-9]+}/exists").Methods("GET").HandlerFunc(s.handleExists)
	router.Path("/dbs/{id:[0-9]+}/ranges").Methods("GET").HandlerFunc(s.handleRanges)
	router.Path("/dbs/{id:[0-9]+}/definitions").Methods("GET").HandlerFunc(s.handleDefinitions)
	router.Path("/dbs/{id:[0-9]+}/references").Methods("GET").HandlerFunc(s.handleReferences)
	router.Path("/dbs/{id:[0-9]+}/hover").Methods("GET").HandlerFunc(s.handleHover)
	router.Path("/dbs/{id:[0-9]+}/diagnostics").Methods("GET").HandlerFunc(s.handleDiagnostics)
	router.Path("/dbs/{id:[0-9]+}/monikersByPosition").Methods("GET").HandlerFunc(s.handleMonikersByPosition)
	router.Path("/dbs/{id:[0-9]+}/monikerResults").Methods("GET").HandlerFunc(s.handleMonikerResults)
	router.Path("/dbs/{id:[0-9]+}/packageInformation").Methods("GET").HandlerFunc(s.handlePackageInformation)
}

// GET /uploads/{id:[0-9]+}
func (s *Server) handleGetUpload(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open(paths.UploadFilename(s.bundleDir, idFromRequest(r)))
	if err != nil {
		http.Error(w, "Upload not found.", http.StatusNotFound)
		return
	}
	defer file.Close()

	// If there was a transient error while the worker was trying to access the upload
	// file, it retries but indicates the number of bytes that it has received. We can
	// fast-forward the file to this position and only give the worker the data that it
	// still needs. This technique saves us from having to pre-chunk the file as we must
	// do in the reverse direction.
	if _, err := file.Seek(int64(getQueryInt(r, "seek")), io.SeekStart); err != nil {
		log15.Error("Failed to seek upload file", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(limitTransferRate(w), file); err != nil {
		log15.Error("Failed to write payload to client", "err", err)
	}
}

// POST /uploads/{id:[0-9]+}
func (s *Server) handlePostUpload(w http.ResponseWriter, r *http.Request) {
	_ = s.doUpload(w, r, paths.UploadFilename)
}

// POST /uploads/{id:[0-9]+}/{index:[0-9]+}
func (s *Server) handlePostUploadPart(w http.ResponseWriter, r *http.Request) {
	makeFilename := func(bundleDir string, id int64) string {
		return paths.UploadPartFilename(bundleDir, id, indexFromRequest(r))
	}

	_ = s.doUpload(w, r, makeFilename)
}

// POST /uploads/{id:[0-9]+}/stitch
func (s *Server) handlePostUploadStitch(w http.ResponseWriter, r *http.Request) {
	id := idFromRequest(r)
	filename := paths.UploadFilename(s.bundleDir, id)
	makePartFilename := func(index int) string {
		return paths.UploadPartFilename(s.bundleDir, id, int64(index))
	}

	if err := codeintelutils.StitchFiles(filename, makePartFilename, false, false); err != nil {
		log15.Error("Failed to stitch multipart upload", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = writeFileSize(w, filename)
}

// DELETE /uploads/{id:[0-9]+}
func (s *Server) handleDeleteUpload(w http.ResponseWriter, r *http.Request) {
	s.deleteUpload(w, r)
}

// POST /dbs/{id:[0-9]+}/{index:[0-9]+}
func (s *Server) handlePostDatabasePart(w http.ResponseWriter, r *http.Request) {
	makeFilename := func(bundleDir string, id int64) string {
		return paths.DBPartFilename(bundleDir, id, indexFromRequest(r))
	}

	_ = s.doUpload(w, r, makeFilename)
}

// POST /dbs/{id:[0-9]+}/stitch
func (s *Server) handlePostDatabaseStitch(w http.ResponseWriter, r *http.Request) {
	id := idFromRequest(r)
	filename := paths.SQLiteDBFilename(s.bundleDir, idFromRequest(r))
	makePartFilename := func(index int) string {
		return paths.DBPartFilename(s.bundleDir, id, int64(index))
	}

	if err := codeintelutils.StitchFiles(filename, makePartFilename, true, false); err != nil {
		log15.Error("Failed to stitch multipart database", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Once we have a database, we no longer need the upload file
	s.deleteUpload(w, r)

	_ = writeFileSize(w, filename)
}

// GET /dbs/{id:[0-9]+}/exists
func (s *Server) handleExists(w http.ResponseWriter, r *http.Request) {
	s.dbQuery(w, r, func(ctx context.Context, db database.Database) (interface{}, error) {
		exists, err := db.Exists(ctx, getQuery(r, "path"))
		if err != nil {
			return nil, pkgerrors.Wrap(err, "db.Exists")
		}
		return exists, nil
	})
}

// GET /dbs/{id:[0-9]+}/ranges
func (s *Server) handleRanges(w http.ResponseWriter, r *http.Request) {
	s.dbQuery(w, r, func(ctx context.Context, db database.Database) (interface{}, error) {
		ranges, err := db.Ranges(ctx, getQuery(r, "path"), getQueryInt(r, "startLine"), getQueryInt(r, "endLine"))
		if err != nil {
			return nil, pkgerrors.Wrap(err, "db.Ranges")
		}
		return ranges, nil
	})
}

// GET /dbs/{id:[0-9]+}/definitions
func (s *Server) handleDefinitions(w http.ResponseWriter, r *http.Request) {
	s.dbQuery(w, r, func(ctx context.Context, db database.Database) (interface{}, error) {
		definitions, err := db.Definitions(ctx, getQuery(r, "path"), getQueryInt(r, "line"), getQueryInt(r, "character"))
		if err != nil {
			return nil, pkgerrors.Wrap(err, "db.Definitions")
		}
		return definitions, nil
	})
}

// GET /dbs/{id:[0-9]+}/references
func (s *Server) handleReferences(w http.ResponseWriter, r *http.Request) {
	s.dbQuery(w, r, func(ctx context.Context, db database.Database) (interface{}, error) {
		references, err := db.References(ctx, getQuery(r, "path"), getQueryInt(r, "line"), getQueryInt(r, "character"))
		if err != nil {
			return nil, pkgerrors.Wrap(err, "db.References")
		}
		return references, nil
	})
}

// GET /dbs/{id:[0-9]+}/hover
func (s *Server) handleHover(w http.ResponseWriter, r *http.Request) {
	s.dbQuery(w, r, func(ctx context.Context, db database.Database) (interface{}, error) {
		text, hoverRange, exists, err := db.Hover(ctx, getQuery(r, "path"), getQueryInt(r, "line"), getQueryInt(r, "character"))
		if err != nil {
			return nil, pkgerrors.Wrap(err, "db.Hover")
		}
		if !exists {
			return nil, nil
		}

		return map[string]interface{}{"text": text, "range": hoverRange}, nil
	})
}

// GET /dbs/{id:[0-9]+}/diagnostics
func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	s.dbQuery(w, r, func(ctx context.Context, db database.Database) (interface{}, error) {
		skip := getQueryInt(r, "skip")
		if skip < 0 {
			return nil, errors.New("illegal skip supplied")
		}

		take := getQueryIntDefault(r, "take", DefaultDiagnosticResultPageSize)
		if take <= 0 {
			return nil, errors.New("illegal take supplied")
		}

		diagnostics, count, err := db.Diagnostics(ctx, getQuery(r, "prefix"), skip, take)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "db.Diagnostics")
		}

		return map[string]interface{}{"diagnostics": diagnostics, "count": count}, nil
	})
}

// GET /dbs/{id:[0-9]+}/monikersByPosition
func (s *Server) handleMonikersByPosition(w http.ResponseWriter, r *http.Request) {
	s.dbQuery(w, r, func(ctx context.Context, db database.Database) (interface{}, error) {
		monikerLocations, err := db.MonikersByPosition(ctx, getQuery(r, "path"), getQueryInt(r, "line"), getQueryInt(r, "character"))
		if err != nil {
			return nil, pkgerrors.Wrap(err, "db.MonikersByPosition")
		}
		return monikerLocations, nil
	})
}

// GET /dbs/{id:[0-9]+}/monikerResults
func (s *Server) handleMonikerResults(w http.ResponseWriter, r *http.Request) {
	s.dbQuery(w, r, func(ctx context.Context, db database.Database) (interface{}, error) {
		var tableName string
		switch getQuery(r, "modelType") {
		case "definition":
			tableName = "definitions"
		case "reference":
			tableName = "references"
		default:
			return nil, errors.New("illegal tableName supplied")
		}

		skip := getQueryInt(r, "skip")
		if skip < 0 {
			return nil, errors.New("illegal skip supplied")
		}

		take := getQueryIntDefault(r, "take", DefaultMonikerResultPageSize)
		if take <= 0 {
			return nil, errors.New("illegal take supplied")
		}

		locations, count, err := db.MonikerResults(
			ctx,
			tableName,
			getQuery(r, "scheme"),
			getQuery(r, "identifier"),
			skip,
			take,
		)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "db.MonikerResults")
		}

		return map[string]interface{}{"locations": locations, "count": count}, nil
	})
}

// GET /dbs/{id:[0-9]+}/packageInformation
func (s *Server) handlePackageInformation(w http.ResponseWriter, r *http.Request) {
	s.dbQuery(w, r, func(ctx context.Context, db database.Database) (interface{}, error) {
		packageInformationData, exists, err := db.PackageInformation(
			ctx,
			getQuery(r, "path"),
			getQuery(r, "packageInformationId"),
		)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "db.PackageInformation")
		}
		if !exists {
			return nil, nil
		}

		return packageInformationData, nil
	})
}

// doUpload writes the HTTP request body to the path determined by the given
// makeFilename function.
func (s *Server) doUpload(w http.ResponseWriter, r *http.Request, makeFilename func(bundleDir string, id int64) string) bool {
	filename := makeFilename(s.bundleDir, idFromRequest(r))

	if err := writeToFile(filename, r.Body); err != nil {
		log15.Error("Failed to write payload", "err", err)
		http.Error(w, fmt.Sprintf("failed to write payload: %s", err.Error()), http.StatusInternalServerError)
		return false
	}

	return writeFileSize(w, filename)
}

func writeToFile(filename string, r io.Reader) (err error) {
	targetFile, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := targetFile.Close(); closeErr != nil {
			err = multierror.Append(err, closeErr)
		}
	}()

	_, err = io.Copy(targetFile, r)
	return err
}

func writeFileSize(w http.ResponseWriter, filename string) bool {
	fi, err := os.Stat(filename)
	if err != nil {
		log15.Error("Failed to stat file", "err", err)
		http.Error(w, fmt.Sprintf("failed to stat file: %s", err.Error()), http.StatusInternalServerError)
		return false
	}

	payload := map[string]int{
		"size": int(fi.Size()),
	}

	writeJSON(w, payload)
	return true
}

func (s *Server) deleteUpload(w http.ResponseWriter, r *http.Request) {
	if err := os.Remove(paths.UploadFilename(s.bundleDir, idFromRequest(r))); err != nil {
		log15.Warn("Failed to delete upload file", "err", err)
	}
}

type dbQueryHandlerFunc func(ctx context.Context, db database.Database) (interface{}, error)

// dbQuery invokes the given handler with the database instance chosen from the
// route's id value and serializes the resulting value to the response writer. If an
// error occurs it will be written to the body of a 500-level response.
func (s *Server) dbQuery(w http.ResponseWriter, r *http.Request, handler dbQueryHandlerFunc) {
	id := idFromRequest(r)

	if err := s.dbQueryErr(w, r, handler); err != nil {
		if err == sqlitereader.ErrUnknownDatabase {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		log15.Error("Failed to handle query", "err", err, "id", id)
		http.Error(w, fmt.Sprintf("failed to handle query: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

// queryBundleErr invokes the given handler with the database instance chosen from the
// route's id value and serializes the resulting value to the response writer. If an
// error occurs it will be returned.
func (s *Server) dbQueryErr(w http.ResponseWriter, r *http.Request, handler dbQueryHandlerFunc) (err error) {
	ctx := r.Context()
	filename := paths.SQLiteDBFilename(s.bundleDir, idFromRequest(r))

	span, ctx := ot.StartSpanFromContext(ctx, "dbQuery")
	span.SetTag("filename", filename)
	defer func() {
		if err != nil {
			ext.Error.Set(span, true)
			span.SetTag("err", err.Error())
		}
		span.Finish()
	}()

	store := postgresreader.NewStore(s.codeIntelDB, int(idFromRequest(r)))
	if _, err := store.ReadMeta(ctx); err != nil {
		if err != postgresreader.ErrNoMetadata {
			return err
		}
	} else {
		db, err := database.OpenDatabase(ctx, filename, persistence.NewObserved(store, s.observationContext))
		if err != nil {
			return pkgerrors.Wrap(err, "database.OpenDatabase")
		}

		payload, err := handler(ctx, database.NewObserved(db, filename, s.observationContext))
		if err != nil {
			return err
		}

		writeJSON(w, payload)
		return nil
	}

	return s.storeCache.WithStore(ctx, filename, func(store persistence.Store) error {
		db, err := database.OpenDatabase(ctx, filename, persistence.NewObserved(store, s.observationContext))
		if err != nil {
			return pkgerrors.Wrap(err, "database.OpenDatabase")
		}

		payload, err := handler(ctx, database.NewObserved(db, filename, s.observationContext))
		if err != nil {
			return err
		}

		writeJSON(w, payload)
		return nil
	})
}

// limitTransferRate applies a transfer limit to the given writer.
//
// In the case that the remote server is running on the same host as this service, an unbounded
// transfer rate can end up being so fast that we harm our own network connectivity. In order to
// prevent the disruption of other in-flight requests, we cap the transfer rate of w to 1Gbps.
func limitTransferRate(w io.Writer) io.Writer {
	return flowrate.NewWriter(w, 1000*1000*1000)
}
