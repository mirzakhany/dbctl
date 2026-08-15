package apiserver

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// A request without any file attached is not multipart encoded. The server used
// to dereference the missing form and take the connection down with it.
func TestCreateDBWithoutFilesDoesNotPanic(t *testing.T) {
	s := NewServer(DefaultPort, "")

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
	s := NewServer(DefaultPort, "")

	req := httptest.NewRequest(http.MethodPost, "/create",
		strings.NewReader("type=postgres&instance_port=not-a-port"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res := httptest.NewRecorder()
	s.CreateDB(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an unparsable port, got %d", res.Code)
	}
}

func TestUploadPath(t *testing.T) {
	ok := map[string]string{
		"001_init.up.sql":            "001_init.up.sql",
		"tenant%2F001_init.up.sql":   "tenant/001_init.up.sql",
		"a%2Fb%2F002_second.up.sql":  "a/b/002_second.up.sql",
		"50%25_off.sql":              "50%_off.sql",
		"plain_name_with_%_char.sql": "plain_name_with_%_char.sql",
	}

	for in, want := range ok {
		got, err := uploadPath(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if got != filepath.FromSlash(want) {
			t.Fatalf("%q: expected %q, got %q", in, want, got)
		}
	}

	// an upload must never be able to write outside its directory
	for _, in := range []string{"..%2F..%2Fetc%2Fpasswd", "%2Fetc%2Fpasswd", "..", "."} {
		if got, err := uploadPath(in); err == nil {
			t.Fatalf("expected %q to be rejected, got %q", in, got)
		}
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
