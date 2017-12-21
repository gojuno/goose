// +build development

package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/gojuno/goose/lib/goose"
)

var createDatabaseCmd = &Command{
	Name:    "create_db",
	Usage:   "",
	Summary: "Create the database",
	Help:    `[soft] use IF NOT EXISTS query`,
	Run:     createDatabaseRun,
}

func createDatabaseRun(cmd *Command, args ...string) {
	conf, err := dbConfFromFlags()
	if err != nil {
		log.Fatal(err)
	}

	if conf.DBName == "" {
		log.Fatal(fmt.Errorf("failed to extract database name from conf: %v", conf))
	}

	conf.NoDB = true

	db, err := goose.OpenDBFromDBConf(conf)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var isSoft = len(args) > 0 && args[0] == "soft"
	switch conf.Driver.Name {
	case "postgres", "pgx":
		isExist := 0
		if isSoft {
			if err := db.QueryRow(fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname = '%s'", conf.DBName)).Scan(&isExist); err != nil {
				switch err {
				case sql.ErrNoRows:
				default:
					log.Fatal(err)
				}
			}
		}
		if isExist == 0 {
			if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", conf.DBName)); err != nil {
				log.Fatal(err)
			}
		}
	case "mysql":
		ifNotExists := ""
		if isSoft {
			ifNotExists = "IF NOT EXISTS"
		}
		if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s %s CHARACTER SET utf8 COLLATE utf8_general_ci", ifNotExists, conf.DBName)); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal(fmt.Errorf("unsupported database type: %v", conf.Driver.Name))
	}

	fmt.Println("goose: database created")
}
