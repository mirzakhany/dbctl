package golang

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
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
)

// MustCreatePostgresDB create a postgres database and return connection string or fail the test
func MustCreatePostgresDB(t *testing.T, opts ...Option) string {
	return MustCreateDB(t, DatabasePostgres, opts...)
}

// MustCreateRedisDB create a redis database and return connection string or fail the test
func MustCreateRedisDB(t *testing.T, opts ...Option) string {
	return MustCreateDB(t, DatabaseRedis, opts...)
}

// MustCreateDB create a database and return connection string or fail the test
// it will also remove the database after the test is finished
func MustCreateDB(t *testing.T, dbType string, opts ...Option) string {
	uri, err := CreateDB(dbType, opts...)
	if err != nil {
		t.Fatalf("failed to create %s database: %v", dbType, err)
	}

	t.Cleanup(func() {
		if err := RemoveDB(dbType, uri); err != nil {
			t.Fatalf("failed to remove %s database: %v", dbType, err)
		}
	})

	return uri
}

// RemoveDB remove a database using connection string
func RemoveDB(dbType, uri string) error {
	return sendRemoveRequest(&RemoveDBRequest{Type: dbType, URI: uri}, defaultConfig.getHostURL())
}

// CreateDB create a database and return connection string
// it up to the caller to remove the database by calling RemoveDB
func CreateDB(dbType string, opts ...Option) (string, error) {
	if dbType != DatabaseRedis && dbType != DatabasePostgres {
		return "", ErrInvalidDatabaseType
	}

	var cfg = defaultConfig
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return "", err
		}
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
		Type:         dbType,
		Migrations:   migrationsPath,
		Fixtures:     fixturesPath,
		InstanceName: cfg.instanceDBName,
		InstancePass: cfg.instancePass,
		InstancePort: cfg.instancePort,
		InstanceUser: cfg.instanceUser,
	}

	res, err := sendCreateRequest(req, cfg.getHostURL())
	if err != nil {
		return "", err
	}

	return res.URI, nil
}

// ErrorMessage is representing rest api error object
type ErrorMessage struct {
	Error string `json:"error"`
}

// CreateDBRequest is the request object for creating a database
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

// CreateDBResponse is the response object for creating a database
type CreateDBResponse struct {
	URI string `json:"uri"`
}

func sendCreateRequest(r *CreateDBRequest, baseURL string) (*CreateDBResponse, error) {
	res, err := httpDo[CreateDBRequest, CreateDBResponse](http.MethodPost, r, baseURL+"/create")
	if err != nil {
		return nil, err
	}

	return res, nil
}

// RemoveDBRequest is the request object for removing a database
type RemoveDBRequest struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
}

func sendRemoveRequest(r *RemoveDBRequest, baseURL string) error {
	_, err := httpDo[RemoveDBRequest, interface{}](http.MethodDelete, r, baseURL+"/remove")
	return err
}

// Response is eatheir CreateDBResponse or ErrorMessage
type Response interface {
	CreateDBResponse | interface{}
}

func httpDo[Req any, Res Response](method string, r *Req, url string) (*Res, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	rawRes, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer rawRes.Body.Close()

	if err := checkForError(rawRes); err != nil {
		return nil, err
	}

	if method == http.MethodDelete {
		return nil, nil
	}

	rawData, err := io.ReadAll(rawRes.Body)
	if err != nil {
		return nil, err
	}

	var res Res
	if err := json.Unmarshal(rawData, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func checkForError(r *http.Response) error {
	if r.StatusCode >= 400 {
		var err ErrorMessage
		if err := json.NewDecoder(r.Body).Decode(&err); err != nil {
			return err
		}
		return errors.New(err.Error)
	}
	return nil
}
