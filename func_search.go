package main

import (
	"bytes"
	"database/sql"
	"errors"
	"strings"

	"encoding/json"

	"fmt"

	fdt "github.com/go-hayden-base/foundation"
	"github.com/go-hayden-base/pod"
	ver "github.com/go-hayden-base/version"
)

func cmd_depend(aArgs *Args) {
	module, version, err := moduleAndVersion(aArgs)
	if err != nil {
		printRed(err.Error(), true)
		return
	}

	if version == "" {
		v, err := queryNewestVersionWithConstraint(module, "")
		if err != nil {
			printRed(err.Error(), false)
			return
		}
		version = v
	}

	// 打印依赖列表
	deps, repoList, err := queryDepends(module, version)
	if err != nil {
		printRed(err.Error(), false)
		return
	}
	if len(deps) == 0 {
		println("未查询到依赖!")
		return
	}
	println("-> 仓库: " + repoList + "  模块: " + module + "  版本: " + version)
	for _, aDep := range deps {
		println("   - " + aDep.N + "  " + aDep.V)
	}
	println("")
}

func cmd_origin(aArgs *Args) {
	module, version, err := moduleAndVersion(aArgs)
	if err != nil {
		printRed(err.Error(), true)
		return
	}

	if version == "" {
		v, err := queryNewestVersionWithConstraint(module, "")
		if err != nil {
			printRed(err.Error(), false)
			return
		}
		version = v
	}

	specJson, repoList, err := queryOriginalSpec(module, version)
	if err == nil {
		println("-> 仓库: " + repoList + "  模块: " + module + "  版本: " + version)
		buffer := new(bytes.Buffer)
		if err = json.Indent(buffer, []byte(specJson), "", "  "); err != nil {
			println(specJson)
		} else {
			println(buffer.String())
		}

	} else {
		printRed(err.Error(), false)
	}
	println("")
}

func cmd_exec_sql(aArgs *Args) {
	sql := aArgs.GetFirstSubArgsMain()
	if sql == "" {
		printRed("参数错误!", true)
		return
	}
	printGreen("执行SQL: "+sql, false)
	if r, e := _DB.Exec(sql); e != nil {
		printRed(e.Error(), false)
	} else {
		println("执行成功!")
		fmt.Println(r)
	}
}

func queryDepends(module string, version string) ([]*pod.DependBase, string, error) {
	specJson, repoList, err := queryOriginalSpec(module, version)
	if err != nil {
		return nil, "", err
	}
	spec, err := pod.NewSpecWithJSONString(specJson)
	if err != nil {
		return nil, "", err
	}
	deps := spec.GetAllDepends(module)
	l := len(deps)
	if l == 0 {
		return nil, "", nil
	}
	cap := l/2 + 1
	resHead := make([]*pod.DependBase, 0, cap)
	resFoot := make([]*pod.DependBase, 0, cap)
	baseModule := fdt.StrSplitFirst(module, "/") + "/"

	for keya, _ := range deps {
		if !strings.HasPrefix(keya, baseModule) {
			continue
		}
		drop := false
		for keyb, _ := range deps {
			if keyb == keya || !strings.HasPrefix(keyb, baseModule) {
				continue
			}
			if strings.HasPrefix(keyb, keya) {
				drop = true
			}
		}
		if drop {
			delete(deps, keya)
		}
	}

	level := len(strings.Split(module, "/"))
	for key, val := range deps {
		aDep := pod.DependBase{N: key, V: val}
		if strings.HasPrefix(key, baseModule) {
			if level < 2 {
				continue
			}
			resHead = append(resHead, &aDep)
		} else {
			resFoot = append(resFoot, &aDep)
		}
	}
	return append(resHead, resFoot...), repoList, nil
}

func queryOriginalSpec(module, version string) (string, string, error) {
	if module == "" || version == "" {
		return "", "", errors.New("模块名和版本号不能为空！")
	}
	baseModule := fdt.StrSplitFirst(module, "/")
	var rows *sql.Rows
	var e error
	if rows, e = _DB.Query(_SQL_QUERY_SPEC, baseModule, version); e != nil {
		return "", "", e
	}
	var buffer bytes.Buffer
	var specJson string
	for rows.Next() {
		var repo, jsonString string
		if e = rows.Scan(&repo, &jsonString); e != nil {
			continue
		}
		buffer.WriteString("[" + repo + "]")
		if specJson == "" && jsonString != "" {
			specJson = jsonString
		}
	}
	return specJson, buffer.String(), nil
}

func queryNewestVersionWithConstraint(module string, constraint string) (string, error) {
	versions, e := queryVersionsWithConstraint(module, constraint)
	if e != nil {
		return "", e
	}
	return ver.MaxVersion("", versions...)
}

func queryNewestVersionWithConstraints(module string, constraints []string) (string, error) {
	versions, e := queryVersionsWithConstraints(module, constraints)
	if e != nil {
		return "", e
	}
	return ver.MaxVersion("", versions...)
}

func queryVersionsWithConstraint(module string, contraint string) ([]string, error) {
	if contraint == "" {
		return queryVersionsWithConstraints(module, nil)
	} else {
		return queryVersionsWithConstraints(module, []string{contraint})
	}
}

func queryVersionsWithConstraints(module string, contraints []string) ([]string, error) {
	baseName := fdt.StrSplitFirst(module, "/")
	rows, e := _DB.Query(_SQL_QUERY_VERSIONS, baseName)
	if e != nil {
		return nil, e
	}
	m := make(map[string]bool)
	for rows.Next() {
		var v string
		e = rows.Scan(&v)
		if e != nil {
			continue
		}
		if len(contraints) > 0 && !ver.MatchVersionConstrains(contraints, v) {
			continue
		}
		m[v] = true
	}
	res := make([]string, 0, len(m))
	for v := range m {
		res = append(res, v)
	}
	return res, nil
}

func versionify(module, version string) string {
	if ver.IsVersion(version) {
		return version
	}

	newVersion, err := queryNewestVersionWithConstraint(module, version)
	if err != nil {
		return ""
	}
	return newVersion
}

func moduleAndVersion(aArgs *Args) (string, string, error) {
	args := aArgs.GetSubargsMain()
	l := len(args)
	if l < 1 {
		return "", "", errors.New("参数错误!")
	} else if l < 2 {
		return args[0], "", nil
	}
	return args[0], args[1], nil
}
