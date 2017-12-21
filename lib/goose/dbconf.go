package goose

import (
	"database/sql"
	"errors"
	"fmt"
	nurl "net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/lib/pq"
)

// DBDriver encapsulates the info needed to work with
// a specific database driver
type DBDriver struct {
	Name        string
	OpenStr     string
	OpenNoDBStr string
	Import      string
	Dialect     SqlDialect
}

type DBConf struct {
	MigrationsDir string
	Env           string
	Driver        DBDriver
	PgSchema      string
	DBName        string
	NoDB          bool
}

// extract configuration details from the given file
func NewDBConf(p, env string, pgschema string) (*DBConf, error) {

	cfgFile := filepath.Join(p, "dbconf.yml")

	f, err := yaml.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}

	drv, err := f.Get(fmt.Sprintf("%s.driver", env))
	if err != nil {
		return nil, err
	}
	drv = os.ExpandEnv(drv)

	open, err := f.Get(fmt.Sprintf("%s.open", env))
	if err != nil {
		return nil, err
	}
	open = os.ExpandEnv(open)

	var dbName, openNoDB string
	// Automatically parse postgres urls
	switch drv {
	case "postgres":
		u, err := nurl.Parse(open)
		if err == nil && u.Path != "" {
			dbName = u.Path[1:]
		}
		// Assumption: If we can parse the URL, we should
		if parsedURL, err := pq.ParseURL(open); err == nil && parsedURL != "" {
			open = parsedURL

			// exclude "dbname" from connection string
			startIdx := strings.Index(open, "dbname=")
			if startIdx != -1 {
				lastIdx := strings.Index(open[startIdx:], " ")
				if lastIdx != -1 {
					openNoDB = open[:startIdx] + open[startIdx+lastIdx:]
					openNoDB += " dbname=postgres"
				}
			}
		}
	case "mysql":
		dsn, err := mysql.ParseDSN(open)
		if err == nil {
			dbName = dsn.DBName
			dsn.DBName = ""
		}
		openNoDB = dsn.FormatDSN()
	}

	d := newDBDriver(drv, open, openNoDB)

	// allow the configuration to override the Import for this driver
	if imprt, err := f.Get(fmt.Sprintf("%s.import", env)); err == nil {
		d.Import = imprt
	}

	// allow the configuration to override the Dialect for this driver
	if dialect, err := f.Get(fmt.Sprintf("%s.dialect", env)); err == nil {
		d.Dialect = dialectByName(dialect)
	}

	if !d.IsValid() {
		return nil, errors.New(fmt.Sprintf("Invalid DBConf: %v", d))
	}

	return &DBConf{
		MigrationsDir: filepath.Join(p, "migrations"),
		Env:           env,
		Driver:        d,
		PgSchema:      pgschema,
		DBName:        dbName,
	}, nil
}

// Create a new DBDriver and populate driver specific
// fields for drivers that we know about.
// Further customization may be done in NewDBConf
func newDBDriver(name, open, openNoDB string) DBDriver {

	d := DBDriver{
		Name:        name,
		OpenStr:     open,
		OpenNoDBStr: openNoDB,
	}

	switch name {
	case "postgres":
		d.Import = "github.com/lib/pq"
		d.Dialect = &PostgresDialect{}

	case "pgx":
		d.Import = "github.com/jackc/pgx/stdlib"
		d.Dialect = &PostgresDialect{}

	case "mymysql":
		d.Import = "github.com/ziutek/mymysql/godrv"
		d.Dialect = &MySqlDialect{}

	case "mysql":
		d.Import = "github.com/go-sql-driver/mysql"
		d.Dialect = &MySqlDialect{}
	}

	return d
}

// ensure we have enough info about this driver
func (drv *DBDriver) IsValid() bool {
	return len(drv.Import) > 0 && drv.Dialect != nil
}

// OpenDBFromDBConf wraps database/sql.DB.Open() and configures
// the newly opened DB based on the given DBConf.
//
// Callers must Close() the returned DB.
func OpenDBFromDBConf(conf *DBConf) (*sql.DB, error) {
	var open = conf.Driver.OpenStr
	if conf.NoDB {
		open = conf.Driver.OpenNoDBStr
	}
	db, err := sql.Open(conf.Driver.Name, open)
	if err != nil {
		return nil, err
	}

	// if a postgres schema has been specified, apply it
	if conf.Driver.Name == "postgres" && conf.PgSchema != "" {
		if _, err := db.Exec("SET search_path TO " + conf.PgSchema); err != nil {
			return nil, err
		}
	}

	return db, nil
}
