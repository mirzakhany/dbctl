package dbctlgo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

var (
	// ErrInvalidDatabaseType is returned when an invalid database type is passed
	ErrInvalidDatabaseType = errors.New("invalid database type")
)

// Database types
const (
	// DatabasePostgres is a postgres database
	DatabasePostgres = "postgres"
	// DatabaseRedis is a redis database
	DatabaseRedis = "redis"
	// DatabaseMongoDB is a mongodb database
	DatabaseMongoDB = "mongodb"
)

// MustCreatePostgresDB create a postgres database and return connection string or fail the test
func MustCreatePostgresDB(t *testing.T, opts ...Option) string {
	return MustCreateDB(t, DatabasePostgres, opts...)
}

// MustCreateRedisDB create a redis database and return connection string or fail the test
func MustCreateRedisDB(t *testing.T, opts ...Option) string {
	return MustCreateDB(t, DatabaseRedis, opts...)
}

// MustCreateMongoDB
func MustCreateMongoDB(t *testing.T, ops ...Option) string {
	return MustCreateDB(t, DatabaseMongoDB, ops...)
}

// MustCreateDB create a database and return connection string or fail the test
// it will also remove the database after the test is finished
func MustCreateDB(t *testing.T, dbType string, opts ...Option) string {
	uri, err := CreateDB(dbType, opts...)
	if err != nil {
		t.Fatalf("failed to create %s database: %v", dbType, err)
	}

	t.Cleanup(func() {
		// the same options have to be used here, the database was created through
		// the server they point at.
		// report, do not abort: the remaining cleanup functions still have to run
		if err := RemoveDB(dbType, uri, opts...); err != nil {
			t.Errorf("failed to remove %s database: %v", dbType, err)
		}
	})

	return uri
}

// RemoveDB remove a database using connection string
func RemoveDB(dbType, uri string, opts ...Option) error {
	cfg, err := configFrom(opts)
	if err != nil {
		return err
	}

	return httpDoRemoveDBRequest(&RemoveDBRequest{Type: dbType, URI: uri}, cfg.getHostURL())
}

// configFrom applies the options to a copy of the defaults, so that the options of
// one call never become the options of the next one.
func configFrom(opts []Option) (*config, error) {
	cfg := *defaultConfig
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}
	return &cfg, nil
}

// CreateDB create a database and return connection string
// it up to the caller to remove the database by calling RemoveDB
func CreateDB(dbType string, opts ...Option) (string, error) {
	if dbType != DatabaseRedis && dbType != DatabasePostgres && dbType != DatabaseMongoDB {
		return "", ErrInvalidDatabaseType
	}

	cfg, err := configFrom(opts)
	if err != nil {
		return "", err
	}

	var migrationsPath, fixturesPath string
	if cfg.migrations != "" {
		s, err := filepath.Abs(cfg.migrations)
		if err != nil {
			return "", fmt.Errorf("get migraions absolute path failed, %w", err)
		}
		migrationsPath = s
	}

	if cfg.fixtures != "" {
		s, err := filepath.Abs(cfg.fixtures)
		if err != nil {
			return "", fmt.Errorf("get fixtures absolute path failed, %w", err)
		}
		fixturesPath = s
	}

	req := &CreateDBRequest{
		Type:                  dbType,
		Migrations:            migrationsPath,
		MigrationsRegex:       cfg.migrationsFileRegex,
		Fixtures:              fixturesPath,
		WithDefaultMigrations: cfg.withDefaultMigrations,
		InstanceName:          cfg.instanceDBName,
		InstancePass:          cfg.instancePass,
		InstancePort:          cfg.instancePort,
		InstanceUser:          cfg.instanceUser,
	}

	res, err := httpDoCreateDBRequest(req, cfg.getHostURL())
	if err != nil {
		return "", fmt.Errorf("create %s database failed: %w", dbType, err)
	}

	return res.URI, nil
}

// ErrorMessage is representing rest api error object
type ErrorMessage struct {
	Error string `json:"error"`
}

