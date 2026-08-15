package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/mirzakhany/dbctl/internal/database"
	"github.com/mirzakhany/dbctl/internal/database/mongodb"
	pg "github.com/mirzakhany/dbctl/internal/database/postgres"
	rs "github.com/mirzakhany/dbctl/internal/database/redis"
	"github.com/mirzakhany/dbctl/internal/logger"
	"github.com/mirzakhany/dbctl/internal/utils"
)

// DefaultPort is the default port for the testing server
const DefaultPort = "1988"

const (
	// EnvLabel holds the label the server scopes its instance lookups to
	EnvLabel = "DBCTL_LABEL"
	// EnvInstances holds the instances discovered when the server was started, used
	// when the server itself cannot reach docker
	EnvInstances = "DBCTL_INSTANCES"
)

// Server is the testing server
type Server struct {
	port  string
	label string

	// instances discovered when the server started, by database type
	instances map[string]*database.Instance
}

// NewServer creates a new testing server. The label scopes which instances the
// server manages.
func NewServer(port, label string) *Server {
	// the containerized server is configured through the environment
	if label == "" {
		label = os.Getenv(EnvLabel)
	}

	s := &Server{port: port, label: label}

	// the containerized server has no access to docker, it is told about the
	// instances that were running when it was started instead.
	if raw := os.Getenv(EnvInstances); raw != "" {
		instances := map[string]*database.Instance{}
		if err := json.Unmarshal([]byte(raw), &instances); err != nil {
			logger.Error("ignoring unreadable "+EnvInstances, err)
		} else {
			s.instances = instances
		}
	}

	return s
}

