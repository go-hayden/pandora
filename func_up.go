package main

import (
	"bytes"
	"os"
	"pandora/pod"
	"path"
	"strconv"
	"strings"
	"time"
)

func cmd_upgrade(aArgs *Args) {
	podfilePaths := aArgs.GetSubargsMain()
	if len(podfilePaths) < 1 {
		exitWithMessage("参数错误！", true)
	}
	flatten := aArgs.CheckSubargs("--flatten")
	merge := aArgs.CheckSubargs("--merge_mode")

	// 尝试转换相对路径为绝对路径
	for i := 0; i < len(podfilePaths); i++ {
		p := podfilePaths[i]
		podfilePaths[i] = AbsPath("", p)
	}

	// 生成Podfile对象
	podfiles := make([]*pod.Podfile, 0, len(podfilePaths))
	for _, podfilePath := range podfilePaths {
		printGreen("开始解析Podfile: "+podfilePath, false)
		if aPodfile, err := buildPodfile(podfilePath); err != nil {
			exitWithMessage(err.Error(), true)
		} else {
			pod.FillPodfile(aPodfile, _Conf.SpecThread, true)
			podfiles = append(podfiles, aPodfile)
		}
	}

	// 解析升级标杆
	var aRulePodfile *pod.Podfile
	var err error
	if aArgs.CheckSubargs("--rules") {
		rules := aArgs.GetSubargs("--rules")
		printGreen("解析升级列表: "+strings.Join(rules, ","), false)
		aRulePodfile, err = buildPodfile(rules)
		if err != nil {
			exitWithMessage("解析升级列表失败！", false)
		}
	} else if aArgs.CheckSubargs("--rule") {
		rule := aArgs.GetFirstSubArgs("--rule")
		printGreen("解析标杆Podfile: "+rule, false)
		aRulePodfile, err = buildPodfile(rule)
		if err != nil {
			exitWithMessage("解析标杆Podfile失败！", false)
		}
	}

	graphPodfiles := make([]pod.GraphPodfile, 0, len(podfiles))
	for _, aPodfile := range podfiles {
		printGreen("开始分析Podfile: "+aPodfile.FilePath, false)
		aGraphPodfile := buildGraphPodfiles(aPodfile, aRulePodfile, merge)
		graphPodfiles = append(graphPodfiles, aGraphPodfile)
	}
	printGreen("开始提取公共依赖 ...", false)
	intersection(graphPodfiles...)

	printGreen("开始输出文件 ...", false)
	date := time.Now()
	for idx, aGraphPodfile := range graphPodfiles {
		aPodfile := podfiles[idx]
		filePath := generateWritePath(aPodfile.FilePath, &date)
		printGreen("生成分析表格 ...", false)
		writeGraphPodfile(filePath, aGraphPodfile, flatten)
		printGreen("生成整体依赖关系图 ...", false)
		writeGraph(filePath, aGraphPodfile)
		printGreen("生成依赖关系最大连通图 ...", false)
		connectedGraphs := writeMaxConnectedGraph(filePath, aGraphPodfile)
		if connectedGraphs != nil && len(connectedGraphs) > 0 {
			printGreen("生成依赖关系图最小生成树 ...", false)
			writeMinimalSpanningTree(filePath, connectedGraphs)
		}
	}
}

// 交集
func intersection(graphPodfiles ...pod.GraphPodfile) {
	if len(graphPodfiles) < 2 {
		return
	}
	commom := make(map[string]bool)
	firstGP := graphPodfiles[0]
	otherGPs := graphPodfiles[1:]
	for dpn, _ := range firstGP {
		isCommon := true
		for _, aGP := range otherGPs {
			_, ok := aGP[dpn]
			if !ok {
				isCommon = false
			}
			if !isCommon {
				break
			}
		}
		if isCommon {
			commom[dpn] = true
		}
	}
	for _, aGP := range graphPodfiles {
		for dpn, module := range aGP {
			_, ok := commom[dpn]
			module.IsCommon = ok
		}
	}
}

