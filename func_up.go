package main

import (
	"bytes"
	"pandora/pod"
	"strconv"
	"time"
)

import "path"

import "os"

func cmd_upgrade(aArgs *Args) {
	joinArgs := aArgs.GetSubargsMain()
	outTypeStr := aArgs.GetFirstSubArgs("--out_type")
	outType := 0
	if r, e := ParseBinaryString(outTypeStr); e == nil && r > -1 {
		outType = r
	}
	if len(joinArgs) < 1 {
		exitWithMessage("参数错误！", true)
	}
	for i := 0; i < len(joinArgs); i++ {
		p := joinArgs[i]
		joinArgs[i] = absolutePath(p)
	}

	flag := aArgs.GetFirstSubArgs("--flag")
	var upPodfile *pod.Podfile
	var err error
	if flag != "" {
		printGreen("开始解析Podfile: "+flag, false)
		upPodfile, err = pod.NewPodfile(flag, true)
		if err != nil {
			exitWithMessage(err.Error(), true)
		}
		pod.FillPodfile(upPodfile, _Conf.SpecThread, true)
	}

	joinPodfiles := make([]*pod.Podfile, 0, len(joinArgs))
	for _, pf := range joinArgs {
		printGreen("开始解析Podfile: "+pf, false)
		aPodfile, err := pod.NewPodfile(pf, true)
		if err != nil {
			exitWithMessage(err.Error(), true)
		}
		pod.FillPodfile(aPodfile, _Conf.SpecThread, true)
		joinPodfiles = append(joinPodfiles, aPodfile)
	}

	graphPodfiles := make([]pod.GraphPodfile, 0, len(joinPodfiles))
	for _, pf := range joinPodfiles {
		printGreen("开始分析Podfile: "+pf.FilePath, false)
		aGraphPodfile := buildGraphPodfiles(pf, upPodfile)
		graphPodfiles = append(graphPodfiles, aGraphPodfile)
	}
	printGreen("开始提取公共依赖 ...", false)
	intersection(graphPodfiles...)

	printGreen("开始输出文件 ...", false)
	date := time.Now()
	for idx, aGP := range graphPodfiles {
		aPF := joinPodfiles[idx]
		fp := generateWritePath(aPF.FilePath, &date)
		writeGraphPodfile(fp, aGP, outType)
	}
}

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

func buildGraphPodfiles(podfile *pod.Podfile, upPodfile *pod.Podfile) pod.GraphPodfile {
	graphPodfile := make(pod.GraphPodfile)
	for _, aTarget := range podfile.Targets {
		for _, aDep := range aTarget.Depends {
			_, ok := graphPodfile[aDep.Name()]
			if ok {
				continue
			}
			aModule := buildGraphModule(aDep, upPodfile)
			graphPodfile[aModule.Name] = aModule
		}
	}
	check(podfile.FilePath, graphPodfile, upPodfile, 1)
	return graphPodfile
}

