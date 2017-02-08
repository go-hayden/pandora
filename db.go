package main

import (
	"database/sql"
	"path"

	_ "github.com/mattn/go-sqlite3"
)

func initDB() error {
	dbpath := path.Join(config["path"], "pandora.db")
	var err error
	db, err = sql.Open("sqlite3", dbpath)
	if err != nil {
		return err
	}
	return createTablesIfNeed()
}

func createTablesIfNeed() error {
	_, err := db.Exec(_REPO_TB_CREATE)
	if err != nil {
		return err
	}
	_, err = db.Exec(_DEPENDENCY_TB_CREATE)
	if err != nil {
		return err
	}
	_, err = db.Exec(_REPO_SYNCLOG_TB_CREATE)
	if err != nil {
		return err
	}
	return nil
}