func buildPodfile(src interface{}) (*pod.Podfile, error) {
	var aPodfile *pod.Podfile
	var err error
	switch src.(type) {
	case string:
		podfilePath := src.(string)
		aPodfile, err = pod.NewPodfile(podfilePath, true)
	case []interface{}:
		rules := src.([]interface{})
		aPodfile = new(pod.Podfile)
		aTarget := new(pod.Target)
		aTarget.Depends = make([]*pod.Depend, 0, 5)
		aPodfile.Targets = []*pod.Target{aTarget}
		for _, item := range rules {
			itemString, ok := item.(string)
			if !ok {
				continue
			}
			items := strings.Split(itemString, ":")
			module := items[0]
			var version string
			if len(items) > 1 {
				version = items[1]
			}
			aDepend := new(pod.Depend)
			aDepend.N = module
			aDepend.V = version
			aTarget.Depends = append(aTarget.Depends, aDepend)
		}
	}
	return aPodfile, err
}

func buildGraphPodfiles(aPodfile *pod.Podfile, aRulePodfile *pod.Podfile, isMergeMode bool) pod.GraphPodfile {
	aGraphPodfile := make(pod.GraphPodfile)
	for _, aTarget := range aPodfile.Targets {
		for _, aDepend := range aTarget.Depends {
			_, ok := aGraphPodfile[aDepend.Name()]
			if ok {
				continue
			}
			aModule := buildGraphModule(aDepend, aRulePodfile)
			aGraphPodfile[aModule.Name] = aModule
		}
	}
	if isMergeMode && aRulePodfile != nil {
		aRulePodfile.EnumerateAllDepends(func(target, depend, version string) {
			if _, ok := aGraphPodfile[depend]; ok {
				return
			}
			aDependBase := pod.DependBase{N: depend, V: version}
			aGraphModule := buildGraphModule(&aDependBase, nil)
			aGraphModule.IsNew = true
			aGraphPodfile[depend] = aGraphModule
		})
	}
	check(aPodfile.FilePath, aGraphPodfile, aRulePodfile, 1)
	return aGraphPodfile
}

func buildGraphModule(aIDepend pod.IDepend, aRulePodfile *pod.Podfile) *pod.GraphModule {
	aGraphModule := new(pod.GraphModule)
	var ruleVersion, newestVersion string

	// 获取标杆版本
	if aRulePodfile != nil {
		var exist bool
		ruleVersion, exist = aRulePodfile.GetDependVersion(nil, aIDepend.Name())
		if exist {
			ruleVersion = realVesion(aIDepend.Name(), ruleVersion)
		}
	}

	// 获取最新版本
	newestVersion, _ = queryNewestVersion(aIDepend.Name(), "")

	// 获取实际的当前版本
	if pod.IsVersion(aIDepend.Version()) {
		aGraphModule.Version = aIDepend.Version()
	} else if aIDepend.Version() == "" {
		aGraphModule.Version = newestVersion
	} else {
		aGraphModule.Version, _ = queryNewestVersion(aIDepend.Name(), aIDepend.Version())
	}

	aGraphModule.Name = aIDepend.Name()
	aGraphModule.UpdateToVersion = ruleVersion
	aGraphModule.NewestVersion = newestVersion
	aGraphModule.IsLocal = aIDepend.IsLocal()

	// 优先获取远端依赖
	if aGraphModule.Version != "" {
		if depends, _, err := queryDepends(aGraphModule.Name, aGraphModule.UseVersion()); err == nil && len(depends) > 0 {
			aGraphModule.Depends = depends
		}
	}
	if len(aGraphModule.Depends) == 0 && aIDepend.Subdepends() != nil && aGraphModule.UseVersion() == aGraphModule.Version {
		aGraphModule.Depends = aIDepend.Subdepends()
	}

	return aGraphModule
}

