package config

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func ConnectDB() (*sqlx.DB, error) {
	connStr := "user=postgres password=postgres dbname=mango sslmode=disable"
	return sqlx.Connect("postgres", connStr)
}
