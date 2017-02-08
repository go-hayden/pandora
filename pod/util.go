package pod

import (
	"regexp"
	"strings"
)

const __REG_NOTE = `^\s*#.*`

func IsSpecNote(line string) bool {
	reg := regexp.MustCompile(__REG_NOTE)
	return reg.MatchString(line)
}

const __REG_END = `^\s*end(\s+(#+.*){0,1}){0,1}$`

func IsEnd(line string) bool {
	reg := regexp.MustCompile(__REG_END)
	return reg.MatchString(line)
}

const __REG_SPEC_SOURCE = `\{\s*:\s*git\s*=>\s*('|")\S+('|")`
const __REG_SPEC_SOURCE_TRIM = `(\{\s*:\s*git\s*=>\s*)|'|"|\s+`

func CheckSource(line string) (string, bool) {
	return CheckCommon(line, __REG_SPEC_SOURCE, __REG_SPEC_SOURCE_TRIM)
}

const __REG_SUBMD = `^\s*\w+\.subspec\s+('|")[^#\s]+('|")`
const __REG_SUBMD_TRIM = `(^\s*\w+\.subspec\s+)|'|"`

func CheckSubmodule(line string) (string, bool) {
	return CheckCommon(line, __REG_SUBMD, __REG_SUBMD_TRIM)
}

const __REG_SPEC_DEP = `^\s*\w+\.dependency\s+('|")[0-9a-zA-Z/]+('|")\s*(,\s*('|")[^#]+('|")){0,1}`
const __REG_SPEC_DEP_TRIM = `(\s*\w+\.dependency\s+)|'|"|\s+`

func CheckDependency(line string) (string, bool) {
	return CheckCommon(line, __REG_SPEC_DEP, __REG_SPEC_DEP_TRIM)
}

const _REG_PF_TARGET = `^\s*target\s+('|")[^#\s]+('|")`
const _REG_PF_TARGET_TRIME = `(^\s*target\s+)|'|"`

func CheckTarget(line string) (string, bool) {
	return CheckCommon(line, _REG_PF_TARGET, _REG_PF_TARGET_TRIME)
}

const __REG_PF_DEP = `^\s*pod\s+['"][0-9a-zA-Z/]+['"]\s*(,\s*(((:path)|(:podspec))\s*=>\s*){0,1}['"][^#]+['"]){0,1}`
const __REG_PF_PATH_DEP = `:path\s*=>`
const __REG_PF_SPEC_DEP = `:podspec\s*=>`
const __REG_PF_DEP_TRIM = `(^\s*pod\s+)|'|"|\s+`

func CheckPodDep(line string) (string, string, string, bool) {
	s, ok := CheckCommon(line, __REG_PF_DEP, __REG_PF_DEP_TRIM)
	if ok {
		var t string
		regPath := regexp.MustCompile(__REG_PF_PATH_DEP)
		regSpec := regexp.MustCompile(__REG_PF_SPEC_DEP)
		if regPath.MatchString(s) {
			t = "p"
			s = Trim(s, __REG_PF_PATH_DEP)
		} else if regSpec.MatchString(s) {
			t = "s"
			s = Trim(s, __REG_PF_SPEC_DEP)
		} else {
			t = "v"
		}

		items := strings.Split(s, ",")
		l := len(items)
		if l > 1 {
			return items[0], items[1], t, true
		} else if l > 0 {
			return items[0], "", t, true
		}
	}
	return "", "", "", false
}

func CheckCommon(line string, find string, trims ...string) (string, bool) {
	reg := regexp.MustCompile(find)
	founds := reg.FindAllString(line, -1)
	if len(founds) > 0 {
		f := founds[0]
		f = Trim(f, trims...)
		return f, true
	}
	return "", false
}

func Trim(line string, trims ...string) string {
	if len(line) == 0 {
		return line
	}
	for _, trim := range trims {
		reg := regexp.MustCompile(trim)
		line = reg.ReplaceAllString(line, "")
	}
	return line
}
