package config

import (
	"os"
	"testing"
)

func TestConnectDatabasePanicsWhenConnectionFails(t *testing.T) {
	for key, value := range map[string]string{
		"DB_HOST":     "invalid[host",
		"DB_USER":     "test",
		"DB_PASSWORD": "test",
		"DB_NAME":     "test",
		"DB_PORT":     "5432",
	} {
		os.Setenv(key, value)
		t.Cleanup(func() { os.Unsetenv(key) })
	}

	assertPanics(t, func() { ConnectDatabase() })
}

func assertPanics(t *testing.T, function func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected function to panic")
		}
	}()
	function()
}
