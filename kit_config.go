package main

import (
	"errors"
	"os"
	"strings"

	"encoding/json"
	"io/ioutil"

	"os/user"

	"path"

	"strconv"

	"github.com/go-hayden-base/fs"
)

const (
	__ENV_RELEASE = iota
	__ENV_ALPHA
	__ENV_BETA
	__ENV_DEBUG
)

type Config struct {
	Workspace   string `json:"-" bson:"-"`
	Environment int    `json:"-" bson:"-"`

	OutputDirectory string        `json:"output_directory,omitempty" bson:"output_directory,omitempty"`
	PodRepoRoot     string        `json:"pod_repo_root,omitempty" bson:"pod_repo_root,omitempty"`
	PodRepos        []*ConfigRepo `json:"pod_repos,omitempty" bson:"pod_repos,omitempty"`
	SpecThread      int           `json:"spec_thread,omitempty" bson:"spec_thread,omitempty"`
}

type ConfigRepo struct {
	Name    string   `json:"name,omitempty" bson:"name,omitempty"`
	Exclude []string `json:"exclude,omitempty" bson:"exclude,omitempty"`
}

func NewConfig() (*Config, error) {
	// 获取工作目录
	p := os.Getenv("PANDORA_PATH")
	if strings.TrimSpace(p) == "" {
		return nil, errors.New("请配置环境变量PANDORA_PATH，用以指定pandora的工作目录!")
	}
	filepath := path.Join(p, "pandora.cfg.json")
	if len(filepath) == 0 || !fs.FileExists(filepath) {
		return nil, errors.New("配置文件路径不正确或配置文件不存在! [" + filepath + "]")
	}
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var config *Config
	err = json.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}
	err = config.Check()
	if err != nil {
		return nil, err
	}
	config.Workspace = p
	if config.OutputDirectory == "" {
		config.OutputDirectory = path.Join(p, "output")
	}

	env := os.Getenv("PANDORA_ENV")
	if envInt, e := strconv.Atoi(env); e == nil && envInt <= __ENV_DEBUG && envInt >= __ENV_RELEASE {
		config.Environment = envInt
	}
	return config, nil
}

func (s *Config) Check() error {
	if len(s.PodRepoRoot) == 0 {
		homeDir, err := user.Current()
		if err != nil {
			return err
		}
		s.PodRepoRoot = path.Join(homeDir.HomeDir, ".cocoapods", "repos")
	}
	if !fs.DirectoryExists(s.PodRepoRoot) {
		return errors.New("CocoaPods Spec仓库根目录不存在! [" + s.PodRepoRoot + "]")
	}
	if len(s.PodRepos) == 0 {
		return errors.New("请设置索引的Spec仓库！")
	}
	if s.SpecThread < 1 {
		s.SpecThread = 5
	} else if s.SpecThread > 20 {
		s.SpecThread = 20
	}
	return nil
}

func (s *Config) IsDebug() bool {
	return s.Environment == __ENV_DEBUG
}

func (s *Config) IsAlpha() bool {
	return s.Environment == __ENV_ALPHA
}

func (s *Config) IsBeta() bool {
	return s.Environment == __ENV_BETA
}

func (s *Config) IsRelease() bool {
	return s.Environment == __ENV_RELEASE
}