// Start starts the testing server
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.Handle("/create", http.HandlerFunc(s.CreateDB))
	mux.Handle("/remove", http.HandlerFunc(s.RemoveDB))

	// The server creates and drops databases for whoever reaches it, so it stays on
	// loopback. In a container it has to listen on every interface to be reachable
	// at all, its published port is what keeps it local.
	host := container.LoopbackAddress
	if os.Getenv("DBCTL_INSIDE_DOCKER") == "true" {
		host = ""
	}

	srv := &http.Server{
		Addr:              net.JoinHostPort(host, s.port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errs := make(chan error, 1)
	go func() {
		logger.Info("starting testing server on", net.JoinHostPort(host, s.port))
		if err := srv.ListenAndServe(); err != nil {
			errs <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down testing server")
		// graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("testing server shutdown failed, %v", err)
		}

		return nil
	case err := <-errs:
		return err
	}
}

// CreateDBRequest is the request body for creating a database
type CreateDBRequest struct {
	Type       string `json:"type"`
	Migrations string `json:"migrations"`
	Fixtures   string `json:"fixtures"`

	// WithDefaultMigrations uses the template built when the instance was started
	// instead of the migrations sent with this request.
	WithDefaultMigrations bool `json:"with_default_migrations"`

	// postgres instance information
	InstanceHost string `json:"instance_host"`
	InstancePort uint32 `json:"instance_port"`
	InstanceUser string `json:"instance_user"`
	InstancePass string `json:"instance_pass"`
	InstanceName string `json:"instance_name"`
}

// CreateDBResponse is the response body for creating a database
type CreateDBResponse struct {
	URI string `json:"uri"`
}

// CreateDB creates a new database
func (s *Server) CreateDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	req := &CreateDBRequest{
		Type:         r.FormValue("type"),
		InstancePass: r.FormValue("instance_pass"),
		InstanceUser: r.FormValue("instance_user"),
		InstanceName: r.FormValue("instance_name"),
		InstanceHost: r.FormValue("instance_host"),

		WithDefaultMigrations: isTrue(r.FormValue("with_default_migrations")),
	}

	if p := r.FormValue("instance_port"); p != "" {
		port, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			JSONError(w, http.StatusBadRequest, fmt.Sprintf("instance_port is not a valid port number: %q", p))
			return
		}
		req.InstancePort = uint32(port)
	}

	if req.Type == "" {
		JSONError(w, http.StatusBadRequest, "type is required")
		return
	}

	// check if type is one of valid options
	if err := validateType(req.Type); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	migrationsDir, err := os.MkdirTemp("/tmp", "migrations-*")
	if err != nil {
		JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer os.RemoveAll(migrationsDir)

	fixturesDir, err := os.MkdirTemp("/tmp", "fixtures-*")
	if err != nil {
		JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer os.RemoveAll(fixturesDir)

	// read migrations
	if err := readMulipartFiles(r, "migrations", migrationsDir); err != nil {
		JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	req.Migrations = migrationsDir

	// read fixtures
	if err := readMulipartFiles(r, "fixtures", fixturesDir); err != nil {
		JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	req.Fixtures = fixturesDir

	// anything the caller left out is taken from the running instance, so that a
	// client does not have to be told which port dbctl picked.
	s.applyInstance(r.Context(), req)

	var uri string
	var createErr error

	switch req.Type {
	case database.TypePostgres:
		uri, createErr = createPostgresDB(r.Context(), req)
	case database.TypeRedis:
		uri, createErr = createRedisDB(r.Context(), req)
	case database.TypeMongoDB:
		uri, createErr = createMongoDBDB(r.Context(), req)
	}

	if createErr != nil {
		JSONError(w, http.StatusInternalServerError, createErr.Error())
		return
	}

	JSON(w, http.StatusOK, CreateDBResponse{URI: uri})
}

// instanceFor returns the running instance of a type, so that a client does not
// have to know which port it was started on. Discovery through docker is used when
// the server can reach it, otherwise the snapshot taken at start up.
func (s *Server) instanceFor(ctx context.Context, dbType string) *database.Instance {
	if instance, ok := s.instances[dbType]; ok {
		return instance
	}

	instance, err := database.FindInstance(ctx, dbType, s.label)
	if err != nil {
		logger.Debug(fmt.Sprintf("no %s instance found, falling back to the defaults: %v", dbType, err))
		return nil
	}

	return instance
}

// applyInstance fills in the connection details the caller did not send from the
// running instance.
func (s *Server) applyInstance(ctx context.Context, r *CreateDBRequest) {
	instance := s.instanceFor(ctx, r.Type)
	if instance == nil {
		return
	}

	if r.InstancePort == 0 {
		r.InstancePort = instance.Port
	}
	if r.InstanceUser == "" {
		r.InstanceUser = instance.User
	}
	if r.InstancePass == "" {
		r.InstancePass = instance.Pass
	}
	if r.InstanceName == "" {
		r.InstanceName = instance.Name
	}
}

// validateType reports whether the api server knows how to create the given
// database type.
func validateType(dbType string) error {
	if utils.OneOf(dbType, database.TypePostgres, database.TypeRedis, database.TypeMongoDB) {
		return nil
	}

	return fmt.Errorf("type %q is not valid, valid options are %s, %s or %s",
		dbType, database.TypePostgres, database.TypeRedis, database.TypeMongoDB)
}

// uploadPath turns the name of an uploaded part into a path relative to the
// directory the request is unpacked into. Clients percent encode the separators so
// that a migration living in a subdirectory survives multipart parsing, which
// otherwise reduces every name to its last element.
func uploadPath(filename string) (string, error) {
	name := filename
	if decoded, err := url.PathUnescape(filename); err == nil {
		name = decoded
	}

	name = filepath.Clean(filepath.FromSlash(name))
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "", fmt.Errorf("invalid file name %q", filename)
	}

	// never let an upload write outside the directory it belongs to
	if filepath.IsAbs(name) || strings.HasPrefix(name, "..") {
		return "", fmt.Errorf("file name %q is outside the upload directory", filename)
	}

	return name, nil
}

// isTrue reports whether a form value is a truthy flag.
func isTrue(v string) bool {
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	return err == nil && b
}

func readMulipartFiles(r *http.Request, key, dst string) error {
	// a request without any file part is not multipart encoded, in that case
	// there is simply nothing to read.
	if r.MultipartForm == nil {
		return nil
	}

	for _, f := range r.MultipartForm.File[key] {
		if err := writeMultipartFile(f, dst); err != nil {
			return err
		}
	}
	return nil
}

func writeMultipartFile(f *multipart.FileHeader, dst string) error {
	name, err := uploadPath(f.Filename)
	if err != nil {
		return err
	}

	src, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		_ = src.Close()
	}()

	target := filepath.Join(dst, name)
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}

	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, src); err != nil {
		return fmt.Errorf("write file (%s) failed: %w", name, err)
	}
	return nil
}

// RemoveDBRequest is the request body for removing a database
type RemoveDBRequest struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
}

