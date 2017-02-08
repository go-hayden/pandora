package main

import cp "github.com/fatih/color"

func printRed(msg string, showTip bool) {
	cp.Red(msg)
	if showTip {
		printTip()
	}
}

func printTip() {
	println("详情请参见：")
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
