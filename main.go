package main

import (
	"database/sql"
	"os"
	"path"
	"strings"

	"time"

	"strconv"

	"github.com/go-hayden-base/cfg"
	"github.com/go-hayden-base/fs"
)

var config cfg.Config
var db *sql.DB
var _funcMap map[string]func([]string)

func init() {
	// 获取工作目录
	p := os.Getenv("PANDORA_PATH")
	if strings.TrimSpace(p) == "" {
		printRed("请配置环境变量PANDORA_PATH，用以指定pandora的工作目录!", true)
		os.Exit(0)
	}

	// 读取配置
	cfgp := path.Join(p, "pandora.cfg")
	if !fs.FileExists(cfgp) {
		printRed("配置文件"+cfgp+"不存在！", true)
		os.Exit(0)
	}
	config = make(cfg.Config)
	err := config.InitWithConfigFile(cfgp)
	if err != nil {
		printRed(err.Error(), true)
		os.Exit(0)
	}
	config["path"] = p

	// 初始化数据库
	err = initDB()
	if err != nil {
		printRed(err.Error(), true)
		os.Exit(0)
	}

	// 初始化方法映射
	_funcMap = make(map[string]func([]string))
	_funcMap["--sync"] = startSync
	_funcMap["--dep"] = depend
	_funcMap["--diff"] = diff
}

func main() {
	l := len(os.Args)
	if l < 2 {
		println("请指定参数！")
		printHelp()
		return
	}
	f, ok := _funcMap[os.Args[1]]
	if ok {
		start := time.Now().UnixNano()

		var args []string
		if l > 2 {
			args = os.Args[2:]
		}
		f(args)

		end := time.Now().UnixNano()
		cost := float64(end-start) / float64(1000000000)
		println("程序耗时:  " + strconv.FormatFloat(cost, 'f', -1, 64) + " 秒")
	}
}