// RemoveDB removes the given database
func (s *Server) RemoveDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		JSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	req := &RemoveDBRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Type == "" {
		JSONError(w, http.StatusBadRequest, "type is required")
		return
	}

	if req.URI == "" {
		JSONError(w, http.StatusBadRequest, "uri is required")
		return
	}

	if err := validateType(req.Type); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var err error
	switch req.Type {
	case database.TypePostgres:
		err = removePostgresDB(r.Context(), req)
	case database.TypeRedis:
		err = removeRedisDB(r.Context(), req)
	case database.TypeMongoDB:
		err = removeMongoDBDB(r.Context(), req)
	}

	if err != nil {
		JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusNoContent, nil)
}

func createPostgresDB(ctx context.Context, r *CreateDBRequest) (string, error) {
	if r.InstancePort == 0 {
		r.InstancePort = pg.DefaultPort
	}

	if r.InstanceUser == "" {
		r.InstanceUser = pg.DefaultUser
	}

	if r.InstancePass == "" {
		r.InstancePass = pg.DefaultPass
	}

	if r.InstanceName == "" {
		r.InstanceName = pg.DefaultName
	}

	pgDB, err := pg.New(pg.WithHost(r.InstanceUser, r.InstancePass, r.InstanceName, r.InstancePort))
	if err != nil {
		return "", err
	}

	res, err := pgDB.CreateDB(ctx, &database.CreateDBRequest{
		Migrations:            r.Migrations,
		Fixtures:              r.Fixtures,
		WithDefaultMigrations: r.WithDefaultMigrations,
	})

	if err != nil {
		return "", err
	}

	return res.URI, nil
}

func createRedisDB(ctx context.Context, r *CreateDBRequest) (string, error) {
	if r.InstancePort == 0 {
		r.InstancePort = rs.DefaultPort
	}

	if r.InstanceUser == "" {
		r.InstanceUser = rs.DefaultUser
	}

	if r.InstancePass == "" {
		r.InstancePass = rs.DefaultPass
	}

	rsDB, err := rs.New(rs.WithHost(r.InstanceUser, r.InstancePass, 0, r.InstancePort))
	if err != nil {
		return "", err
	}

	res, err := rsDB.CreateDB(ctx, &database.CreateDBRequest{Fixtures: r.Fixtures})
	if err != nil {
		return "", err
	}

	return res.URI, nil
}

func createMongoDBDB(ctx context.Context, r *CreateDBRequest) (string, error) {
	if r.InstancePort == 0 {
		r.InstancePort = mongodb.DefaultPort
	}

	if r.InstanceUser == "" {
		r.InstanceUser = mongodb.DefaultUser
	}

	if r.InstancePass == "" {
		r.InstancePass = mongodb.DefaultPass
	}

	if r.InstanceName == "" {
		r.InstanceName = mongodb.DefaultName
	}

	mongoDbDB, err := mongodb.New(mongodb.WithHost(r.InstanceUser, r.InstancePass, r.InstanceName, r.InstancePort))
	if err != nil {
		return "", err
	}

	res, err := mongoDbDB.CreateDB(ctx, &database.CreateDBRequest{
		Migrations: r.Migrations,
		Fixtures:   r.Fixtures,
	})

	if err != nil {
		return "", err
	}

	return res.URI, nil
}

// instanceOptions returns the options needed to reach the instance the database
// lives on. Removing a database has to target the same instance it was created
// on, so the caller supplied connection details are honoured here as well.
func removePostgresDB(ctx context.Context, r *RemoveDBRequest) error {
	pgDB, err := pg.New(pg.WithURI(r.URI))
	if err != nil {
		return err
	}
	return pgDB.RemoveDB(ctx, r.URI)
}

func removeRedisDB(ctx context.Context, r *RemoveDBRequest) error {
	rsDB, err := rs.New(rs.WithURI(r.URI))
	if err != nil {
		return err
	}
	return rsDB.RemoveDB(ctx, r.URI)
}

func removeMongoDBDB(ctx context.Context, r *RemoveDBRequest) error {
	mongoDbDB, err := mongodb.New(mongodb.WithURI(r.URI))
	if err != nil {
		return err
	}
	return mongoDbDB.RemoveDB(ctx, r.URI)
}

// JSONError writes the given status code and error message to the ResponseWriter.
func JSONError(w http.ResponseWriter, status int, err string) {
	JSON(w, status, map[string]string{"error": err})
}

// JSON writes the given status code and data to the ResponseWriter.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)
	if data == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("error encoding json", err)
		return
	}
}
