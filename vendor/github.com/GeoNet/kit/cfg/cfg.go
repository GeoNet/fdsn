// Package cfg is for reading config from environment variables.
package cfg

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

type Postgres struct {
	Host           string // The host to connect to [DB_HOST].
	User           string // The user to sign in as [DB_USER].
	Password       string // The user's password [DB_PASSWD].
	Name           string // The name of the database to connect to [DB_NAME].
	SSLMode        string // Whether or not to use SSL [DB_SSLMODE].
	ConnectTimeout int    // Maximum wait for connection, in seconds [DB_CONN_TIMEOUT].
	MaxIdle        int    // Maximum idle db connections [DB_MAX_IDLE_CONNS].
	MaxOpen        int    // Connection pool size [DB_MAX_OPEN_CONNS].
}

// PostgresEnv returns a Postgres with configuration from the environment variables.
// Returns an error for missing config or errors parsing int values.
func PostgresEnv() (Postgres, error) {
	p := Postgres{
		Host:     os.Getenv("DB_HOST"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWD"),
		Name:     os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	c := os.Getenv("DB_CONN_TIMEOUT")
	idle := os.Getenv("DB_MAX_IDLE_CONNS")
	open := os.Getenv("DB_MAX_OPEN_CONNS")

	switch "" {
	case p.Host:
		return Postgres{}, errors.New("DB_HOST env var must be set.")
	case p.User:
		return Postgres{}, errors.New("DB_USER env var must be set.")
	case p.Password:
		return Postgres{}, errors.New("DB_PASSWD env var must be set.")
	case p.Name:
		return Postgres{}, errors.New("DB_NAME env var must be set.")
	case p.SSLMode:
		return Postgres{}, errors.New("DB_SSLMODE env var must be set.")
	case c:
		return Postgres{}, errors.New("DB_CONN_TIMEOUT env var must be set.")
	case idle:
		return Postgres{}, errors.New("DB_MAX_IDLE_CONNS env var must be set.")
	case open:
		return Postgres{}, errors.New("DB_MAX_OPEN_CONNS env var must be set.")
	}

	var err error

	p.ConnectTimeout, err = strconv.Atoi(c)
	if err != nil {
		return Postgres{}, errors.New("DB_CONN_TIMEOUT invalid: " + err.Error())
	}

	p.MaxIdle, err = strconv.Atoi(idle)
	if err != nil {
		return Postgres{}, errors.New("DB_MAX_IDLE_CONNS invalid: " + err.Error())
	}

	p.MaxOpen, err = strconv.Atoi(open)
	if err != nil {
		return Postgres{}, errors.New("DB_MAX_OPEN_CONNS invalid: " + err.Error())
	}

	return p, nil
}

func (p *Postgres) Connection() string {
	return fmt.Sprintf("host=%s connect_timeout=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.ConnectTimeout, p.User, p.Password, p.Name, p.SSLMode)
}
