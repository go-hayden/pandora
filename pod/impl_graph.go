package pod

import (
	"bytes"
	"strconv"
	"strings"
)

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

func (s GraphPodfile) Check() []*DependBase {
	unfound := make([]*DependBase, 0, 10)
	for _, val := range s {
		for _, dep := range val.Depends {
			base := dep.Name()
			ok := true
			for {
				var depModule *GraphModule
				depModule, ok = s[base]
				if ok {
					if depModule.UseVersion() == "" || depModule.UseVersion() == "*" || dep.Version() == "" {
						break
					}
					ok = MatchVersionConstraint(dep.Version(), depModule.UseVersion())
					if ok {
						break
					}
				}
				base, ok = ModuleBase(base)
				if !ok {
					break
				}
			}
			if !ok {
				unfound = append(unfound, dep)
			}
		}
	}
	return unfound
}

func ModuleBase(s string) (string, bool) {
	tmp := strings.Split(s, "/")
	if len(tmp) < 2 {
		return "", false
	}
	return strings.Join(tmp[:len(tmp)-1], "/"), true
}

func (s GraphPodfile) Bytes() []byte {
	var buffer bytes.Buffer
	for _, m := range s {
		buffer.WriteString(m.Name + "," + strconv.FormatBool(m.IsCommon) + "," + strconv.FormatBool(m.IsNew) + "," + strconv.FormatBool(m.IsLocal) + ",")
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
