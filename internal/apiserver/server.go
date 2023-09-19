package apiserver

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mirzakhany/dbctl/internal/database"
	pg "github.com/mirzakhany/dbctl/internal/database/postgres"
	rs "github.com/mirzakhany/dbctl/internal/database/redis"
	"github.com/mirzakhany/dbctl/internal/logger"
)

// DefaultPort is the default port for the testing server
const DefaultPort = "1988"

// Server is the testing server
type Server struct {
	port string
}

// NewServer creates a new testing server
func NewServer(port string) *Server {
	return &Server{port: port}
}

// Start starts the testing server
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.Handle("/create", http.HandlerFunc(s.CreateDB))
	mux.Handle("/remove", http.HandlerFunc(s.RemoveDB))

	srv := &http.Server{
		Addr:    net.JoinHostPort("", s.port),
		Handler: mux,
	}

	errs := make(chan error, 1)
	go func() {
		logger.Info("starting testing server on port", s.port)
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
	}

	if req.Type == "" {
		JSONError(w, http.StatusBadRequest, "type is required")
		return
	}

	// check if type is one of valid options
	if req.Type != "postgres" && req.Type != "redis" {
		JSONError(w, http.StatusBadRequest, "type is not valid, valid options are postgres or redis")
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

	var uri string
	var createErr error

	switch req.Type {
	case "postgres":
		uri, createErr = createPostgresDB(r.Context(), req)
	case "redis":
		uri, createErr = createRedisDB(r.Context(), req)
	}

	if createErr != nil {
		JSONError(w, http.StatusInternalServerError, createErr.Error())
		return
	}

	JSON(w, http.StatusOK, CreateDBResponse{URI: uri})
}

func readMulipartFiles(r *http.Request, key, dst string) error {
	for _, f := range r.MultipartForm.File[key] {
		dst, err := os.Create(filepath.Join(dst, f.Filename))
		if err != nil {
			return err
		}
		f, err := f.Open()
		if err != nil {
			return err
		}
		io.Copy(dst, f)
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

	var err error
	switch req.Type {
	case "postgres":
		err = removePostgresDB(r.Context(), req)
	case "redis":
		err = removeRedisDB(r.Context(), req)
	}

	if err != nil {
		JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusNoContent, `"{"message":"db removed successfully"}"`)
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

	pgDB, _ := pg.New(pg.WithHost(r.InstanceUser, r.InstancePass, r.InstanceName, r.InstancePort))
	res, err := pgDB.CreateDB(ctx, &database.CreateDBRequest{
		Migrations: r.Migrations,
		Fixtures:   r.Fixtures,
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

	// TODO handle redis fixtures
	rsDB, _ := rs.New()
	res, err := rsDB.CreateDB(ctx, &database.CreateDBRequest{})

	if err != nil {
		return "", err
	}

	return res.URI, nil
}

func removePostgresDB(ctx context.Context, r *RemoveDBRequest) error {
	pgDB, _ := pg.New()
	return pgDB.RemoveDB(ctx, r.URI)
}

func removeRedisDB(ctx context.Context, r *RemoveDBRequest) error {
	rsDB, _ := rs.New()
	return rsDB.RemoveDB(ctx, r.URI)
}

// JSONError writes the given status code and error message to the ResponseWriter.
func JSONError(w http.ResponseWriter, status int, err string) {
	JSON(w, status, map[string]string{"error": err})
}

// JSON writes the given status code and data to the ResponseWriter.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("error encoding json", err)
		return
	}
}
