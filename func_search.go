package main

import (
	"errors"
	"pandora/pod"

	"bytes"

	"database/sql"

	"strings"

	ver "github.com/hashicorp/go-version"
)

func cmd_depend(aArgs *Args) {
	args := aArgs.Values
	l := len(args)
	if l < 2 {
		println("参数错误!")
		printHelp()
		return
	}
	module := args[0]
	version := args[1]

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

func queryDepends(module string, version string) ([]*pod.DependBase, string, error) {
	if module == "" || version == "" {
		return nil, "", errors.New("模块名和版本号不能为空！")
	}
	baseModule := pod.BaseModule(module)
	var rows *sql.Rows
	var e error
	if rows, e = _DB.Query(_SQL_QUERY_SPEC, baseModule, version); e != nil {
		return nil, "", e
	}
	var buffer bytes.Buffer
	var spec *pod.Spec
	for rows.Next() {
		var repo, jsonString string
		if e = rows.Scan(&repo, &jsonString); e != nil {
			continue
		}
		buffer.WriteString("[" + repo + "]")
		if spec == nil && jsonString != "" {
			spec, _ = pod.NewSpecWithJSONString(jsonString)
		}
	}
	if spec == nil {
		return nil, "", nil
	}
	deps := spec.GetAllDepends(module)
	l := len(deps)
	if l == 0 {
		return nil, "", nil
	}
	cap := l/2 + 1
	resHead := make([]*pod.DependBase, 0, cap)
	resFoot := make([]*pod.DependBase, 0, cap)
	baseModule += "/"
	for key, val := range deps {
		aDep := pod.DependBase{N: key, V: val}
		if strings.HasPrefix(key, baseModule) {
			resHead = append(resHead, &aDep)
		} else {
			resFoot = append(resFoot, &aDep)
		}
	}
	return append(resHead, resFoot...), buffer.String(), nil
}

func queryNewestVersion(module string, constraint string) (string, error) {
	versions, e := queryVersion(module, constraint)
	if e != nil {
		return "", e
	}
	return pod.MaxVersion("", versions...)
}

func queryVersion(module string, contraint string) ([]string, error) {
	var aConstraint ver.Constraints
	if len(contraint) > 0 {
		aConstraint, _ = ver.NewConstraint(contraint)
	}
	rows, e := _DB.Query(_SQL_QUERY_VERSIONS, pod.BaseModule(module))
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
		if aConstraint != nil {
			aVer, e := ver.NewVersion(v)
			if e != nil || !aConstraint.Check(aVer) {
				continue
			}
		}
		m[v] = true
	}
	res := make([]string, 0, len(m))
	for v, _ := range m {
		res = append(res, v)
	}
	return res, nil
}
