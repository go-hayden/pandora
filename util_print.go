package main

import (
	"fmt"

	cp "github.com/fatih/color"
)

const (
	__COLOR_DEFAULT = iota
	__COLOR_RED
	__COLOR_GREEN
	__COLOR_BLUE
	__COLOR_YELLOW
)

func printColor(colorTag int, showHelp bool, msg string) {
	switch colorTag {
	case __COLOR_RED:
		cp.Red(msg)
	case __COLOR_GREEN:
		cp.Green(msg)
	case __COLOR_BLUE:
		cp.Green(msg)
	case __COLOR_YELLOW:
		cp.Green(msg)
	default:
		println(msg)
	}
	if showHelp {
		printHelp()
	}
}

func printRed(msg string, showHelp bool) {
	printColor(__COLOR_RED, showHelp, msg)
}

func printGreen(msg string, showHelp bool) {
	printColor(__COLOR_GREEN, showHelp, msg)
}

func printBlue(msg string, showHelp bool) {
	printColor(__COLOR_BLUE, showHelp, msg)
}

func printYellow(msg string, showHelp bool) {
	printColor(__COLOR_YELLOW, showHelp, msg)
}

func printlnDebug(s ...interface{}) {
	if _Conf.IsDebug() {
		tmp := make([]interface{}, 1, len(s)+1)
		tmp[0] = "DEBUG =>"
		tmp = append(tmp, s...)
		fmt.Println(tmp...)
	}
}

const __help_info = `
****** 参数帮助 ******
--sync           :索引本地Pod并同步到数据库，建议先执行pod repo update命令更新本地Pod仓库
--dep            :查询某版本的模块所有依赖，例如: pandora --dep NVNetwork 1.0.3

详情请参考：
**********************

`

func printHelp() {
	print(__help_info)
}
