package utils

import "testing"

func TestGetListHash(t *testing.T) {
	list := []string{"a", "b", "c"}
	hash := GetListHash(list)
	if hash == "" {
		t.Fatal("hash is empty")
	}

	if hash != "dcd229f9224c1d8a1b514239d207f5be800d6a78001e5f550263db0fd05ff979" {
		t.Fatalf("expected dcd229f9224c1d8a1b514239d207f5be800d6a78001e5f550263db0fd05ff979, got %s", hash)
	}
}
