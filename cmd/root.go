package cmd

import (
	"iwaradl/config"
	"iwaradl/util"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
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
	vidList    []string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "iwaradl [flags] [URL...]",
	Short: "A downloader for iwara.tv",
	Long: `A downloader for iwara.tv that supports:
- Multiple URLs download
- URL list file
- Resume unfinished downloads
- Custom download directory
- Proxy support`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !resumeJob && len(args) == 0 && listFile == "" {
			return cmd.Help()
		}

		err := config.LoadConfig(&config.Cfg, configFile)
		if err != nil {
			return err
		}

		// 命令行参数优先级高于配置文件
		if rootDir != "" {
			config.Cfg.RootDir = rootDir
		}
		if useSubDir {
			config.Cfg.UseSubDir = useSubDir
		}
		if auth != "" {
			config.Cfg.Authorization = auth
		}
		if proxyUrl != "" {
			config.Cfg.ProxyUrl = proxyUrl
		}
		if threadNum > 0 {
			config.Cfg.ThreadNum = threadNum
		}
		if maxRetry > 0 {
			config.Cfg.MaxRetry = maxRetry
		}

		if debug {
			util.Debug = true
		}

		// 处理下载任务
		if resumeJob {
			vidList = LoadVidList()
		}
		if len(args) > 0 {
			processUrlList(args)
		}
		if listFile != "" {
			_, err := os.Stat(listFile)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(listFile)
			if err != nil {
				return err
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

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yaml", "config file")
	rootCmd.PersistentFlags().StringVarP(&listFile, "list-file", "l", "", "URL list file")
	rootCmd.PersistentFlags().BoolVarP(&resumeJob, "resume", "r", false, "resume unfinished job")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
	rootCmd.PersistentFlags().StringVar(&rootDir, "root-dir", "", "root directory for videos")
	rootCmd.PersistentFlags().BoolVar(&useSubDir, "use-sub-dir", false, "use user name as sub directory")
	rootCmd.PersistentFlags().StringVar(&auth, "auth-token", "", "authorization token")
	rootCmd.PersistentFlags().StringVar(&proxyUrl, "proxy-url", "", "proxy url")
	rootCmd.PersistentFlags().IntVar(&threadNum, "thread-num", -1, "concurrent download thread number (default 3)")
	rootCmd.PersistentFlags().IntVar(&maxRetry, "max-retry", -1, "max retry times (default 3)")
}