func check(podfilePath string, aGraphPodfile pod.GraphPodfile, aRulePodfile *pod.Podfile, times int) {
	println("分析隐性依赖[第" + strconv.Itoa(times) + "次迭代]: " + podfilePath)
	if aGraphPodfile == nil {
		return
	}

	unfoundDepends := aGraphPodfile.Check()
	if len(unfoundDepends) == 0 {
		return
	}
	var buffer bytes.Buffer
	for _, aDepend := range unfoundDepends {
		if _Conf.IsDebug() {
			buffer.WriteString("[")
			buffer.WriteString(aDepend.Name())
			if aDepend.Version() != "" {
				buffer.WriteString(":" + aDepend.Version())
			}
			buffer.WriteString("] ")
		}
		oldDepend, ok := aGraphPodfile[aDepend.Name()]
		if ok {
			v, e := queryNewestVersion(aDepend.Name(), aDepend.Version())
			if aDepend.Name() == "NVStyle" {
				println(aDepend.Version(), oldDepend.UseVersion())
				println(pod.MatchVersionConstraint(aDepend.Version(), oldDepend.UseVersion()))
				os.Exit(1)
			}
			if e != nil || v == "" {
				oldDepend.UpdateToVersion = "*"
			} else {
				oldDepend.UpdateToVersion = v
				depends, _, e := queryDepends(oldDepend.Name, oldDepend.UseVersion())
				if e == nil {
					oldDepend.Depends = depends
				}
			}
		} else {
			aModule := buildGraphModule(aDepend, aRulePodfile)
			aModule.IsImplicit = true
			aModule.IsNew = true
			aGraphPodfile[aModule.Name] = aModule
		}
	}
	if _Conf.IsDebug() {
		println(buffer.String())
	}
	check(podfilePath, aGraphPodfile, aRulePodfile, times+1)
}

func realVesion(module string, version string) string {
	if pod.IsVersion(version) {
		return version
	}
	if version == "" {
		v, e := queryNewestVersion(module, "")
		if e != nil {
			return ""
		}
		return v
	}
	v, e := queryNewestVersion(module, version)
	if e != nil {
		return ""
	}
	return v
}

func generateWritePath(filePath string, date *time.Time) string {
	name := podfileName(filePath)
	if name == "" {
		name = string(RandomString(32, KC_RAND_KIND_ALL))
	}

	if date == nil {
		x := time.Now()
		date = &x
	}
	directoryName := "up_" + date.Format("20060102150405")
	return path.Join(_Conf.OutputDirectory, directoryName, name)
}

func writeGraphPodfile(filePath string, aGraphPodfile pod.GraphPodfile, isFlatten bool) {
	if filePath == "" {
		printRed("输出路径不能为空！", false)
		return
	}

	csvFilePath := filePath
	ext := path.Ext(csvFilePath)
	if ext != ".csv" {
		csvFilePath += ".csv"
	}
	var buffer bytes.Buffer
	if _, e := buffer.WriteString("ModuleName,IsCommon,IsNew,IsImplicit,IsLocal,Current,UpgradeTo,UpgradeTag,Newest,Dependencies\n"); e != nil {
		printRed(e.Error(), false)
		return
	}
	if aGraphPodfile != nil {
		if _, e := buffer.Write(aGraphPodfile.Bytes()); e != nil {
			printRed(e.Error(), false)
			return
		}
	}
	if e := WriteFile(csvFilePath, buffer.Bytes(), true, os.ModePerm); e != nil {
		printRed("输出文件错误["+csvFilePath+"]: "+e.Error(), false)
	} else {
		println("输出文件: " + csvFilePath)
	}

	// 是否输出辅助文件
	if !isFlatten {
		return
	}

	bufferCommon := new(bytes.Buffer)
	bufferCommon.WriteString("# Common\n")
	bufferRemote := new(bytes.Buffer)
	bufferRemote.WriteString("\n# Remote\n")
	bufferLocal := new(bytes.Buffer)
	bufferLocal.WriteString("\n# Local\n")
	var bufferSwitch *bytes.Buffer
	for _, module := range aGraphPodfile {
		if module.IsCommon {
			bufferSwitch = bufferCommon
		} else if module.IsLocal {
			bufferSwitch = bufferLocal
		} else {
			bufferSwitch = bufferRemote
		}

		bufferSwitch.WriteString("'" + module.Name + "'")
		useVersion := module.UseVersion()
		if useVersion != "" {
			bufferSwitch.WriteString(", '" + useVersion + "'")
		}
		if module.IsLocal {
			bufferSwitch.WriteString(" # Local")
		}
		bufferSwitch.WriteString("\n")
	}
	bufferCommon.Write(bufferRemote.Bytes())
	bufferCommon.Write(bufferLocal.Bytes())
	if e := WriteFile(filePath, bufferCommon.Bytes(), true, os.ModePerm); e != nil {
		printRed("输出文件失败: "+filePath, false)
	} else {
		println("输出文件: " + filePath)
	}
}

