package ports

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	pg "github.com/mirzakhany/dbctl/internal/database/postgres"
)

var (
	ErrBadRequest          = Message{Error: "ErrBadRequest"}
	ErrInternalServerError = Message{Error: "ErrInternalServerError"}
)

func StartHTTPServer(ctx context.Context, port string) error {
	mux := &http.ServeMux{}
	srv := http.Server{
		Addr:    net.JoinHostPort(":", port),
		Handler: mux,
	}

	mux.HandleFunc("/create-database", createDatabase)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("start http server failed: %s\n", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutdown signal received, stopping database")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("http server shutdown failed %s", err)
	}

	return nil
}

func createDatabase(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrBadRequest.WithError(err))
		return
	}

	var req CreateDatabaseRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrBadRequest.WithError(err))
		return
	}

	db, err := pg.New(
		pg.WithHost("postgres", "postgres", "postgres", 54321),
		pg.WithLogger(io.Discard),
		pg.WithMigrations(req.MigrationFilesPath),
		pg.WithFixtures(req.FixtureFilePath),
	)
	if err != nil {
		return
	}

	if err := db.Start(context.Background(), true); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrInternalServerError.WithError(err))
		return
	}

	writeJSON(w, http.StatusCreated, CreateDatabaseResponse{
		DatabaseName:  "postgres",
		ConnectionURI: db.URI(),
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	d, err := json.Marshal(data)
	if err != nil {
		log.Printf("writejJSON failed %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code": 500, "message": "Internal Server error"}`))
		return
	}

	w.WriteHeader(statusCode)
	_, _ = w.Write(d)
}
