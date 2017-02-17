package main

// ** Query **
const _SQL_QUERY_EXIST_KEY = `SELECT key FROM repo`

const _SQL_QUERY_SPEC = `SELECT repo, spec_json FROM repo WHERE module=? AND version=?`

const _SQL_QUERY_VERSIONS = `SELECT version FROM repo WHERE module=?`

// ** Insert **
const _SQL_INSERT_REPO = `
INSERT INTO repo (key, repo, module, version, path, spec_json, ctime)
VALUES (?, ?, ?, ?, ?, ?, ?)
`

const _SQLINSERT_LOG = `
INSERT INTO updatelog (sync_time) VALUES (?)
`

// ** Create Table ***
const _SQL_REPO_TB_CREATE = `
CREATE TABLE IF NOT EXISTS repo (
	key        TEXT NOT NULL PRIMARY KEY,
	repo       TEXT NOT NULL,
	module     TEXT NOT NULL,
	version    TEXT NOT NULL,
	path       TEXT NOT NULL,
	spec_json  TEXT,
	ctime      datetime
)
`

const _SQL_REPO_SYNCLOG_TB_CREATE = `
CREATE TABLE IF NOT EXISTS updatelog (
	sync_time      datetime
)
`
