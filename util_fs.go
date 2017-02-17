package main

import (
	"errors"
	"path"
	"strings"

	"os"

	"io/ioutil"

	"github.com/go-hayden-base/fs"
)

func Mkdir(dir string, mode os.FileMode) error {
	dir = strings.TrimSpace(dir)
	if dir == "" || !path.IsAbs(dir) {
		return errors.New("请正确设置目录的绝对路径！")
	}
	if !fs.DirectoryExists(dir) {
		e := Mkdir(path.Dir(dir), mode)
		if e != nil {
			return e
		}
		return os.Mkdir(dir, mode)
	}
	return nil
}

func WriteFile(filePath string, data []byte, cover bool, mode os.FileMode) error {
	if data == nil {
		data = make([]byte, 0, 0)
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" || !path.IsAbs(filePath) {
		return errors.New("请正确设置文件的绝对路径！")
	}
	if !cover && fs.FileExists(filePath) {
		return errors.New("文件存在 [" + filePath + "]")
	}
	e := Mkdir(path.Dir(filePath), mode)
	if e != nil {
		return e
	}
	e = ioutil.WriteFile(filePath, data, mode)
	if e != nil {
		return e
	}
	return nil
}
