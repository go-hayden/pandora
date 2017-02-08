package main

import (
	"pandora/pod"
)

func diff(args []string) {
	if len(args) == 0 {
		printRed("参数错误!", false)
		printHelp()
		return
	}
	pf := args[0]
	p := new(pod.Podfile)
	p.FilePath = pf
	e := p.Read()
	if e != nil {
		println(e)
	} else {
		p.Print()
	}
}
