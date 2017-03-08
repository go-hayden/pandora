package main

import (
	"database/sql"
	"os"
	"path"
	"strings"
	"tb/str"
	"time"

	cp "github.com/fatih/color"
	fdt "github.com/go-hayden-base/foundation"
	"github.com/go-hayden-base/pod"
	ver "github.com/go-hayden-base/version"
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
						if !ver.MatchVersionConstraint(constraintStr, v) {
							return true
						}
					}
				}
			}
		}
		return false
	}

	repos := readRepos()
	println("开始索引Pod ...")
	aPod, e := pod.PodIndex(_Conf.PodRepoRoot, repos, filerFunc)
	if e != nil {
		cp.Red(e.Error())
		return
	}
	if len(aPod.PodRepos) == 0 {
		println("暂时没有需要更新的Pod，请尝试执行pod update更新指定仓库后在尝试索引!")
		return
	}
	all := 0
	resolveSuccess := 0
	resolveLogFunc := func(success bool, msg string) {
		all++
		if success {
			resolveSuccess++
			println(msg)
		} else {
			printRed(msg, false)
		}
	}

	current := time.Now()
	saveSuccess := 0
	saveFunc := func(specs []*pod.Spec) {
		saveSuccess += save(specs, &current)
	}

	println("开始解析Podspec ...")
	pod.ResolvePodSpecs(aPod, _Conf.SpecThread, saveFunc, resolveLogFunc)

	resolveFail := all - resolveSuccess
	saveFail := resolveSuccess - saveSuccess

	println("\n====== Summary ======")
	println("  解析总数:", all)
	println("  解析成功:", resolveSuccess)
	println("  解析失败:", resolveFail)
	println("  同步成功:", saveSuccess)
	println("  同步失败:", saveFail)
	println("=====================\n")
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

func save(specs []*pod.Spec, current *time.Time) int {
	tx, e := _DB.Begin()
	if e != nil {
		printRed(e.Error(), false)
		return 0
	}
	repoStmt, e := tx.Prepare(_SQL_INSERT_REPO)
	if e != nil {
		printRed(e.Error(), false)
		return 0
	}
	defer repoStmt.Close()

	logStmt, e := tx.Prepare(_SQLINSERT_LOG)
	if e != nil {
		printRed(e.Error(), false)
		return 0
	}
	defer logStmt.Close()
	suc := 0
	for _, aSpec := range specs {
		k, r, m, v := flattenSpec(aSpec)
		json := ""
		if b, e := aSpec.JSON(); e == nil && b != nil {
			json = string(b)
		}
		if _, e := repoStmt.Exec(k, r, m, v, aSpec.FilePath, json, current); e != nil {
			if strings.HasPrefix(e.Error(), __STR_DB_UNQ_ERR) {
				println("Warn: 重复主键 { key: " + k + ", path: " + aSpec.FilePath + " }")
				continue
			} else {
				cp.Red(e.Error())
				tx.Rollback()
				os.Exit(1)
			}
		} else {
			suc++
		}
	}

	_, e = logStmt.Exec(current)
	if e != nil {
		printRed(e.Error(), false)
		tx.Rollback()
		return 0
	}

	tx.Commit()
	return suc
}

func flattenSpec(aSpec *pod.Spec) (string, string, string, string) {
	root := path.Dir(aSpec.FilePath)
	k := fdt.StrMD5(root)
	v := path.Base(root)
	root = path.Dir(root)
	m := path.Base(root)
	root = path.Dir(root)
	r := path.Base(root)
	return k, r, m, v
}
