// +build development

package main

var commands = []*Command{
	upCmd,
	downCmd,
	redoCmd,
	statusCmd,
	createCmd,
	dbVersionCmd,
	createDatabaseCmd,
	dropDatabaseCmd,
}
