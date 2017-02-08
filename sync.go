package main

import (
	"database/sql"
	"pandora/pod"
	"strings"
	"time"

	"path"

	"strconv"

	"os"

	cp "github.com/fatih/color"
	"github.com/go-hayden-base/str"
)

func startSync(args []string) {
	println("准备数据...")
	m, e := readExistKeys()
	if e != nil {
		cp.Red(e.Error())
		return
	}

	println("开始索引Pod...")
	podRoot := config["pod_path"]
	repo := config["pod_repos"]
	tc, ok := config["spec_read_thread"]
	threadCount := __SPEC_THREAD_MIN
	if ok {
		tmp, err := strconv.Atoi(tc)
		if err == nil {
			if tmp > __SPEC_THREAD_MAX {
				threadCount = __SPEC_THREAD_MAX
			} else if tmp > __SPEC_THREAD_MIN {
				threadCount = tmp
			}
		}
	}
	repos := strings.Split(repo, ",")
	p, success, failure, e := pod.Index(podRoot, repos, m, threadCount, true)
	if e != nil {
		cp.Red(e.Error())
		return
	}

	if len(p.PodRepos) == 0 {
		println("暂时没有需要更新的Pod，请尝试执行pod update更新指定仓库后在尝试索引!")
		return
	}
	println("解析成功：" + strconv.Itoa(success) + " 解析失败：" + strconv.Itoa(failure))

	println("开始同步到数据库...")
	rc, dc := syncToDB(p)
	println("更新模块：" + strconv.Itoa(rc) + " 子模块和依赖：" + strconv.Itoa(dc))
}

func readExistKeys() (map[string]bool, error) {
	var rows *sql.Rows
	rows, e := db.Query(_QUERY_ITEM_SQL)
	if e != nil {
		return nil, e
	}

	m := make(map[string]bool)
	for rows.Next() {
		var key string
		e = rows.Scan(&key)
		if e != nil {
			return nil, e
		}
		if len(key) > 0 {
			m[key] = true
		}
	}
	return m, nil
}

const __STR_DB_UNQ_ERR = `UNIQUE constraint failed:`

func syncToDB(p *pod.Pod) (int, int) {
	tx, e := db.Begin()
	if e != nil {
		printRed(e.Error(), false)
		return 0, 0
	}
	repoStmt, e := tx.Prepare(_INSERT_REPO_SQL)
	if e != nil {
		printRed(e.Error(), false)
		return 0, 0
	}
	defer repoStmt.Close()
	dependencyStmt, e := tx.Prepare(_INSERT_DP_SQL)
	if e != nil {
		printRed(e.Error(), false)
		return 0, 0
	}
	defer dependencyStmt.Close()
	logStmt, e := tx.Prepare(_INSERT_LOG_SQL)
	if e != nil {
		printRed(e.Error(), false)
		return 0, 0
	}
	defer logStmt.Close()

	currentTime := time.Now()
	var repocount, dpcount int
	joinPod(p, func(k string, r string, m string, v string, p string, s string, spec *pod.Spec) {
		_, e := repoStmt.Exec(k, r, m, v, p, s, currentTime)
		if e != nil {
			if strings.HasPrefix(e.Error(), __STR_DB_UNQ_ERR) {
				println("Warn: 重复主键 { key: " + k + ", path: " + p + " }")
				return
			} else {
				cp.Red(e.Error())
				tx.Rollback()
				os.Exit(1)
			}
		} else {
			repocount++
		}
		if spec == nil {
			return
		}
		joinSpec(spec, "", func(mm string, dd string, vv string) {
			_, e := dependencyStmt.Exec(k, mm, dd, vv, currentTime)
			if e != nil {
				if strings.HasPrefix(e.Error(), __STR_DB_UNQ_ERR) {
					println("Warn: 重复主键 { key: " + k + ", submodule: " + mm + ", dependency: " + dd + ", path: " + p + " }")
				} else {
					cp.Red(e.Error())
					tx.Rollback()
					os.Exit(1)
				}
			} else {
				dpcount++
			}
		})
	})

	_, e = logStmt.Exec(currentTime)
	if e != nil {
		printRed(e.Error(), false)
		tx.Rollback()
		return 0, 0
	}

	tx.Commit()
	return repocount, dpcount
}

type funcJoinPodCallback func(k string, r string, m string, v string, p string, s string, spec *pod.Spec)

func joinPod(p *pod.Pod, f funcJoinPodCallback) {
	if p == nil || f == nil {
		return
	}
	for _, repo := range p.PodRepos {
		for _, module := range repo.Modules {
			for _, version := range module.Versions {
				key := str.MD5(version.Root)
				specPath := path.Join(version.Root, version.FileName)
				var src string
				var spec *pod.Spec
				if version.Podspec != nil && version.Podspec.Source != nil {
					src = version.Podspec.Source.Git
				}
				if version.Err != nil {
					println("Warn: 解析失败->" + specPath + " 原因->" + version.Err.Error())
				} else {
					spec = version.Podspec
				}
				f(key, repo.Name, module.Name, version.Name, specPath, src, spec)
			}
		}
	}
}

type funcJoinSpecCallback func(m string, d string, v string)

func joinSpec(aSpec *pod.Spec, parent string, f funcJoinSpecCallback) {
	if aSpec == nil || f == nil {
		return
	}
	mm := aSpec.Name
	if len(parent) > 0 {
		mm = parent + "/" + mm
	}
	if aSpec.Dependences != nil {
		for n, _ := range aSpec.Dependences {
			v := aSpec.Dependences.Version(n)
			f(mm, n, v)
		}
	}
	subspec := aSpec.Subspecs
	if len(subspec) == 0 {
		return
	}
	for _, aTmpSpec := range subspec {
		joinSpec(aTmpSpec, mm, f)
	}
}

const _INSERT_REPO_SQL = `
INSERT INTO repo (key, repo, module, version, path, source, ctime)
VALUES (?, ?, ?, ?, ?, ?, ?)
`

const _INSERT_DP_SQL = `
INSERT INTO dependency (key, sub_module, dependency, version, ctime) VALUES (?,?,?,?,?)
`

const _INSERT_LOG_SQL = `
INSERT INTO updatelog (sync_time) VALUES (?)
`

const _QUERY_ITEM_SQL = `SELECT key FROM repo`
