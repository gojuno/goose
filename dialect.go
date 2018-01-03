package goose

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// SQLDialect abstracts the details of specific SQL dialects
// for goose's few SQL specific statements
type SQLDialect interface {
	createVersionTableSQL() string // sql string to create the goose_db_version table
	insertVersionSQL() string      // sql string to insert the initial version table row
	dbVersionQuery(db *sql.DB) (*sql.Rows, error)
	getDBName(dbstring string) (string, error)
	connectToServer(dbstring string) (*sql.DB, error) //ignores dbname when connecting to the server
}

var dialect SQLDialect = &PostgresDialect{}

// GetDialect gets the SQLDialect
func GetDialect() SQLDialect {
	return dialect
}

// SetDialect sets the SQLDialect
func SetDialect(d string) error {
	switch d {
	case "postgres", "pgx":
		dialect = &PostgresDialect{}
	case "mysql":
		dialect = &MySQLDialect{}
	case "redshift":
		dialect = &RedshiftDialect{}
	case "tidb":
		dialect = &TiDBDialect{}
	default:
		return fmt.Errorf("%q: unknown dialect", d)
	}

	return nil
}

////////////////////////////
// Postgres
////////////////////////////

// PostgresDialect struct.
type PostgresDialect struct{}

func (pg PostgresDialect) createVersionTableSQL() string {
	return `CREATE TABLE goose_db_version (
            	id serial NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`
}

func (pg PostgresDialect) insertVersionSQL() string {
	return "INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, $2);"
}

func (pg PostgresDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("SELECT version_id, is_applied from goose_db_version ORDER BY id DESC")
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (pg PostgresDialect) connectToServer(dbstring string) (*sql.DB, error) {
	var connstring string

	dbURL, err := url.ParseRequestURI(dbstring)
	if err != nil {
		//if strings.Contains(dbstring, "dbname=")
		connstring = regexp.MustCompile(`(dbname=)(.*?)( .*|$)`).ReplaceAllString(dbstring, "dbname=postgres$3")
		if connstring == dbstring {
			return nil, fmt.Errorf("unsupported dbstring: %q", dbstring)
		}
	} else {
		dbURL.Path = "postgres"
		connstring = dbURL.String()
	}

	return sql.Open("postgres", connstring)
}

func (pg PostgresDialect) getDBName(dbstring string) (string, error) {
	dbURL, err := url.ParseRequestURI(dbstring)
	if err != nil {
		dbName := regexp.MustCompile(`.*dbname=(.*?)( .*|$)`).ReplaceAllString(dbstring, `$1`)
		if dbName == dbstring {
			return "", fmt.Errorf("unsupported dbstring: %q", dbstring)
		}

		return dbName, nil
	}
	return strings.Replace(dbURL.Path, "/", "", -1), nil
}

////////////////////////////
// MySQL
////////////////////////////

// MySQLDialect struct.
type MySQLDialect struct{}

func (m MySQLDialect) createVersionTableSQL() string {
	return `CREATE TABLE goose_db_version (
                id serial NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`
}

func (m MySQLDialect) insertVersionSQL() string {
	return "INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, ?);"
}

func (m MySQLDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("SELECT version_id, is_applied from goose_db_version ORDER BY id DESC")
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (m MySQLDialect) connectToServer(dbstring string) (*sql.DB, error) {
	return nil, errors.New("not implemented")
}

func (m MySQLDialect) getDBName(dbstring string) (string, error) {
	dbURL, err := url.ParseRequestURI(dbstring)
	if err != nil {
		return "", err
	}
	return strings.Replace(dbURL.Path, "/", "", -1), nil
}

////////////////////////////
// Redshift
////////////////////////////

// RedshiftDialect struct.
type RedshiftDialect struct{}

func (rs RedshiftDialect) createVersionTableSQL() string {
	return `CREATE TABLE goose_db_version (
            	id integer NOT NULL identity(1, 1),
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default sysdate,
                PRIMARY KEY(id)
            );`
}

func (rs RedshiftDialect) insertVersionSQL() string {
	return "INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, $2);"
}

func (rs RedshiftDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("SELECT version_id, is_applied from goose_db_version ORDER BY id DESC")
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (rs RedshiftDialect) connectToServer(dbstring string) (*sql.DB, error) {
	var connstring string

	dbURL, err := url.ParseRequestURI(dbstring)
	if err != nil {
		//if strings.Contains(dbstring, "dbname=")
		connstring = regexp.MustCompile(`(dbname=)(.*?)( .*|$)`).ReplaceAllString(dbstring, "dbname=postgres$3")
		if connstring == dbstring {
			return nil, fmt.Errorf("unsupported dbstring: %q", dbstring)
		}
	} else {
		dbURL.Path = "postgres"
		connstring = dbURL.String()
	}

	return sql.Open("postgres", connstring)

}

func (rs RedshiftDialect) getDBName(dbstring string) (string, error) {
	dbURL, err := url.ParseRequestURI(dbstring)
	if err != nil {
		dbName := regexp.MustCompile(`.*dbname=(.*?)( .*|$)`).ReplaceAllString(dbstring, `$1`)
		if dbName == dbstring {
			return "", fmt.Errorf("unsupported dbstring: %q", dbstring)
		}

		return dbName, nil
	}
	return strings.Replace(dbURL.Path, "/", "", -1), nil
}

////////////////////////////
// TiDB
////////////////////////////

// TiDBDialect struct.
type TiDBDialect struct{}

func (m TiDBDialect) createVersionTableSQL() string {
	return `CREATE TABLE goose_db_version (
                id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`
}

func (m TiDBDialect) insertVersionSQL() string {
	return "INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, ?);"
}

func (m TiDBDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("SELECT version_id, is_applied from goose_db_version ORDER BY id DESC")
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (m TiDBDialect) connectToServer(dbstring string) (*sql.DB, error) {
	return nil, errors.New("not implemented")
}

func (m TiDBDialect) getDBName(dbstring string) (string, error) {
	dbURL, err := url.ParseRequestURI(dbstring)
	if err != nil {
		return "", err
	}
	return strings.Replace(dbURL.Path, "/", "", -1), nil
}
