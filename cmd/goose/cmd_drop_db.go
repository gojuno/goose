// +build development

package main

import (
	"fmt"
	"log"

	"bitbucket.org/liamstask/goose/lib/goose"
)

var dropDatabaseCmd = &Command{
	Name:    "drop_db",
	Usage:   "",
	Summary: "Drop the database",
	Help:    `[soft] use IF EXISTS query`,
	Run:     dropDatabaseRun,
}

func dropDatabaseRun(cmd *Command, args ...string) {
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

	var ifExists = ""
	if len(args) > 0 && args[0] == "soft" {
		ifExists = "IF EXISTS"
	}
	if _, err := db.Exec(fmt.Sprintf("DROP DATABASE %s %s", ifExists, conf.DBName)); err != nil {
		log.Fatal(err)
	}

	fmt.Println("goose: database dropped")
}
