package main

import (
	"flag"
	"iwaradl/config"
	"iwaradl/util"
	"os"
	"strings"
	"time"
)

var cliFlag struct {
	configFile string
	listFile   string
	resumeJob  bool
	debug      bool
	rootDir    string
	useSubDir  bool
	auth       string
	proxyUrl   string
	threadNum  int
	maxRetry   int
}

var vidList []string

func init() {
	flag.StringVar(&cliFlag.configFile, "c", "config.yaml", "config file (default \"config.yaml\")")
	flag.StringVar(&cliFlag.listFile, "l", "", "URL list file")
	flag.BoolVar(&cliFlag.resumeJob, "r", false, "resume unfinished job")
	flag.BoolVar(&cliFlag.debug, "debug", false, "enable debug logging")
	flag.StringVar(&cliFlag.rootDir, "root-dir", "", "root directory for videos")
	flag.BoolVar(&cliFlag.useSubDir, "use-sub-dir", false, "use user name as sub directory")
	flag.StringVar(&cliFlag.auth, "auth-token", "", "authorization token")
	flag.StringVar(&cliFlag.proxyUrl, "proxy-url", "", "proxy url")
	flag.IntVar(&cliFlag.threadNum, "thread-num", -1, "concurrent download thread number (default 3)")
	flag.IntVar(&cliFlag.maxRetry, "max-retry", -1, "max retry times (default 3)")
	flag.Usage = usage
}

func usage() {
	println("Usage: iwaradl [options] URL1 URL2 ...")
	println("Options:")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()
	if !cliFlag.resumeJob && flag.NArg() == 0 && cliFlag.listFile == "" {
		flag.Usage()
		return
	}
	err := config.LoadConfig(&config.Cfg, cliFlag.configFile)
	if err != nil {
		panic(err)
	}

	// 命令行参数优先级高于配置文件
	if cliFlag.rootDir != "" {
		config.Cfg.RootDir = cliFlag.rootDir
	}
	if cliFlag.useSubDir {
		config.Cfg.UseSubDir = cliFlag.useSubDir
	}
	if cliFlag.auth != "" {
		config.Cfg.Authorization = cliFlag.auth
	}
	if cliFlag.proxyUrl != "" {
		config.Cfg.ProxyUrl = cliFlag.proxyUrl
	}
	if cliFlag.threadNum > 0 {
		config.Cfg.ThreadNum = cliFlag.threadNum
	}
	if cliFlag.maxRetry > 0 {
		config.Cfg.MaxRetry = cliFlag.maxRetry
	}

	if cliFlag.debug {
		util.Debug = true
	}
	if cliFlag.resumeJob {
		vidList = LoadVidList()
	}
	if flag.NArg() > 0 {
		processUrlList(flag.Args())
	}
	if cliFlag.listFile != "" {
		_, err := os.Stat(cliFlag.listFile)
		if err != nil {
			println(err.Error())
			return
		}
		data, err := os.ReadFile(cliFlag.listFile)
		if err != nil {
			println(err.Error())
			return
		}
		urls := strings.Split(string(data), "\n")
		for i, v := range urls {
			urls[i] = strings.TrimRight(v, "\r")
		}
		processUrlList(urls)
	}
	SaveVidList(vidList)

	failed := len(vidList)
	for i := 0; i < config.Cfg.MaxRetry && failed > 0; i++ {
		failed = ConcurrentDownload()
		if failed > 0 && i < config.Cfg.MaxRetry-1 {
			time.Sleep(30 * time.Second)
		}
	}

}
