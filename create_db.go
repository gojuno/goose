package goose

import "fmt"

// CreateDB creates database
func CreateDB(dbstring string) error {
	d := GetDialect()

	dbName, err := d.getDBName(dbstring)
	if err != nil {
		return fmt.Errorf("failed to get db name: %v", err)
	}

	db, err := d.connectToServer(dbstring)
	if err != nil {
		return fmt.Errorf("failed to connect to the server: %v", err)
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	return err
}
