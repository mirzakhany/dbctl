package apiserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A request without any file attached is not multipart encoded. The server used
// to dereference the missing form and take the connection down with it.
func TestCreateDBWithoutFilesDoesNotPanic(t *testing.T) {
	s := NewServer(DefaultPort)

	req := httptest.NewRequest(http.MethodPost, "/create",
		strings.NewReader("type=postgres&instance_port=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res := httptest.NewRecorder()
	s.CreateDB(res, req)

	// no database is running in a unit test, the point is that the request is
	// answered at all instead of panicking.
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected the request to be answered with 500, got %d: %s", res.Code, res.Body.String())
	}
}

func TestValidateType(t *testing.T) {
	// mongodb used to be rejected here even though the clients offer it
	for _, dbType := range []string{"postgres", "redis", "mongodb"} {
		if err := validateType(dbType); err != nil {
			t.Fatalf("%s was rejected: %v", dbType, err)
		}
	}

	if err := validateType("mysql"); err == nil {
		t.Fatal("expected an unknown type to be rejected")
	}
}

func TestCreateDBRejectsInvalidPort(t *testing.T) {
	s := NewServer(DefaultPort)

	req := httptest.NewRequest(http.MethodPost, "/create",
		strings.NewReader("type=postgres&instance_port=not-a-port"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res := httptest.NewRecorder()
	s.CreateDB(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an unparsable port, got %d", res.Code)
	}
}

func TestIsTrue(t *testing.T) {
	for _, v := range []string{"true", "True", "1", " true "} {
		if !isTrue(v) {
			t.Fatalf("expected %q to be true", v)
		}
	}

	for _, v := range []string{"", "false", "0", "yes please"} {
		if isTrue(v) {
			t.Fatalf("expected %q to be false", v)
		}
	}
}
