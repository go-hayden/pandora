package main

import (
	"database/sql"
	"os"
	"pandora/pod"
	"path"
	"strconv"
	"strings"
	"time"

	cp "github.com/fatih/color"
	"github.com/go-hayden-base/str"
)

func cmd_sync(args *Args) {
	println("准备数据...")
	excluedKeyMap, e := readExistKeys()
	if e != nil {
		cp.Red(e.Error())
		return
	}

	excludeModuleMap := readExcluedModule()
	constraintMap := readConstraintMap()

	filerFunc := func(p string, level pod.PodLevel) bool {
		switch level {
		case pod.ENUM_POD_LEVEL_MODULE:
			mn := path.Base(p)
			rn := path.Base(path.Dir(p))
			if r, ok := excludeModuleMap[rn]; ok {
				if _, ok := r[mn]; ok {
					printYellow("忽略模块: "+rn+"/"+mn, false)
					return true
				}
			}
		case pod.ENUM_POD_LEVEL_VERSION:
			md5 := str.MD5(p)
			if _, ok := excluedKeyMap[md5]; ok {
				return true
			}
			if constraintMap != nil {
				tmp := p
				v := path.Base(tmp)
				tmp = path.Dir(tmp)
				mn := path.Base(tmp)
				tmp = path.Dir(tmp)
				rn := path.Base(tmp)
				if constraint, ok := constraintMap[rn]; ok {
					if constraintStr, ok := constraint[mn]; ok && strings.TrimSpace(constraintStr) != "" {
						if !pod.MatchVersionConstraint(constraintStr, v) {
							return true
						}
					}
				}
			}
		}
		return false
	}

	repos := readRepos()
	println("开始索引Pod...")
	p, success, failure, e := pod.PodIndex(_Conf.PodRepoRoot, repos, _Conf.SpecThread, true, filerFunc)
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
	suc, fail := syncToDB(p)
	println("同步数据： 成功 " + strconv.Itoa(suc) + " 条， 失败 " + strconv.Itoa(fail) + " 条")
}

// ** 前期数据 **
func readExistKeys() (map[string]bool, error) {
	var rows *sql.Rows
	rows, e := _DB.Query(_SQL_QUERY_EXIST_KEY)
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

func readExcluedModule() map[string]map[string]bool {
	res := make(map[string]map[string]bool)
	for _, repo := range _Conf.PodRepos {
		r, ok := res[repo.Name]
		if !ok {
			r = make(map[string]bool)
			res[repo.Name] = r
		}
		for _, m := range repo.Exclude {
			r[m] = true
		}
	}
	return res
}

func readConstraintMap() map[string]map[string]string {
	res := make(map[string]map[string]string)
	for _, repo := range _Conf.PodRepos {
		res[repo.Name] = repo.Constraints
	}
	return res
}

func readRepos() []string {
	res := make([]string, 0, len(_Conf.PodRepos))
	for _, repo := range _Conf.PodRepos {
		res = append(res, repo.Name)
	}
	return res
}

const __STR_DB_UNQ_ERR = `UNIQUE constraint failed:`

func syncToDB(p *pod.Pod) (int, int) {
	tx, e := _DB.Begin()
	if e != nil {
		printRed(e.Error(), false)
		return 0, 0
	}
	repoStmt, e := tx.Prepare(_SQL_INSERT_REPO)
	if e != nil {
		printRed(e.Error(), false)
		return 0, 0
	}
	defer repoStmt.Close()

	logStmt, e := tx.Prepare(_SQLINSERT_LOG)
	if e != nil {
		printRed(e.Error(), false)
		return 0, 0
	}
	defer logStmt.Close()

	currentTime := time.Now()
	var suc, fail int
	joinPod(p, func(k string, r string, m string, v string, p string, spec *pod.Spec) {
		json := ""
		if spec != nil {
			if b, e := spec.JSON(); e == nil && b != nil {
				json = string(b)
			}
		}
		if _, e := repoStmt.Exec(k, r, m, v, p, json, currentTime); e != nil {
			if strings.HasPrefix(e.Error(), __STR_DB_UNQ_ERR) {
				println("Warn: 重复主键 { key: " + k + ", path: " + p + " }")
				return
			} else {
				cp.Red(e.Error())
				tx.Rollback()
				os.Exit(1)
			}
			fail++
		} else {
			suc++
		}
	})

	_, e = logStmt.Exec(currentTime)
	if e != nil {
		printRed(e.Error(), false)
		tx.Rollback()
		return 0, 0
	}

	tx.Commit()
	return suc, fail
}

type funcJoinPodCallback func(k string, r string, m string, v string, p string, spec *pod.Spec)

func joinPod(p *pod.Pod, f funcJoinPodCallback) {
	if p == nil || f == nil {
		return
	}
	for _, repo := range p.PodRepos {
		for _, module := range repo.Modules {
			for _, version := range module.Versions {
				key := str.MD5(version.Root)
				specPath := path.Join(version.Root, version.FileName)
				var spec *pod.Spec
				if version.Err != nil {
					println("Warn: 解析失败->" + specPath + " 原因->" + version.Err.Error())
				} else {
					spec = version.Podspec
				}
				f(key, repo.Name, module.Name, version.Name, specPath, spec)
			}
		}
	}
}