func writeGraph(filePath string, aGraphPodfile pod.GraphPodfile) {
	if _Conf.Graph == nil {
		printRed("未提供关系图模板，不能生成关系图！", false)
		return
	}

	filePath = path.Join(filePath, "Graph")
	aCanvas := NewCanvas(filePath, _Conf.Graph.HtmlPath, _Conf.Graph.CSSPaths, _Conf.Graph.JSPaths)
	aGraphAll := new(Graph)
	aGraphAll.Name = "All"
	aCanvas.Append(aGraphAll)

	// 添加节点
	for key, _ := range aGraphPodfile {
		aGraphAll.AppendNode(key, "", "", "", 12)
	}

	// 添加路径
	for moduleName, aModule := range aGraphPodfile {
		for _, aDep := range aModule.Depends {
			to := aDep.Name()
			if _, ok := aGraphPodfile[to]; !ok {
				printRed("添加节点失败: "+moduleName+" -> "+aDep.Name(), false)
				return
			}
			if moduleName == to {
				continue
			}
			if e := aGraphAll.AppendEdge("", moduleName, to, "to", "", false); e != nil {
				printRed(e.Error(), false)
				return
			}
		}
	}

	if msg, e := aCanvas.Output(); e != nil {
		printRed(e.Error(), false)
	} else {
		println(msg)
	}
}

func writeMaxConnectedGraph(filePath string, aGraphPodfile pod.GraphPodfile) []*ConnectedGraph {
	if _Conf.Graph == nil {
		printRed("未提供关系图模板，不能生成关系图！", false)
		return nil
	}

	aNodeMap := make(map[string]INode)
	for key, val := range aGraphPodfile {
		aNodeMap[key] = val
	}
	connectedGraphs := ConnectedGraphs(aNodeMap)
	filePath = path.Join(filePath, "ConnectedGraph")
	aCanvas := NewCanvas(filePath, _Conf.Graph.HtmlPath, _Conf.Graph.CSSPaths, _Conf.Graph.JSPaths)
	for idx, aConnectedGraph := range connectedGraphs {
		aGraph := new(Graph)
		graphName := strings.Replace(aConnectedGraph.RootNode, "/", "_", -1)
		graphName = strings.Replace(graphName, "\\", "_", -1)
		aGraph.Name = strconv.Itoa(idx+1) + "." + graphName

		// 添加节点
		for moduleName, _ := range aConnectedGraph.NodeMap {
			aGraph.AppendNode(moduleName, "", "", "", 12)
		}

		// 添加路径
		for moduleName, aNode := range aConnectedGraph.NodeMap {
			for _, aDep := range aNode.ReferenceNodes() {
				to := aDep
				ok := aGraph.CheckNode(to)
				if ok {
					if moduleName == to {
						continue
					}
				} else {
					aGraph.AppendNode(aDep, "", _COLOR_RED, "", 12)
				}
				aGraph.AppendEdge("", moduleName, to, "to", "", false)
			}
		}
		aCanvas.Append(aGraph)
	}
	if msg, e := aCanvas.Output(); e != nil {
		printRed(e.Error(), false)
	} else {
		println(msg)
	}
	return connectedGraphs
}

