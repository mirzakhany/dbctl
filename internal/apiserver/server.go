package apiserver

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/mirzakhany/dbctl/internal/database"
	pg "github.com/mirzakhany/dbctl/internal/database/postgres"
	rs "github.com/mirzakhany/dbctl/internal/database/redis"
)

const DefaultPort = "1988"

type Server struct {
	port string
}

func NewServer(port string) *Server {
	return &Server{port: port}
}

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
		log.Println("starting testing server on port", s.port)
		if err := srv.ListenAndServe(); err != nil {
			errs <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("shutting down testing server")
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

type CreateDBRequest struct {
	Type       string `json:"type"`
	Migrations string `json:"migrations"`
	Fixtures   string `json:"fixtures"`

	// postgres instance information
	InstancePort uint32 `json:"instance_port"`
	InstanceUser string `json:"instance_user"`
	InstancePass string `json:"instance_pass"`
	InstanceName string `json:"instance_name"`
}

type CreateDBResponse struct {
	URI string `json:"uri"`
}

func (s *Server) CreateDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	req := &CreateDBRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
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

	var uri string
	var err error

	switch req.Type {
	case "postgres":
		uri, err = createPostgresDB(r.Context(), req)
	case "redis":
		uri, err = createRedisDB(r.Context(), req)
	}

	if err != nil {
		JSON(w, http.StatusInternalServerError, err)
		return
	}

	JSON(w, http.StatusOK, CreateDBResponse{URI: uri})
}

type RemoveDBRequest struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
}

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

	rsDB, _ := rs.New(rs.WithHost(r.InstanceUser, r.InstancePass, 10, r.InstancePort))
	res, err := rsDB.CreateDB(ctx, &database.CreateDBRequest{
		Migrations: r.Migrations,
		Fixtures:   r.Fixtures,
	})

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

func JSONError(w http.ResponseWriter, status int, err string) {
	JSON(w, status, map[string]string{"error": err})
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
