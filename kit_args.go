package main

import (
	"os"
	"strings"
)

type ArgsBase struct {
	Name    string
	Values  []string
	funcMap map[string]func(*Args)
}

type Args struct {
	ArgsBase
	Subargs []*ArgsBase
}

func NewArgs() *Args {
	args := os.Args
	l := len(args)
	if l < 2 {
		return nil
	}

	mainArgString := args[1]
	if !strings.HasPrefix(mainArgString, "-") {
		return nil
	}
	aArgs := new(Args)
	aArgs.Name = mainArgString
	if l < 3 {
		return aArgs
	}
	aArgs.Values = args[2:]
	subargs := make([]*ArgsBase, 0, len(aArgs.Values)/2+1)
	currentArgs := new(ArgsBase)
	currentArgs.Name = "*"
	currentArgs.Values = make([]string, 0, 2)
	saved := currentArgs
	for _, subitem := range aArgs.Values {
		if strings.HasPrefix(subitem, "--") {
			currentArgs = new(ArgsBase)
			currentArgs.Name = subitem
			currentArgs.Values = make([]string, 0, 2)
			subargs = append(subargs, currentArgs)
		} else {
			currentArgs.Values = append(currentArgs.Values, subitem)
		}
	}
	if len(saved.Values) > 0 {
		subargs = append(subargs, saved)
	}
	if len(subargs) > 0 {
		aArgs.Subargs = subargs
	}
	return aArgs
}

func (s *Args) Print() {
	println("-> " + s.Name + " [" + strings.Join(s.Values, " ") + "]")
	for _, subargs := range s.Subargs {
		println("   - " + subargs.Name + " " + strings.Join(subargs.Values, " "))
	}
}

func (s *Args) CheckSubargsMain() bool {
	return s.CheckSubargs("*")
}

func (s *Args) GetSubargsMain() []string {
	return s.GetSubargs("*")
}

func (s *Args) CheckSubargs(name string) bool {
	for _, subargs := range s.Subargs {
		if subargs.Name == name {
			return true
		}
	}
	return false
}

func (s *Args) GetSubargs(name string) []string {
	for _, subargs := range s.Subargs {
		if subargs.Name == name {
			return subargs.Values
		}
	}
	return nil
}

func (s *Args) GetFirstSubArgsMain() string {
	return s.GetFirstSubArgs("*")
}

func (s *Args) GetFirstSubArgs(name string) string {
	args := s.GetSubargs(name)
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

func (s *Args) RegisterFunc(cmd string, f func(*Args)) {
	if len(cmd) == 0 || f == nil {
		return
	}
	if s.funcMap == nil {
		s.funcMap = make(map[string]func(*Args))
	}
	s.funcMap[cmd] = f
}

func (s *Args) Exec() bool {
	if s.funcMap == nil {
		return false
	}
	f := s.funcMap[s.Name]
	if f == nil {
		return false
	}
	f(s)
	return true
}

func cmd_test_args(aArgs *Args) {
	aArgs.Print()
}