func writeMinimalSpanningTree(filePath string, connectedGraphs []*ConnectedGraph) {
	if len(connectedGraphs) == 0 {
		return
	}
	if _Conf.Tree == nil {
		printRed("未提供树图模板，不能生成树图！", false)
		return
	}

	trees := make([]*Tree, 0, len(connectedGraphs))
	// 生成树
	for _, aConnectedGraph := range connectedGraphs {
		aTree, e := NewTreeWithConnectedGraph(aConnectedGraph)
		if e != nil {
			printRed("根节点为 "+aConnectedGraph.RootNode+" 转换为最小生成树发生错误: "+e.Error(), false)
			continue
		}
		trees = append(trees, aTree)
	}

	// 关联树
	for _, aTree := range trees {
		if aTree.NotReference == nil {
			continue
		}
		for _, aUnrefTree := range aTree.NotReference {
			var aRefRootTree *Tree
			var aRefTree *Tree
			for _, aSearchTree := range trees {
				if aSearchTree == aTree {
					continue
				}
				tmp := aSearchTree.TreeWithName(aUnrefTree.NodeName)
				if tmp != nil {
					aRefRootTree = aSearchTree
					aRefTree = tmp
					break
				}
			}
			if aRefTree != nil {
				aUnrefTree.Value = "Root[" + aRefRootTree.NodeName + "].Lv[" + strconv.Itoa(aRefTree.Level) + "]." + aRefTree.NodeName
			} else {
				aUnrefTree.Value = "NotFound"
			}
		}
	}

	filePath = path.Join(filePath, "MinimalSpanningTree")
	aCav := NewCanvas(filePath, _Conf.Tree.HtmlPath, _Conf.Tree.CSSPaths, _Conf.Tree.JSPaths)
	aSingleNodeGraph := new(Graph)
	aSingleNodeGraph.Name = "0.Single_Node_Tree"
	for idx, aRootTree := range trees {
		if aRootTree.Subtrees == nil || len(aRootTree.Subtrees) == 0 {
			aSingleNodeGraph.AppendNode(aRootTree.NodeName, "", "", "", 10)
			continue
		}
		aGraph := new(Graph)
		graphName := strconv.Itoa(idx+1) + "." + strings.Replace(aRootTree.NodeName, "/", "_", -1)
		aGraph.Name = graphName
		DepthFirstTraversalTree(aRootTree, func(aTTree *Tree) {
			nodeName := aTTree.LevelNodeName()
			nodeDesc := ""
			nodeColor := ""
			if aTTree.Reference != nil && len(aTTree.Reference) > 0 {
				nodeColor = _COLOR_YELLOW
				buffer := new(bytes.Buffer)
				buffer.WriteString(`<div style='text-align:left'>Reference:`)
				for _, aRefTree := range aTTree.Reference {
					buffer.WriteString(`<br/>&nbsp;-&nbsp;Lv[` + strconv.Itoa(aRefTree.Level) + "]." + aRefTree.NodeName)
				}
				buffer.WriteString("</div>")
				nodeDesc = buffer.String()
			} else if aTTree.Value != nil {
				if referenceRoot, ok := aTTree.Value.(string); ok {
					nodeColor = _COLOR_RED
					nodeDesc = "<div style='text-align:left'>Reference:<br/>&nbsp;-&nbsp;" + referenceRoot + "</div>"
				} else {
					nodeColor = _COLOR_LIGHT_GRAY
					nodeDesc = "Convert string failed!"
				}
			}
			if !aGraph.CheckNode(nodeName) {
				if e := aGraph.AppendNode(nodeName, nodeDesc, nodeColor, "", 10); e != nil {
					printRed("添加节点失败: "+nodeName, false)
					return
				}
			}
			if aTTree.Parent == nil {
				return
			}
			parentNodeName := aTTree.Parent.LevelNodeName()
			if !aGraph.CheckNode(parentNodeName) {
				if e := aGraph.AppendNode(parentNodeName, "", "", "", 10); e != nil {
					printRed("添加节点 "+aTTree.NodeName+" 的父节点 "+aTTree.Parent.NodeName+" 失败!", false)
					return
				}
			}
			if e := aGraph.AppendEdge("", parentNodeName, nodeName, "to", "", false); e != nil {
				printRed("添加从节点 "+aTTree.Parent.NodeName+" 到节点 "+aTTree.NodeName+" 失败!", false)
			}
		})
		aCav.Append(aGraph)
	}
	if aSingleNodeGraph.HasNode() {
		aCav.Append(aSingleNodeGraph)
	}
	if msg, e := aCav.Output(); e != nil {
		printRed("输出最小生成树发生错误: "+e.Error(), false)
	} else {
		println(msg)
	}
}

func podfileName(p string) string {
	n := path.Base(p)
	for index := 0; index < 2; index++ {
		p = path.Dir(p)
		b := path.Base(p)
		if b != "" {
			n = b + "_" + n
		} else {
			break
		}
	}
	return n
}
