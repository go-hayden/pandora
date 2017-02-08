package main

const _REPO_TB_CREATE = `
CREATE TABLE IF NOT EXISTS repo (
	key        TEXT NOT NULL PRIMARY KEY,
	repo       TEXT NOT NULL,
	module     TEXT NOT NULL,
	version    TEXT NOT NULL,
	path       TEXT NOT NULL,
	source     TEXT,
	ctime      datetime
)
`
const _DEPENDENCY_TB_CREATE = `
CREATE TABLE IF NOT EXISTS dependency (
	key        TEXT NOT NULL,
	sub_module TEXT NOT NULL DEFAULT '*',
	dependency TEXT NOT NULL,
	version    TEXT NOT NULL DEFAULT '',
	ctime	   datetime,
	PRIMARY KEY(key, sub_module, dependency)
)
`

const _REPO_SYNCLOG_TB_CREATE = `
CREATE TABLE IF NOT EXISTS updatelog (
	sync_time      datetime
)
`
