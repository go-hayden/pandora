package main

import (
	"strings"

	"errors"

	"os"

	"path"
	"time"

	"strconv"

	fdt "github.com/go-hayden-base/foundation"
	"github.com/go-hayden-base/fs"
	"github.com/go-hayden-base/pod"
	tmp "github.com/go-hayden-base/template"
)

func EmptyAnalysisPodfile(pathArg string, mergeMode bool, flatten bool) *AnalysisPodfile {
	aAPodfile := new(AnalysisPodfile)
	if idx := strings.Index(pathArg, ":"); idx > -1 {
		aAPodfile.FilePath = pathArg[:idx]
		aAPodfile.Target = pathArg[idx+1:]
	} else {
		aAPodfile.FilePath = pathArg
	}
	aAPodfile.FilePath = fs.AbsPath("", aAPodfile.FilePath)
	aAPodfile.MergeMode = mergeMode
	aAPodfile.Flatten = flatten
	return aAPodfile
}

type AnalysisPodfile struct {
	FilePath   string
	Target     string
	MergeMode  bool
	Flatten    bool
	Podfile    *pod.Podfile
	MapPodfile pod.MapPodfile
	Rules      map[string]string

	iterationCount int
	outputRoot     string
}

func (s *AnalysisPodfile) BuildPodfile() error {
	aPodfile, err := pod.NewPodfile(s.FilePath)
	if err != nil {
		return err
	}
	aPodfile.FillLocalModuleDepends(_Conf.SpecThread, func(suc bool, msg string) {
		if suc {
			println(msg)
		} else {
			printRed(msg, false)
		}
	})
	s.Podfile = aPodfile
	return nil
}

func (s *AnalysisPodfile) BuildMapPodfile() error {
	if s.Podfile == nil {
		return errors.New("请先生成Podfile")
	}
	s.MapPodfile = s.Podfile.MapPodfileWithTarget(s.Target)
	if s.MergeMode {
		for module, version := range s.Rules {
			if _, ok := s.MapPodfile[module]; !ok {
				aMapMoudel := new(pod.MapPodfileModule)
				aMapMoudel.Name = module
				aMapMoudel.Version = version
				aMapMoudel.IsNew = true
				s.MapPodfile[module] = aMapMoudel
			}
		}
	}
	return nil
}

func (s *AnalysisPodfile) FillMapPodfile() {
	for _, aModule := range s.MapPodfile {
		if aModule.NewestVersion == "" {
			if newest, err := queryNewestVersionWithConstraint(aModule.Name, ""); err != nil {
				aModule.NewestVersion = "*"
			} else {
				aModule.NewestVersion = newest
			}
		}

		if aModule.UpdateToVersion == "" {
			if version, exist := s.RuleVersion(aModule.Name); exist {
				aModule.UpdateToVersion = versionify(aModule.Name, version)
			}
		}

		aModule.Version = versionify(aModule.Name, aModule.Version)

		if _, ok := aModule.Depends(); !ok {
			if depends, _, err := queryDepends(aModule.Name, aModule.UseVersion()); err != nil || depends == nil {
				aModule.SetDepends([]*pod.DependBase{})
			} else {
				aModule.SetDepends(depends)
			}
		}
	}
}

func (s *AnalysisPodfile) IterationMapPodfile() {
	if s.MapPodfile == nil {
		return
	}
	s.FillMapPodfile()
	s.iterationCount++
	dissatisfy := s.MapPodfile.Check()
	if len(dissatisfy) == 0 {
		s.iterationCount = 0
		return
	}
	println("分析隐性依赖[第", s.iterationCount, "次迭代]：")
	for module, versions := range dissatisfy {
		if str := strings.Join(versions, ", "); str != "" {
			println(" -", module, "->", str)
		} else {
			println(" -", module)
		}
		version, err := queryNewestVersionWithConstraints(module, versions)
		if err != nil {
			printRed(err.Error(), false)
			os.Exit(1)
		}
		if aModule, ok := s.MapPodfile[module]; ok {
			aModule.UpdateToVersion = version
		} else {
			aModule = new(pod.MapPodfileModule)
			aModule.Name = module
			aModule.Version = version
			aModule.IsNew = true
			aModule.IsImplicit = true
			s.MapPodfile[module] = aModule
		}
	}
	s.IterationMapPodfile()
}

func (s *AnalysisPodfile) RuleVersion(module string) (string, bool) {
	if s.Rules == nil {
		return "", false
	}
	var version string
	exist := false
	if strings.Index(module, "/") < 0 {
		version, exist = s.Rules[module]
		return version, exist
	}
	fdt.StrEnumerate(module, "/", true, func(surplus string, current string, stop *bool) {
		if version, exist = s.Rules[surplus]; exist {
			*stop = true
		}
	})
	return version, exist
}

func (s *AnalysisPodfile) OutputAnalysisTable() (string, error) {
	src, ok := _Conf.Templates["jqgrid"]
	if !ok {
		return "", errors.New("没有设置模板")
	}

	aJQGrid := tmp.NewJQGrid(s.fileNamePrefix(), "IsCommon", "desc", 10)
	aJQGrid.SetColCaption("ModuleName", "IsCommon", "IsNew", "IsImplicit", "IsLocal", "Current", "UpgradeTo", "UpgradeTag", "Newest", "Dependencies")
	aJQGrid.SetColModel(0, "ModuleName", "", "left", 100, true)
	aJQGrid.SetColModel(1, "IsCommon", "", "center", 60, true)
	aJQGrid.SetColModel(2, "IsNew", "", "center", 60, true)
	aJQGrid.SetColModel(3, "IsImplicit", "", "center", 60, true)
	aJQGrid.SetColModel(4, "IsLocal", "", "center", 60, true)
	aJQGrid.SetColModel(5, "Current", "", "center", 60, true)
	aJQGrid.SetColModel(6, "UpgradeTo", "", "center", 60, true)
	aJQGrid.SetColModel(7, "UpgradeTag", "", "center", 60, true)
	aJQGrid.SetColModel(8, "Newest", "", "center", 60, true)
	aJQGrid.SetColModel(9, "Dependencies", "", "left", 300, false)
	s.MapPodfile.EnumerateAll(func(module, current, upgradeTo, upgradeTag, newest, dependencies string, isCommon, isNew, isImplicit, isLocal bool) {
		isCommonStr := strconv.FormatBool(isCommon)
		isNewStr := strconv.FormatBool(isNew)
		isImplicitStr := strconv.FormatBool(isImplicit)
		isLocalStr := strconv.FormatBool(isLocal)
		aJQGrid.AddRowData(module, isCommonStr, isNewStr, isImplicitStr, isLocalStr, current, upgradeTo, upgradeTag, newest, dependencies)
	})
	outputPath := s.outputPath() + "_table"
	println(src)
	if err := aJQGrid.Output(src, outputPath); err != nil {
		return "", err
	}
	return "输出分析表格: " + outputPath, nil
}

func (s *AnalysisPodfile) fileNamePrefix() string {
	filePath := s.FilePath
	output := path.Base(filePath)
	filePath = path.Dir(filePath)
	output = path.Base(filePath) + "_" + output
	filePath = path.Dir(filePath)
	return path.Base(filePath) + "_" + output
}

func (s *AnalysisPodfile) outputPath() string {
	if s.outputRoot == "" {
		timestamp := time.Now().Format("20060101_150405")
		s.outputRoot = path.Join(_Conf.OutputDirectory, "up_"+timestamp, s.fileNamePrefix())
	}
	return s.outputRoot
}
