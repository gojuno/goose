package main

import (
	"database/sql"
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/gojuno/goose"
	"gopkg.in/yaml.v2"

	// Init DB drivers.
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/ziutek/mymysql/godrv"
)

var (
	flags        = flag.NewFlagSet("goose", flag.ExitOnError)
	dir          = flags.String("dir", "db/migrations", "directory with migration files")
	conf         = flags.String("conf", "etc/config.yaml", "configuration file")
	driverFlag   = flags.String("driver", "", "db driver")
	dbstringFlag = flags.String("dbstring", "", "db conn string")
)

func main() {
	flags.Usage = usage
	flags.Parse(os.Args[1:])

	args := flags.Args()

	if len(args) > 1 && args[0] == "create" {
		if err := goose.Run("create", nil, *dir, args[1:]...); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	}

	if len(args) < 1 {
		flags.Usage()
		return
	}

	if args[0] == "-h" || args[0] == "--help" {
		flags.Usage()
		return
	}

	command, args := args[0], args[1:]

	driver, dbstring := *driverFlag, *dbstringFlag
	switch {
	case driver != "" && dbstring != "":
	case driver == "" && dbstring == "":
		var err error
		driver, dbstring, err = readConfig(*conf)
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("-dbstring and -driver must be either both present or absent")
	}

	if err := goose.SetDialect(driver); err != nil {
		log.Fatal(err)
	}

	goose.GetDialect()

	switch driver {
	case "redshift", "pgx":
		driver = "postgres"
	case "tidb":
		driver = "mysql"
	}

	if dbstring == "" {
		log.Fatalf("-dbstring=%q not supported\n", dbstring)
	}

	switch command {
	case "create_db":
		if err := goose.CreateDB(dbstring); err != nil {
			log.Fatalf("goose run: %v", err)
		}
	case "drop_db":
		if err := goose.DropDB(dbstring); err != nil {
			log.Fatalf("goose run: %v", err)
		}
	default:
		db, err := sql.Open(driver, dbstring)
		if err != nil {
			log.Fatalf("-dbstring=%q: %v\n", dbstring, err)
		}

		if err := goose.Run(command, db, *dir, args...); err != nil {
			log.Fatalf("goose run: %v", err)
		}
	}
}

// extract configuration details from the given file
func readConfig(filename string) (driver, connstring string, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", "", err
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return "", "", err
	}

	conf := struct {
		DBX struct {
			Driver     string `yaml:"Driver"`
			Connstring string `yaml:"Connstring"`
		} `yaml:"DBX"`
	}{}

	if err := yaml.Unmarshal(b, &conf); err != nil {
		return "", "", err
	}

	return os.ExpandEnv(conf.DBX.Driver), os.ExpandEnv(conf.DBX.Connstring), nil
}

func usage() {
	log.Print(usagePrefix)
	flags.PrintDefaults()
	log.Print(usageCommands)
}

var (
	usagePrefix = `Usage: goose [OPTIONS] COMMAND

Supported drivers:
    postgres
    pgx
    mysql
    redshift

Examples:
    goose status

Options:
`

	usageCommands = `
Commands:
    up                   Migrate the DB to the most recent version available
    up-to VERSION        Migrate the DB to a specific VERSION
    down                 Roll back the version by 1
    down-to VERSION      Roll back to a specific VERSION
    redo                 Re-run the latest migration
    reset                Roll back all migrations
    status               Dump the migration status for the current DB
    version              Print the current version of the database
    create NAME [sql|go] Creates new migration file with next version
    create_db            Creates database
    drop_db              Drops database
`
)
