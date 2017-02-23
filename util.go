package main

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

func CurrentDir() (string, bool) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", false
	}
	return dir, true
}

func AbsPath(dir string, p string) string {
	p = strings.TrimSpace(p)
	if path.IsAbs(p) {
		return p
	}

	dir = strings.TrimSpace(dir)
	if dir != "" {
		return path.Join(dir, p)
	}

	d, ok := CurrentDir()
	if ok {
		p = path.Join(d, p)
	}
	return p
}

func exitWithMessage(msg string, p bool) {
	printRed(msg, p)
	os.Exit(1)
}

func ParseBinaryString(s string) (int, error) {
	s = strings.TrimSpace(s)
	if !regexp.MustCompile(`^[0-1]+$`).MatchString(s) {
		return 0, errors.New("字符串必须仅为0何1的组合！")
	}
	res := 0
	l := len(s)
	for idx, c := range s {
		offset := l - idx - 1
		if c == '1' {
			res = res | (1 << uint(offset))
		}
	}
	return res, nil
}
