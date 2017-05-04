package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

var retry = time.Duration(30) * time.Second
var maxOpenConns = 2
var maxIdleConns = 1

type DB struct {
	*sql.DB
}

var (
	DBHost           = os.Getenv("DB_HOST")
	DBConnectTimeout = os.Getenv("DB_CONN_TIMEOUT")
	DBUser           = os.Getenv("DB_USER")
	DBPassword       = os.Getenv("DB_PASSWD")
	DBName           = os.Getenv("DB_NAME")
	DBSSLMode        = os.Getenv("DB_SSLMODE")
)

func InitPG() (DB, error) {
	db, err := sql.Open("postgres", dbOpenString())

	if s := os.Getenv("DB_MAX_IDLE_CONNS"); s != "" {
		if i, err := strconv.Atoi(s); err == nil {
			maxIdleConns = i
		} else {
			log.Printf("DB_MAX_IDLE_CONNS setting error:%s\n", err)
		}
	}
	db.SetMaxIdleConns(maxIdleConns)

	if s := os.Getenv("DB_MAX_OPEN_CONNS"); s != "" {
		if i, err := strconv.Atoi(s); err == nil {
			maxOpenConns = i
		} else {
			log.Printf("DB_MAX_OPEN_CONNS setting error:%s\n", err)
		}
	}
	db.SetMaxOpenConns(maxOpenConns)

	return DB{db}, err
}

func (db *DB) Check() {
	for {
		if err := db.Ping(); err != nil {
			log.Printf("WARN - pinging DB: %s", err)
			log.Println("WARN - waiting then trying DB again.")
			time.Sleep(retry)
			continue
		}
		break
	}
}

func dbOpenString() (dbstring string) {
	return fmt.Sprintf("host=%s "+
		"connect_timeout=%s "+
		"user=%s "+
		"password=%s "+
		"dbname=%s "+
		"sslmode=%s",
		DBHost, DBConnectTimeout, DBUser, DBPassword, DBName, DBSSLMode)
}
