package pod

import (
	"io/ioutil"
	"path"

	"errors"

	merr "github.com/go-hayden-base/err"
	"github.com/go-hayden-base/fs"
	"github.com/go-hayden-base/str"
)

func (s *Pod) Index(root string, repos []string, exclude map[string]bool) error {
	if !fs.DirectoryExists(root) {
		return merr.NewErrMessage(merr.ErrCodeNotExist, "Pod根目录不存在["+root+"]")
	}
	if repos == nil || len(repos) == 0 {
		return merr.NewErrMessage(merr.ErrCodeParamInvalid, "没有索引的仓库！")
	}
	podrepos := make([]*PodRepo, 0, len(repos))
	for _, rn := range repos {
		var reporoot string
		if rn == "master" {
			reporoot = path.Join(root, rn, "Specs")
		} else {
			reporoot = path.Join(root, rn)
		}
		if !fs.DirectoryExists(reporoot) {
			return merr.NewErrMessage(merr.ErrCodeNotExist, "仓库不存在["+reporoot+"]")
		}
		repo := new(PodRepo)
		repo.Name = rn
		repo.Root = reporoot
		podrepos = append(podrepos, repo)
	}
	c := make(chan error, len(podrepos))
	for _, repo := range podrepos {
		go goIndexRepo(repo, exclude, c)
	}
	for i := 0; i < len(podrepos); i++ {
		<-c
	}
	s.PodRepos = podrepos
	return nil
}

func (s *Pod) Print() {
	for _, repo := range s.PodRepos {
		println("\n======== " + repo.Name + "========")
		for _, module := range repo.Modules {
			println("->" + repo.Name)
			for _, version := range module.Versions {
				println(" |_" + version.Name + " [" + path.Join(version.Root, version.FileName) + "]")
			}
		}
	}
}

func goIndexRepo(repo *PodRepo, exclude map[string]bool, c chan error) {
	c <- repo.Index(exclude)
}

func (s *PodRepo) Index(exclude map[string]bool) error {
	dirs, err := ioutil.ReadDir(s.Root)
	if err != nil {
		return merr.NewErr(merr.ErrCodeUnknown, err)
	}
	l := len(dirs)
	modules := make([]*PodModule, 0, l)
	for _, f := range dirs {
		if !f.IsDir() {
			continue
		}
		if f.Name() == ".git" {
			continue
		}
		module := new(PodModule)
		module.Name = f.Name()
		module.Root = path.Join(s.Root, module.Name)
		e := module.Index(exclude)
		if e == nil {
			modules = append(modules, module)
		}
	}
	if len(modules) == 0 {
		return errors.New("Warn: No module in repo " + s.Name + " [" + s.Root + "]")
	}
	s.Modules = modules
	return nil
}

func (s *PodModule) Index(exclude map[string]bool) error {
	dirs, err := ioutil.ReadDir(s.Root)
	if err != nil {
		return merr.NewErr(merr.ErrCodeUnknown, err)
	}
	l := len(dirs)
	versions := make([]*PodModuleVersion, 0, l)
	for _, f := range dirs {
		if !f.IsDir() {
			continue
		}
		pwd := path.Join(s.Root, f.Name())
		pwdMD5 := str.MD5(pwd)
		if exclude != nil {
			_, ok := exclude[pwdMD5]
			if ok {
				continue
			}
		}
		version := new(PodModuleVersion)
		version.Name = f.Name()
		version.Root = pwd
		e := version.Index()
		if e == nil {
			versions = append(versions, version)
		}
	}
	if len(versions) == 0 {
		return errors.New("Warn: No version in module " + s.Name + " [" + s.Root + "]")
	}
	s.Versions = versions
	return nil
}

func (s *PodModuleVersion) Index() error {
	dirs, err := ioutil.ReadDir(s.Root)
	if err != nil {
		return merr.NewErr(merr.ErrCodeUnknown, err)
	}
	for _, fi := range dirs {
		if fi.IsDir() {
			continue
		}
		ext := path.Ext(fi.Name())
		if ext != ".podspec" && ext != ".json" {
			continue
		}
		s.FileName = fi.Name()
		break
	}
	if len(s.FileName) == 0 {
		return errors.New("Warn: Can not find spec file in " + s.Root)
	}

	return nil
}

func Index(podRoot string, repos []string, exclude map[string]bool, threadNum int, printLog bool) (*Pod, int, int, error) {
	if threadNum < 1 {
		threadNum = 1
	}
	aPod := new(Pod)
	err := aPod.Index(podRoot, repos, exclude)
	if err != nil {
		return nil, 0, 0, err
	}
	cPipe := make(chan bool, threadNum)
	for i := 0; i < threadNum; i++ {
		cPipe <- true
	}
	var success, failure int
	funcAsyncRead := func(v *PodModuleVersion) {
		specPath := path.Join(v.Root, v.FileName)
		aSpec, err := ReadSpec(specPath, printLog)
		if err == nil {
			v.Podspec = aSpec
			success++
		} else {
			v.Err = err
			failure++
		}
		cPipe <- true
	}

	for _, repo := range aPod.PodRepos {
		for _, module := range repo.Modules {
			for _, version := range module.Versions {
				<-cPipe
				go funcAsyncRead(version)
			}
		}
	}

	for i := 0; i < threadNum; i++ {
		<-cPipe
	}
	return aPod, success, failure, nil
}
