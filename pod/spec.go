package pod

import (
	"errors"
	"io/ioutil"
	"os/exec"
	"path"
	"strings"

	"encoding/json"

	"github.com/go-hayden-base/fs"
)

type Spec struct {
	FilePath    string
	Name        string          `json:"name,omitempty" bson:"name,omitempty"`
	Version     string          `json:"version,omitempty" bson:"version,omitempty"`
	Source      *SpecSource     `json:"source,omitempty" bson:"source,omitempty"`
	Dependences SpecDenpendence `json:"dependencies,omitempty" bson:"dependencies,omitempty"`
	Subspecs    []*Spec         `json:"subspecs,omitempty" bson:"subspecs,omitempty"`
}

type SpecSource struct {
	Git    string `json:"git,omitempty" bson:"git,omitempty"`
	Tag    string `json:"tag,omitempty" bson:"tag,omitempty"`
	Branch string `json:"branch,omitempty" bson:"branch,omitempty"`
	Commit string `json:"commit,omitempty" bson:"commit,omitempty"`
}

type SpecDenpendence map[string][]string

func (s SpecDenpendence) Version(name string) string {
	versions, ok := s[name]
	if ok && len(versions) > 0 {
		return versions[0]
	}
	return ""
}

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
	var spec *Spec
	err = json.Unmarshal(b, &spec)
	if err != nil {
		return nil, err
	}

	spec.FilePath = filePath
	return spec, nil
}

func printIfNeed(p bool, msg string) {
	if p {
		println(msg)
	}
}
