/*
Package encdb defines an encrypted database used within Mute.
Such an encrypted database consists of two file for a given database file with
name "dbname":

  dbname.db
  dbname.key

The file "dbname.db" is an AES-256 encrypted sqlite3 file managed by the
package "mute/go-sqlcipher". The file named "dbname.key" is an
AES-256 encrypted text file which contains the (randomly generated) raw
encryption key for "dbname.db". To decrypt the key file the key derivation
function PBKDF2 is applied to a supplied passphrase (with a configurable number
of iterations) and the derived key is used as the AES-256 key for "dbname.key".

This design allows a very cheap rekey of the database, because only the key
file needs to be changed and the database file itself doesn't have to be
modified for a rekey operation.
*/
package encdb

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/mutecomm/mute/log"
	"github.com/mutecomm/mute/util"

	_ "github.com/mutecomm/go-sqlcipher" // encDB is tightly coupled to go-sqlcipher
)

const (
	dbsuffix  = ".db"
	keysuffix = ".key"
)

func createTables(db *sql.DB, createStmts []string) error {
	for _, stmt := range createStmts {
		_, err := db.Exec(stmt)
		if err != nil {
			return log.Errorf("encdb: %q: %s", err, stmt)
		}
	}
	return nil
}

// Create tries to create an encrypted database with the given passphrase and
// iter many KDF iterations. Thereby, dbname is the prefix of the following
// two database files which will be created and must not exist already:
//
//  dbname.db
//  dbname.key
//
// The SQL database is initialized with the statements given in createStmts.
// In case of error (for example, the database files do exist already or
// cannot be created) an error is returned.
func Create(dbname string, passphrase []byte, iter int, createStmts []string) error {
	dbfile := dbname + dbsuffix
	keyfile := dbname + keysuffix
	// make sure files do not exist already
	if _, err := os.Stat(dbfile); err == nil {
		return log.Errorf("encdb: dbfile '%s' exists already", dbfile)
	}
	if _, err := os.Stat(keyfile); err == nil {
		return log.Errorf("encdb: dbfile '%s' exists already", keyfile)
	}
	// create keyfile
	key, err := generateKeyfile(keyfile, passphrase, iter)
	if err != nil {
		return err
	}
	// create DB
	db, err := sql.Open("sqlite3", dbfile)
	if err != nil {
		return log.Error(err)
	}
	defer db.Close()
	_, err = db.Exec(fmt.Sprintf("PRAGMA key = \"x'%s'\";", hex.EncodeToString(key)))
	if err != nil {
		return log.Error(err)
	}
	_, err = db.Exec("PRAGMA cipher_page_size = 4096;")
	if err != nil {
		return log.Error(err)
	}
	_, err = db.Exec("PRAGMA auto_vacuum = FULL;")
	if err != nil {
		return log.Error(err)
	}
	return createTables(db, createStmts)
}

// Open tries to open an encrypted database with the given passphrase.
// Thereby, dbname is the prefix of the following two database files (which
// must already exist):
//
//  dbname.db
//  dbname.key
//
// In case of error (for example, the database files do not exist or the
// passphrase is wrong) an error is returned.
func Open(dbname string, passphrase []byte) (*sql.DB, error) {
	dbfile := dbname + dbsuffix
	keyfile := dbname + keysuffix
	// make sure files exists
	if _, err := os.Stat(dbfile); err != nil {
		return nil, log.Error(err)
	}
	if _, err := os.Stat(keyfile); err != nil {
		return nil, log.Error(err)
	}
	// get key from keyfile
	key, err := readKeyfile(keyfile, passphrase)
	if err != nil {
		return nil, err
	}
	// open DB
	db, err := sql.Open("sqlite3", dbfile)
	if err != nil {
		return nil, log.Error(err)
	}
	_, err = db.Exec(fmt.Sprintf("PRAGMA key = \"x'%s'\";", hex.EncodeToString(key)))
	if err != nil {
		return nil, log.Error(err)
	}
	_, err = db.Exec("PRAGMA cipher_page_size = 4096;")
	if err != nil {
		return nil, log.Error(err)
	}
	// test key
	_, err = db.Exec("select count(*) from sqlite_master")
	if err != nil {
		return nil, log.Error(err)
	}
	// enable foreign key support
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, log.Error(err)
	}
	return db, nil
}

// Rekey tries to rekey an encrypted database with the given newPassphrase and
// newIter many KDF iterations. The correct oldPassphrase must be supplied.
// Thereby, dbname is the prefix of the following two database files (which must
// already exist):
//
//  dbname.db
//  dbname.key
//
// Rekey replaces the dbname.key file and leaves the dbname.db file unmodified,
// allowing for very fast rekey operations. In case of error (for example, the
// database files do not exist or the oldPassphrase is wrong) an error is
// returned.
func Rekey(dbname string, oldPassphrase, newPassphrase []byte, newIter int) error {
	encdb, err := Open(dbname, oldPassphrase)
	if err != nil {
		return err
	}
	defer encdb.Close()
	keyfile := dbname + keysuffix
	return replaceKeyfile(keyfile, oldPassphrase, newPassphrase, newIter)
}

var autoVacuumModes = []string{
	"NONE",
	"FULL",
	"INCREMENTAL",
}

// Status returns the autoVacuum and freelistCount of db.
func Status(db *sql.DB) (autoVacuum string, freelistCount int64, err error) {
	var av int64
	err = db.QueryRow("PRAGMA auto_vacuum;").Scan(&av)
	if err != nil {
		return "", 0, err
	}
	autoVacuum = autoVacuumModes[av]
	err = db.QueryRow("PRAGMA freelist_count;").Scan(&freelistCount)
	if err != nil {
		return "", 0, err
	}
	return
}

// Vacuum executes VACUUM command in db. If autoVacuumMode is not nil and
// different from the current one, the auto_vacuum mode is changed before
// VACUUM is executed.
func Vacuum(db *sql.DB, autoVacuumMode string) error {
	if autoVacuumMode != "" {
		if !util.ContainsString(autoVacuumModes, autoVacuumMode) {
			return log.Errorf("encdb: unknown auto_vacuum mode: %s", autoVacuumMode)
		}
		var av int64
		err := db.QueryRow("PRAGMA auto_vacuum;").Scan(&av)
		if err != nil {
			return err
		}
		currentMode := autoVacuumModes[av]
		if currentMode != autoVacuumMode {
			_, err = db.Exec(fmt.Sprintf("PRAGMA auto_vacuum = %s;", autoVacuumMode))
			if err != nil {
				return err
			}
		}
	}
	_, err := db.Exec("VACUUM;")
	if err != nil {
		return log.Error(err)
	}
	return nil
}

// Incremental executes incremental_vacuum to free up to pages many pages. If
// pages is 0, all pages are freed. If the current auto_vacuum mode is not
// INCREMENTAL, an error is returned.
func Incremental(db *sql.DB, pages int64) error {
	var av int64
	err := db.QueryRow("PRAGMA auto_vacuum;").Scan(&av)
	if err != nil {
		return err
	}
	if autoVacuumModes[av] != "INCREMENTAL" {
		return log.Error("encdb: current auto_vacuum mode is not INCREMENTAL")
	}
	_, err = db.Exec(fmt.Sprintf("PRAGMA incremental_vacuum(%d);", pages))
	if err != nil {
		log.Error(err)
	}
	return nil
}