func buildGraphModule(aDepend pod.IDepend, upPodfile *pod.Podfile) *pod.GraphModule {
	graphModule := new(pod.GraphModule)
	var upVersion, newest string
	if upPodfile != nil {
		var exist bool
		upVersion, exist = upPodfile.GetDependVersion(nil, aDepend.Name())
		if exist {
			upVersion = realVesion(aDepend.Name(), upVersion)
		}
	}
	newest, _ = queryNewestVersion(aDepend.Name(), "")
	if pod.IsVersion(aDepend.Version()) {
		graphModule.Version = aDepend.Version()
	} else if aDepend.Version() == "" {
		graphModule.Version = newest
	} else {
		graphModule.Version, _ = queryNewestVersion(aDepend.Name(), aDepend.Version())
	}
	graphModule.Name = aDepend.Name()
	graphModule.UpdateToVersion = upVersion
	graphModule.NewestVersion = newest
	graphModule.IsLocal = aDepend.IsLocal()
	if aDepend.Subdepends() != nil && graphModule.UseVersion() == graphModule.Version {
		graphModule.Depends = aDepend.Subdepends()
	} else if graphModule.Version != "" {
		deps, _, e := queryDepends(graphModule.Name, graphModule.UseVersion())
		if e == nil && len(deps) > 0 {
			graphModule.Depends = deps
		}
	}
	return graphModule
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

func check(filePath string, graphPodfile pod.GraphPodfile, upPodfile *pod.Podfile, times int) {
	println("分析隐性依赖[第" + strconv.Itoa(times) + "次迭代]: " + filePath)
	if graphPodfile == nil {
		return
	}
	unfound := graphPodfile.Check()
	if len(unfound) == 0 {
		return
	}
	var buffer bytes.Buffer
	for _, aDep := range unfound {
		if _Conf.IsDebug() {
			buffer.WriteString("[")
			buffer.WriteString(aDep.Name())
			if aDep.Version() != "" {
				buffer.WriteString(":" + aDep.Version())
			}
			buffer.WriteString("] ")
		}
		old, ok := graphPodfile[aDep.Name()]
		if ok {
			v, e := queryNewestVersion(aDep.Name(), aDep.Version())
			if e != nil || v == "" {
				old.UpdateToVersion = "*"
			} else {
				old.UpdateToVersion = v
				deps, _, e := queryDepends(old.Name, old.UseVersion())
				if e == nil {
					old.Depends = deps
				}
			}
		} else {
			aModule := buildGraphModule(aDep, upPodfile)
			aModule.IsNew = true
			graphPodfile[aModule.Name] = aModule
		}
	}
	if _Conf.IsDebug() {
		println(buffer.String())
	}
	check(filePath, graphPodfile, upPodfile, times+1)
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

func writeGraphPodfile(filePath string, graphPodfile pod.GraphPodfile, outputMode int) {
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
	if _, e := buffer.WriteString("ModuleName,IsCommon,IsImplicit,IsLocal,Current,UpgradeTo,UpgradeTag,Newest,Dependencies\n"); e != nil {
		printRed(e.Error(), false)
		return
	}
	if graphPodfile != nil {
		if _, e := buffer.Write(graphPodfile.Bytes()); e != nil {
			printRed(e.Error(), false)
			return
		}
	}
	if e := WriteFile(csvFilePath, buffer.Bytes(), true, os.ModePerm); e != nil {
		printRed("输出文件错误["+csvFilePath+"]: "+e.Error(), false)
	}

	// 是否输出辅助文件
	if outputMode == 0 {
		return
	}

	var bufferCommon bytes.Buffer
	var bufferRemote bytes.Buffer
	if outputMode&__OPTION_OUTPUT_COMMON == __OPTION_OUTPUT_COMMON {
		bufferCommon.WriteString("\n#Common\n")
	}
	if outputMode&__OPTION_OUTPUT_REMOTE == __OPTION_OUTPUT_REMOTE {
		bufferRemote.WriteString("\n#Remote\n")
	}
	for _, aModule := range graphPodfile {
		if outputMode&__OPTION_OUTPUT_COMMON == __OPTION_OUTPUT_COMMON && aModule.IsCommon {
			note := ""
			if aModule.IsLocal {
				note = " # Local"
			}
			bufferCommon.WriteString("pod '" + aModule.Name + "', '" + aModule.UseVersion() + "'" + note + "\n")
			continue
		}
		if outputMode&__OPTION_OUTPUT_REMOTE == __OPTION_OUTPUT_REMOTE && !aModule.IsCommon && !aModule.IsLocal {
			bufferRemote.WriteString("pod '" + aModule.Name + "', '" + aModule.UseVersion() + "'\n")
		}
	}
	br := bufferRemote.Bytes()
	bc := bufferCommon.Bytes()
	cap := len(br) + len(bc)
	if cap == 0 {
		return
	}
	b := make([]byte, 0, cap)
	b = append(b, bc...)
	b = append(b, br...)
	if e := WriteFile(filePath, b, true, os.ModePerm); e != nil {
		printRed("输出文件错误["+csvFilePath+"]: "+e.Error(), false)
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
