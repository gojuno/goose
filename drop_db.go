package goose

import "fmt"

// DropDB drops database
func DropDB(dbstring string) error {
	d := GetDialect()

	dbName, err := d.getDBName(dbstring)
	if err != nil {
		return fmt.Errorf("failed to get db name: %v", err)
	}

	db, err := d.connectToServer(dbstring)
	if err != nil {
		return fmt.Errorf("failed to connect to the server: %v", err)
	}

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	return err
}
