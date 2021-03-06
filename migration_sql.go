package goose

import (
	"bufio"
	"bytes"
	"database/sql"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

const (
	sqlCmdPrefix   = "-- +goose "
	scannerBufSize = 4 * 1024 * 1024
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, scannerBufSize)
	},
}

// Checks the line to see if the line has a statement-ending semicolon
// or if the line contains a double-dash comment.
func endsWithSemicolon(line []byte) bool {
	prev := ""
	scanner := bufio.NewScanner(bytes.NewReader(line))
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)

	scanner.Buffer(buf, cap(buf))
	scanner.Split(bufio.ScanWords)

	for scanner.Scan() {
		word := scanner.Text()
		if strings.HasPrefix(word, "--") {
			break
		}
		prev = word
	}

	return strings.HasSuffix(prev, ";")
}

// Split the given sql script into individual statements.
//
// The base case is to simply split on semicolons, as these
// naturally terminate a statement.
//
// However, more complex cases like pl/pgsql can have semicolons
// within a statement. For these cases, we provide the explicit annotations
// 'StatementBegin' and 'StatementEnd' to allow the script to
// tell us to ignore semicolons.
func getSQLStatements(r io.Reader, direction bool) (stmts []string, tx bool) {
	var buf bytes.Buffer

	buff := bufferPool.Get().([]byte)
	defer bufferPool.Put(buff)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(buff, cap(buff))

	// track the count of each section
	// so we can diagnose scripts with no annotations
	upSections := 0
	downSections := 0

	statementEnded := false
	ignoreSemicolons := false
	directionIsActive := false
	tx = true

	for scanner.Scan() {
		line := scanner.Bytes()

		// handle any goose-specific commands
		if bytes.HasPrefix(line, []byte(sqlCmdPrefix)) {
			cmd := bytes.TrimSpace(line[len(sqlCmdPrefix):])
			switch string(cmd) {
			case "Up":
				directionIsActive = (direction == true)
				upSections++
				break

			case "Down":
				directionIsActive = (direction == false)
				downSections++
				break

			case "StatementBegin":
				if directionIsActive {
					ignoreSemicolons = true
				}
				break

			case "StatementEnd":
				if directionIsActive {
					statementEnded = (ignoreSemicolons == true)
					ignoreSemicolons = false
				}
				break

			case "NO TRANSACTION":
				tx = false
				break
			}
		}

		if !directionIsActive {
			continue
		}

		if _, err := buf.Write(line); err != nil {
			log.Fatalf("io err: %v", err)
		}

		if _, err := buf.WriteString("\n"); err != nil {
			log.Fatalf("io err: %v", err)
		}

		// Wrap up the two supported cases: 1) basic with semicolon; 2) psql statement
		// Lines that end with semicolon that are in a statement block
		// do not conclude statement.
		if (!ignoreSemicolons && endsWithSemicolon(line)) || statementEnded {
			statementEnded = false
			stmts = append(stmts, buf.String())
			buf.Reset()
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("scanning migration: %v", err)
	}

	// diagnose likely migration script errors
	if ignoreSemicolons {
		log.Println("WARNING: saw '-- +goose StatementBegin' with no matching '-- +goose StatementEnd'")
	}

	if bufferRemaining := strings.TrimSpace(buf.String()); len(bufferRemaining) > 0 {
		log.Printf("WARNING: Unexpected unfinished SQL query: %s. Missing a semicolon?\n", bufferRemaining)
	}

	if upSections == 0 && downSections == 0 {
		log.Fatalf(`ERROR: no Up/Down annotations found, so no statements were executed.
			See https://bitbucket.org/liamstask/goose/overview for details.`)
	}

	return
}

// Run a migration specified in raw SQL.
//
// Sections of the script can be annotated with a special comment,
// starting with "-- +goose" to specify whether the section should
// be applied during an Up or Down migration
//
// All statements following an Up or Down directive are grouped together
// until another direction directive is found.
func runSQLMigration(db *sql.DB, scriptFile string, v int64, direction bool) error {
	f, err := os.Open(scriptFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	statements, useTx := getSQLStatements(f, direction)

	if useTx {
		// TRANSACTION.

		tx, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}

		for _, query := range statements {
			if _, err = tx.Exec(query); err != nil {
				tx.Rollback()
				return err
			}
		}
		if _, err := tx.Exec(GetDialect().insertVersionSQL(), v, direction); err != nil {
			tx.Rollback()
			return err
		}

		return tx.Commit()
	}

	// NO TRANSACTION.
	for _, query := range statements {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	if _, err := db.Exec(GetDialect().insertVersionSQL(), v, direction); err != nil {
		return err
	}

	return nil
}
