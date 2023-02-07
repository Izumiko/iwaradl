package main

import (
	"flag"
	"iwaradl/config"
	"os"
	"strings"
	"time"
)

var cliFlag struct {
	configFile string
	listFile   string
	resumeJob  bool
}

var urlList []VideoInfo

func init() {
	flag.StringVar(&cliFlag.configFile, "c", "config.yaml", "config file")
	flag.StringVar(&cliFlag.listFile, "l", "", "URL list file")
	flag.BoolVar(&cliFlag.resumeJob, "r", false, "resume unfinished job")
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
	config.LoadConfig(&config.Cfg, cliFlag.configFile)
	if cliFlag.resumeJob {
		urlList = LoadUrlList()
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
		processUrlList(urls)
	}
	SaveUrlList(urlList)

	failed := len(urlList)
	for i := 0; i < config.Cfg.MaxRetry && failed > 0; i++ {
		failed = ConcurrentDownload()
		if i < config.Cfg.MaxRetry-1 {
			time.Sleep(30 * time.Second)
		}
	}

}
