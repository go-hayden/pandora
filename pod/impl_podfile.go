package pod

import (
	"os/exec"
	"path"

	fdt "github.com/go-hayden-base/foundation"
	ver "github.com/go-hayden-base/version"
	yaml "gopkg.in/yaml.v2"
)

// ** Podfile Impl **
func (s *Podfile) HasDepend(targets []string, name string) bool {
	has := false
	for _, aTarget := range s.Targets {
		if targets != nil && !fdt.SliceContainsStr(aTarget.Name, targets) {
			continue
		}
		has = aTarget.HasDepend(name)
		if has {
			break
		}
	}
	return has
}

func (s *Podfile) GetDependVersion(targets []string, depend string) (string, bool) {
	found := make([]string, 0, 5)
	exist := false
	for _, aTarget := range s.Targets {
		if targets != nil && !fdt.SliceContainsStr(aTarget.Name, targets) {
			continue
		}
		aDepend := aTarget.FuzzyDepndWithName(depend)
		if aDepend != nil {
			if !exist {
				exist = true
			}
			if len(aDepend.Version()) > 0 {
				found = append(found, aDepend.Version())
			}
		}
	}
	l := len(found)
	if l < 1 {
		return "", exist
	}
	max, err := ver.MaxVersion("", found...)
	if err != nil {
		return found[0], exist
	}
	return max, exist
}

func (s *Podfile) EnumerateAllDepends(f func(target, depend, version string)) {
	if f == nil {
		return
	}
	for _, aTarget := range s.Targets {
		for _, aDepends := range aTarget.Depends {
			f(aTarget.Name, aDepends.N, aDepends.V)
		}
	}
}

func (s *Podfile) TargetWithName(name string) *Target {
	for _, target := range s.Targets {
		if target.Name == name {
			return target
		}
	}
	return nil
}

func (s *Podfile) Print() {
	for _, target := range s.Targets {
		println("-> " + target.Name)
		for _, depend := range target.Depends {
			println("   -", depend.Name, depend.Version, depend.Type, depend.SpecPath)
		}
	}
}

// ** Target Impl **
func (s *Target) DepndWithName(name string) *Depend {
	for _, dep := range s.Depends {
		if dep.Name() == name {
			return dep
		}
	}
	return nil
}

func (s *Target) FuzzyDepndWithName(name string) *Depend {
	for _, dep := range s.Depends {
		if fdt.StrSplitFirst(dep.Name(), "/") == fdt.StrSplitFirst(name, "/") {
			return dep
		}
	}
	return nil
}

func (s *Target) HasDepend(name string) bool {
	has := false
	for _, dep := range s.Depends {
		if dep.Name() == name {
			has = true
			break
		}
	}
	return has
}

// ** Depend Impl **
func (s *Depend) Subdepends() []*DependBase {
	return s.SpecDepends
}

func (s *Depend) IsLocal() bool {
	return s.SpecPath != ""
}

// ** Func Public **
func NewPodfile(filePath string, rel bool) (*Podfile, error) {
	b, e := exec.Command("pod", "ipc", "podfile", filePath).Output()
	if e != nil {
		return nil, e
	}
	var pf *p_podfile
	e = yaml.Unmarshal(b, &pf)
	if e != nil {
		return nil, e
	}
	var dir string
	if rel && len(filePath) > 0 {
		dir = path.Dir(filePath)
	}
	podfile := new(Podfile)
	podfile.Targets = make([]*Target, 0, 10)
	for _, a := range pf.Target_definitions {
		// 读取子Targert
		for _, b := range a.Children {
			target := new(Target)
			target.Name = b.Name
			target.Depends = generateDepend(dir, b.Dependencies)
			podfile.Targets = append(podfile.Targets, target)
		}

		// 读取主Target
		if len(a.Dependencies) > 0 {
			target := new(Target)
			target.Name = "*"
			target.Depends = generateDepend(dir, a.Dependencies)
			podfile.Targets = append(podfile.Targets, target)
		}
	}
	podfile.FilePath = filePath
	return podfile, nil
}

func FillPodfile(podfile *Podfile, threadNum int, printLog bool) {
	if threadNum < 1 {
		threadNum = 1
	}
	c := make(chan bool, threadNum)
	for i := 0; i < threadNum; i++ {
		c <- true
	}
	funcAsync := func(d *Depend) {
		s, e := ReadSpec(d.SpecPath, printLog)
		if e != nil {
			d.Err = e
		} else {
			d.V = s.Version
			d.SpecDepends = getAllDependsFromSpec(s)
		}
		c <- true
	}
	for _, target := range podfile.Targets {
		for _, depend := range target.Depends {
			if len(depend.Type) == 0 {
				continue
			}
			<-c
			go funcAsync(depend)
		}
	}
	for i := 0; i < threadNum; i++ {
		<-c
	}
}

// ** Func Private **
func getAllDependsFromSpec(spec *Spec) []*DependBase {
	if spec == nil {
		return nil
	}
	mapDup := make(map[string]*DependBase)
	spec.enumerateDepends(func(module, depend, version string) {
		_, ok := mapDup[depend]
		if ok {
			return
		}
		aDepend := new(DependBase)
		aDepend.N = depend
		aDepend.V = version
		mapDup[depend] = aDepend
	})
	if len(mapDup) == 0 {
		return nil
	}
	res := make([]*DependBase, 0, len(mapDup))
	for _, val := range mapDup {
		res = append(res, val)
	}
	return res
}

func generateDepend(f string, dep []interface{}) []*Depend {
	res := make([]*Depend, 0, 20)
	for _, c := range dep {
		s, ok := c.(string)
		if ok {
			depend := new(Depend)
			depend.N = s
			res = append(res, depend)
			continue
		}
		cm, ok := c.(map[interface{}]interface{})
		if !ok {
			continue
		}
		for k, v := range cm {
			ks, ok := k.(string)
			if !ok {
				continue
			}
			depend := new(Depend)
			depend.N = ks
			res = append(res, depend)

			varr, ok := v.([]interface{})
			if !ok {
				continue
			}
			if len(varr) == 0 {
				continue
			}
			d := varr[0]
			switch d.(type) {
			case string:
				depend.V = d.(string)
			case map[interface{}]interface{}:
				x := d.(map[interface{}]interface{})
				for kk, vv := range x {
					kkk, okk := kk.(string)
					vvv, okv := vv.(string)
					if okk && okv {
						depend.Type = kkk
						if len(f) > 0 {
							depend.SpecPath = path.Join(f, vvv)
						} else {
							depend.SpecPath = vvv
						}
					}
				}
			}
		}
	}
	return res
}