// CreateDBRequest is the request object for creating a database
type CreateDBRequest struct {
	Type            string `json:"type"`
	Migrations      string `json:"migrations"`
	MigrationsRegex string `json:"migrations_regex,omitempty"` // optional regex to filter migration files
	Fixtures        string `json:"fixtures"`

	// WithDefaultMigrations reuses the migrations the instance was started with
	// instead of uploading them with every request.
	WithDefaultMigrations bool `json:"with_default_migrations"`

	// postgres instance information
	InstancePort uint32 `json:"instance_port"`
	InstanceUser string `json:"instance_user"`
	InstancePass string `json:"instance_pass"`
	InstanceName string `json:"instance_name"`
}

// CreateDBResponse is the response object for creating a database
type CreateDBResponse struct {
	URI string `json:"uri"`
}

// RemoveDBRequest is the request object for removing a database
type RemoveDBRequest struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
}

// Request is eatheir CreateDBRequest or RemoveDBRequest
type Request interface {
	CreateDBRequest | RemoveDBRequest
}

// Response is eatheir CreateDBResponse or ErrorMessage
type Response interface {
	CreateDBResponse | interface{}
}

func httpDoCreateDBRequest(r *CreateDBRequest, baseURL string) (*CreateDBResponse, error) {
	url := baseURL + "/create"

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	migrationFiles, err := getFilesList(r.Migrations, r.MigrationsRegex)
	if err != nil {
		return nil, fmt.Errorf("read migrations failed: %w", err)
	}

	fixtureFiles, err := getFilesList(r.Fixtures, "")
	if err != nil {
		return nil, fmt.Errorf("read fixtures failed: %w", err)
	}

	kv := map[string]string{
		"type":                    r.Type,
		"instance_port":           fmt.Sprintf("%d", r.InstancePort),
		"instance_user":           r.InstanceUser,
		"instance_pass":           r.InstancePass,
		"instance_name":           r.InstanceName,
		"with_default_migrations": strconv.FormatBool(r.WithDefaultMigrations),
	}

	for _, f := range migrationFiles {
		if err := addFileToWriter(bodyWriter, "migrations", f); err != nil {
			return nil, err
		}
	}

	for _, f := range fixtureFiles {
		if err := addFileToWriter(bodyWriter, "fixtures", f); err != nil {
			return nil, err
		}
	}

	for k, v := range kv {
		if err := bodyWriter.WriteField(k, v); err != nil {
			return nil, err
		}
	}

	contentType := bodyWriter.FormDataContentType()
	if err := bodyWriter.Close(); err != nil {
		return nil, err
	}

	req, err := http.Post(url, contentType, bodyBuf)
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()

	if err := checkForError(req); err != nil {
		return nil, err
	}

	var res CreateDBResponse
	if err := json.NewDecoder(req.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &res, nil
}

func httpDoRemoveDBRequest(r *RemoveDBRequest, baseURL string) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodDelete, baseURL+"/remove", bytes.NewReader(data))
	if err != nil {
		return err
	}

	rawRes, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer rawRes.Body.Close()

	return checkForError(rawRes)
}

func addFileToWriter(w *multipart.Writer, fieldname, filename string) error {
	fileWriter, err := w.CreateFormFile(fieldname, filepath.Base(filename))
	if err != nil {
		return err
	}

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	if _, err := io.Copy(fileWriter, f); err != nil {
		return fmt.Errorf("read file (%s) failed: %w", filename, err)
	}

	return nil
}

func checkForError(r *http.Response) error {
	if r.StatusCode < 400 {
		return nil
	}

	var msg ErrorMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil || msg.Error == "" {
		return fmt.Errorf("dbctl server returned %s", r.Status)
	}

	return errors.New(msg.Error)
}

// getFilesList returns a list of files in a directory
func getFilesList(dir string, regexPattern string) ([]string, error) {
	if dir == "" {
		return nil, nil
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	var regex *regexp.Regexp
	if regexPattern != "" {
		regex, err = regexp.Compile(regexPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	var out []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		fullPath := filepath.Join(absPath, f.Name())
		if regex == nil || regex.MatchString(f.Name()) {
			out = append(out, fullPath)
		}
	}

	return out, nil
}
