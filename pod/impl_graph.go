package pod

import (
	"bytes"
	"strconv"
	"strings"
)

// ** GraphPodfile Impl **
func (s GraphPodfile) Check() []*DependBase {
	s.banlanceVersion()
	unfound := make([]*DependBase, 0, 10)
	for _, val := range s {
		for _, dep := range val.Depends {
			base := dep.Name()
			depModule, ok := s[base]
			if ok {
				if depModule.UseVersion() == "" || depModule.UseVersion() == "*" || dep.Version() == "" {
					continue
				}
				ok = MatchVersionConstraint(dep.Version(), depModule.UseVersion())
				if ok {
					continue
				}
			}
			unfound = append(unfound, dep)
		}
	}
	return unfound
}

func (s GraphPodfile) Bytes() []byte {
	var buffer bytes.Buffer
	for _, m := range s {
		buffer.WriteString(m.Name + "," + strconv.FormatBool(m.IsCommon) + "," + strconv.FormatBool(m.IsNew) + "," + strconv.FormatBool(m.IsImplicit) + "," + strconv.FormatBool(m.IsLocal) + ",")
		buffer.WriteString(m.Version + "," + m.UpdateToVersion + "," + m.UpgradeTag() + "," + m.NewestVersion + ",")
		for _, aDep := range m.Depends {
			buffer.WriteString(aDep.String() + " ")
		}
		buffer.WriteString("\n")
	}
	return buffer.Bytes()
}

func (s GraphPodfile) String() string {
	return string(s.Bytes())
}

// 平衡子模块版本号(让所有跟模块相同的模块版本保持最大版本)
func (s GraphPodfile) banlanceVersion() {
	foundMap := make(map[string]string)
	// 发现父模块及其最大版本
	for moduleName, aModule := range s {
		if strings.Index(moduleName, "/") < 0 {
			continue
		}
		baseName := BaseModule(moduleName)
		currentVersion, _ := foundMap[baseName]
		useVersion := aModule.UseVersion()
		if currentVersion != "" && useVersion != "" {
			// 对比版本
			if maxVersion, e := MaxVersion("", currentVersion, useVersion); e != nil {
				foundMap[baseName] = useVersion
			} else {
				foundMap[baseName] = maxVersion
			}
		} else if currentVersion == "" {
			foundMap[baseName] = useVersion
		} else {
			foundMap[baseName] = currentVersion
		}
	}

	// 给所有子模块赋值最大版本
	for moduleName, version := range foundMap {
		// 先发现是否有根模块及其版本
		if rootModule, ok := s[moduleName]; ok {
			rootModuleVersion := rootModule.UseVersion()
			if rootModuleVersion != "" && version != "" {
				if maxVersion, e := MaxVersion("", rootModuleVersion, version); e == nil {
					version = maxVersion
				}
			} else if version == "" {
				version = rootModuleVersion
			}
		}
		if version == "" {
			continue
		}
		for moduleNameOther, aDependOther := range s {
			if BaseModule(moduleNameOther) != moduleName {
				continue
			}
			if aDependOther.UpdateToVersion == "" {
				aDependOther.Version = version
			} else {
				aDependOther.UpdateToVersion = version
			}
		}
	}
}

// ** GraphModule Impl **
func (s *GraphModule) UpgradeTag() string {
	if s.UpdateToVersion == "" || s.Version == "" {
		return "-"
	} else {
		c := CompareVersion(s.Version, s.UpdateToVersion)
		switch c {
		case -1:
			return "up"
		case 1:
			return "down"
		default:
			return "-"
		}
	}
}

func (s *GraphModule) UseVersion() string {
	if len(s.UpdateToVersion) == 0 {
		return s.Version
	} else if len(s.NewestVersion) == 0 {
		return s.UpdateToVersion
	}
	c := CompareVersion(s.UpdateToVersion, s.NewestVersion)
	if c > 0 {
		return s.NewestVersion
	} else {
		return s.UpdateToVersion
	}
}

func (s *GraphModule) ReferenceNodes() []string {
	if s.flattenDepends == nil {
		l := len(s.Depends)
		if l > 0 {
			s.flattenDepends = make([]string, l, l)
			for idx, aDepend := range s.Depends {
				s.flattenDepends[idx] = aDepend.N
			}
		}
	}
	return s.flattenDepends
}
