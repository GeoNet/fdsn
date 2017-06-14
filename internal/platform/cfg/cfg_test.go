package cfg_test

import (
	"github.com/GeoNet/fdsn/internal/platform/cfg"
	"os"
	"strings"
	"testing"
)

func TestPostgresEnv(t *testing.T) {
	os.Setenv("DB_HOST", "")
	os.Setenv("DB_USER", "")
	os.Setenv("DB_PASSWD", "")
	os.Setenv("DB_NAME", "")
	os.Setenv("DB_SSLMODE", "")
	os.Setenv("DB_CONN_TIMEOUT", "")
	os.Setenv("DB_MAX_IDLE_CONNS", "")
	os.Setenv("DB_MAX_OPEN_CONNS", "")

	var err error

	_, err = cfg.PostgresEnv()
	if err == nil {
		t.Error("expected error")
	}
	if !strings.HasPrefix(err.Error(), "DB_HOST") {
		t.Errorf("expected error starting with DB_HOST... got: %s", err.Error())
	}

	os.Setenv("DB_HOST", "host")

	_, err = cfg.PostgresEnv()
	if err == nil {
		t.Error("expected error")
	}
	if !strings.HasPrefix(err.Error(), "DB_USER") {
		t.Errorf("expected error starting with DB_USER... got: %s", err.Error())
	}

	os.Setenv("DB_USER", "user")

	_, err = cfg.PostgresEnv()
	if err == nil {
		t.Error("expected error")
	}
	if !strings.HasPrefix(err.Error(), "DB_PASSWD") {
		t.Errorf("expected error starting with DB_NAME... got: %s", err.Error())
	}

	os.Setenv("DB_PASSWD", "passwd")

	_, err = cfg.PostgresEnv()
	if err == nil {
		t.Error("expected error")
	}
	if !strings.HasPrefix(err.Error(), "DB_NAME") {
		t.Errorf("expected error starting with DB_NAME... got: %s", err.Error())
	}

	os.Setenv("DB_NAME", "name")

	_, err = cfg.PostgresEnv()
	if err == nil {
		t.Error("expected error")
	}
	if !strings.HasPrefix(err.Error(), "DB_SSLMODE") {
		t.Errorf("expected error starting with DB_SSLMODE... got: %s", err.Error())
	}

	os.Setenv("DB_SSLMODE", "false")

	_, err = cfg.PostgresEnv()
	if err == nil {
		t.Error("expected error")
	}
	if !strings.HasPrefix(err.Error(), "DB_CONN_TIMEOUT") {
		t.Errorf("expected error starting with DB_CONN_TIMEOUT... got: %s", err.Error())
	}

	os.Setenv("DB_CONN_TIMEOUT", "30")

	_, err = cfg.PostgresEnv()
	if err == nil {
		t.Error("expected error")
	}
	if !strings.HasPrefix(err.Error(), "DB_MAX_IDLE_CONNS") {
		t.Errorf("expected error starting with DB_MAX_IDLE_CONNS... got: %s", err.Error())
	}

	os.Setenv("DB_MAX_IDLE_CONNS", "1")

	_, err = cfg.PostgresEnv()
	if err == nil {
		t.Error("expected error")
	}
	if !strings.HasPrefix(err.Error(), "DB_MAX_OPEN_CONNS") {
		t.Errorf("expected error starting with DB_MAX_OPEN_CONNS... got: %s", err.Error())
	}

	os.Setenv("DB_MAX_OPEN_CONNS", "2")

	p, err := cfg.PostgresEnv()
	if err != nil {
		t.Errorf("unexpected error %s", err)
	}

	if p.Connection() != "host=host connect_timeout=30 user=user password=passwd dbname=name sslmode=false" {
		t.Errorf("expected host=host connect_timeout=30 user=user password=passwd dbname=name sslmode=false got %s", p.Connection())
	}

	if p.MaxIdle != 1 {
		t.Errorf("expected 1 got %d", p.MaxIdle)
	}

	if p.MaxOpen != 2 {
		t.Errorf("expected 2 got %d", p.MaxOpen)
	}
}
