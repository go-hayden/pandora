package main

import (
	"database/sql"
	"os"

	"time"

	"strconv"
)

var _Conf *Config
var _DB *sql.DB
var _Args *Args

func init() {
	// 获取配置
	cfg, err := NewConfig()
	if err != nil {
		printRed(err.Error(), true)
		os.Exit(1)
	}
	_Conf = cfg

	// 初始化数据库
	err = initDB()
	if err != nil {
		printRed(err.Error(), true)
		os.Exit(0)
	}

	// 初始化方法映射
	_Args = NewArgs()
	if _Args == nil {
		printRed("无法解析参数！", true)
		os.Exit(0)
	}
	_Args.RegisterFunc("-sync", cmd_sync)
	_Args.RegisterFunc("-dep", cmd_depend)
	_Args.RegisterFunc("-up", cmd_upgrade)
	_Args.RegisterFunc("-origin", cmd_origin)
	_Args.RegisterFunc("-sql", cmd_exec_sql)
	if _Conf.IsDebug() {
		_Args.RegisterFunc("-test_args", cmd_test_args)
		_Args.RegisterFunc("-test_spec", cmd_test_spec)
	}
}

func main() {
	l := len(os.Args)
	if l < 2 {
		println("请指定参数！")
		printHelp()
		return
	}
	start := time.Now().UnixNano()
	ok := _Args.Exec()
	if !ok {
		printRed("未执行任何操作！", false)
		printHelp()
	}
	end := time.Now().UnixNano()
	cost := float64(end-start) / float64(1000000000)
	println("程序耗时:  " + strconv.FormatFloat(cost, 'f', -1, 64) + " 秒")
}
