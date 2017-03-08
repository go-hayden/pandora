package main

import "github.com/go-hayden-base/pod"

func cmd_test_spec(aArgs *Args) {
	p := aArgs.GetFirstSubArgs("--p")
	spec, e := pod.ReadSpec(p)
	if e != nil {
		printRed(e.Error(), false)
	} else {
		if b, e := spec.JSON(); e != nil {
			printRed(e.Error(), false)
		} else {
			println(string(b))
		}
	}

	search := aArgs.GetFirstSubArgs("--s")
	if len(search) > 0 {
		println("======= Search ======")
		m := spec.GetAllDepends(search)
		if m != nil {
			for key, val := range m {
				println(key, val)
			}
		} else {
			println("No dependencies!")
		}
	}
}
