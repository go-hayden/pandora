package main

const __search_depend_sql = `
SELECT a.repo as repo, b.sub_module as submodule, b.dependency as dependency, b.version as version from (
    SELECT key, repo from repo WHERE module=? AND version=?
) a INNER JOIN dependency b ON a.key = b.key
`

func depend(args []string) {
	l := len(args)
	if l < 2 {
		println("参数错误!")
		printHelp()
		return
	}
	module := args[0]
	ver := args[1]

	rows, e := db.Query(__search_depend_sql, module, ver)
	if e != nil {
		printRed(e.Error(), false)
		return
	}
	res := make(map[string]map[string][]string)
	hasdata := false
	for rows.Next() {
		hasdata = true
		var repo, submodule, dependency, version string
		rows.Scan(&repo, &submodule, &dependency, &version)
		submap, ok := res[repo]
		if ok {
			submodules, ok := submap[submodule]
			if ok {
				submodules = append(submodules, dependency+" "+version)
			} else {
				submodules = make([]string, 1, 6)
				submodules[0] = dependency + " " + version
			}
			submap[submodule] = submodules
		} else {
			submap = make(map[string][]string)
			submodules := make([]string, 1, 6)
			submodules[0] = dependency + " " + version
			submap[submodule] = submodules
			res[repo] = submap
		}
	}
	if !hasdata {
		printRed("版本为 "+ver+" 的模块 "+module+" 没有查询到依赖!", false)
		return
	}
	for repo, submap := range res {
		println("\n-> 仓库：" + repo + "  模块：" + module + "  版本：" + ver)
		for subm, dps := range submap {
			println("    - 子模块：" + subm)
			for _, dp := range dps {
				println("        " + dp)
			}
		}
	}
	println("")
}
