package main

import (
	"database/sql"
	"path"

	_ "github.com/mattn/go-sqlite3"
)

func initDB() error {
	dbpath := path.Join(_Conf.Workspace, "pandora.db")
	var err error
	_DB, err = sql.Open("sqlite3", dbpath)
	if err != nil {
		return err
	}
	return createTablesIfNeed()
}

func createTablesIfNeed() error {
	_, err := _DB.Exec(_SQL_REPO_TB_CREATE)
	if err != nil {
		return err
	}

	_, err = _DB.Exec(_SQL_REPO_SYNCLOG_TB_CREATE)
	if err != nil {
		return err
	}
	return nil
}
