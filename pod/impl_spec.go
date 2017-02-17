package pod

import (
	"errors"
	"io/ioutil"
	"os/exec"
	"path"
	"strings"

	"encoding/json"

	"regexp"

	"github.com/go-hayden-base/fs"
)

// ** Spec Impl **
func (s *Spec) HashSpec() {
	if s.hasHash {
		return
	}

	if s.ModulePath != "" {
		s.ModulePath = s.ModulePath + "/" + s.Name
	} else {
		s.ModulePath = s.Name
	}

	if len(s.Subspecs) == 0 {
		return
	}

	for _, subspec := range s.Subspecs {
		if s.IsDefaultSpec(subspec.Name) {
			if s.DefaultSpecsMap == nil {
				s.DefaultSpecsMap = make(map[string]*Spec)
			}
			s.DefaultSpecsMap[subspec.Name] = subspec
		} else {
			if s.SingleSpecsMap == nil {
				s.SingleSpecsMap = make(map[string]*Spec)
			}
			s.SingleSpecsMap[subspec.Name] = subspec
		}
		subspec.ModulePath = s.ModulePath
		subspec.HashSpec()
	}
	s.hasHash = true
}

func (s *Spec) JSON() ([]byte, error) {
	return json.Marshal(s)
}

func (s *Spec) enumerateDepends(f func(module, depend, version string)) {
	if f == nil {
		return
	}
	if s.Dependences != nil {
		s.Dependences.enumerateDepends(func(dep, ver string) {
			f(s.Name, dep, ver)
		})
	}
	if s.Subspecs != nil {
		for _, spec := range s.Subspecs {
			spec.enumerateDepends(f)
		}
	}
}

func (s *Spec) IsDefaultSpec(name string) bool {
	if s.DefaultSpecs == nil {
		return true
	}
	if ds, ok := s.DefaultSpecs.(string); ok && ds == name {
		return true
	}
	if dss, ok := s.DefaultSpecs.([]interface{}); ok && len(dss) > 0 {
		for _, item := range dss {
			if s, ok := item.(string); ok && s == name {
				return true
			}
		}
	}
	return false
}

func (s *Spec) GetAllDepends(name string) map[string]string {
	s.HashSpec()
	baseName := BaseModule(name)
	if baseName == "" || baseName != s.Name {
		return nil
	}
	regString := `^` + baseName + `(/\S+)*$`
	reg := regexp.MustCompile(regString)
	res := make(map[string]string)
	add := make(map[string]bool)
	res[name] = ""
	f := func(a map[string]string, check map[string]bool, r *regexp.Regexp) (string, bool) {
		if a == nil || check == nil {
			return "", false
		}
		for key, _ := range a {
			if _, ok := check[key]; !ok && r.MatchString(key) {
				return key, true
			}
		}
		return "", false
	}

	for {
		p, ok := f(res, add, reg)
		if !ok {
			break
		}
		if specs := s.getPathSubspecs(p); specs != nil && len(specs) > 0 {
			mergeDpendMap(res, getPathDepends(specs))
		}
		add[p] = true
	}
	delete(res, name)
	if len(res) > 0 {
		return res
	}
	return nil
}

func (s *Spec) GetExcludeSubspecDepends() map[string]string {
	s.HashSpec()
	if s.Dependences == nil {
		return nil
	}
	res := make(map[string]string)
	s.Dependences.enumerateDepends(func(d, v string) {
		res[d] = v
	})
	return res
}

func (s *Spec) GetDepends() map[string]string {
	s.HashSpec()
	res := make(map[string]string)
	mergeDpendMap(res, s.GetExcludeSubspecDepends())
	if s.DefaultSpecsMap != nil {
		for _, spec := range s.DefaultSpecsMap {
			res[spec.ModulePath] = ""
			mergeDpendMap(res, spec.GetDepends())
		}
	} else {
		for _, spec := range s.SingleSpecsMap {
			res[spec.ModulePath] = ""
			mergeDpendMap(res, spec.GetDepends())
		}
	}
	return res
}

func (s *Spec) getPathSubspecs(name string) []*Spec {
	subs := strings.Split(name, "/")
	l := len(subs)
	if l == 1 && s.Name != name {
		return nil
	}
	specs := make([]*Spec, 1, l)
	specs[0] = s
	currentSpec := s
	newSubs := subs[1:]
	for _, item := range newSubs {
		nextSpec, ok := currentSpec.searchSubspecsFromMap(item)
		if !ok {
			return nil
		}
		specs = append(specs, nextSpec)
		currentSpec = nextSpec
	}
	return specs
}

func (s *Spec) searchSubspecsFromMap(name string) (*Spec, bool) {
	var spec *Spec
	var ok bool
	if s.DefaultSpecsMap != nil {
		spec, ok = s.DefaultSpecsMap[name]
	}
	if ok {
		return spec, ok
	}
	if s.SingleSpecsMap != nil {
		spec, ok = s.SingleSpecsMap[name]
	}
	return spec, ok
}

// ** SpecDenpendence Impl **
func (s SpecDenpendence) enumerateDepends(f func(depend, version string)) {
	if f == nil {
		return
	}
	for key, val := range s {
		if len(val) > 0 {
			for _, v := range val {
				f(key, v)
			}
		} else {
			f(key, "")
		}
	}
}

func (s SpecDenpendence) Version(name string) string {
	versions, ok := s[name]
	if ok && len(versions) > 0 {
		return versions[0]
	}
	return ""
}

// ** Private Func **
func mergeDpendMap(a map[string]string, b map[string]string) {
	for key, val := range b {
		a[key] = val
	}
}

func getPathDepends(specs []*Spec) map[string]string {
	l := len(specs)
	if l == 0 {
		return nil
	}
	res := make(map[string]string)
	for idx, spec := range specs {
		if idx == l-1 {
			mergeDpendMap(res, spec.GetDepends())
		} else {
			mergeDpendMap(res, spec.GetExcludeSubspecDepends())
		}
	}
	return res
}

// ** Public Func **
func ReadSpec(filePath string, printLog bool) (*Spec, error) {
	printIfNeed(printLog, "解析Spec文件: "+filePath+" ... ")
	if len(filePath) == 0 || !fs.FileExists(filePath) {
		return nil, errors.New("请正确指定spec文件！")
	}
	ext := strings.ToLower(path.Ext(filePath))
	if ext != ".json" && ext != ".podspec" {
		return nil, errors.New("spec 文件格式不正确！")
	}
	var b []byte
	var err error
	if ext == ".json" {
		b, err = ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
	} else {
		b, err = exec.Command("pod", "ipc", "spec", filePath).Output()
		if err != nil {
			return nil, err
		}
	}

	if spec, err := NewSpecWithJSONBytes(b); err != nil {
		return nil, err
	} else {
		spec.FilePath = filePath
		return spec, nil
	}
}

func NewSpecWithJSONString(json string) (*Spec, error) {
	return NewSpecWithJSONBytes([]byte(json))
}

func NewSpecWithJSONBytes(b []byte) (*Spec, error) {
	var spec *Spec
	if err := json.Unmarshal(b, &spec); err != nil {
		return nil, err
	}
	return spec, nil
}
