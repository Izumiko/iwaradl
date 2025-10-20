# iwara download tool

English, [中文说明](readme.zh_CN.md)

```shell
A downloader for iwara.tv that supports:
- Multiple URLs download
- URL list file
- Resume unfinished downloads
- Custom download directory
- Proxy support

Usage:
  iwaradl [flags] [URL...]
  iwaradl [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  version     Print the version number

Flags:
  -u  --email string        email
  -p  --password string     password
      --auth-token string   authorization token
  -c, --config string       config file (default "config.yaml")
      --debug               enable debug logging
  -h, --help                help for iwaradl
  -l, --list-file string    URL list file
      --max-retry int       max retry times (default -1)
      --proxy-url string    proxy url
  -r, --resume              resume unfinished job
      --root-dir string     root directory for videos
      --thread-num int      concurrent download thread number (default -1)
      --use-sub-dir         use user name as sub directory
      --update-nfo          update nfo files in root directory (--root-dir flag required)
      --update-delay        delay in seconds between updating each nfo file (default: 1)

Use "iwaradl [command] --help" for more information about a command.
```

### config.yaml

```yaml
rootDir: "D:\\MMD" # root dir for videos. or /home/user/MMD in linux
useSubDir: false # use user name as sub dir
email:  "" # email for login
password: "" #  password for login
authorization: "" # token for login, without leading "Bearer "
proxyUrl: "http://127.0.0.1:11081" # proxy url
threadNum: 4 # concurrent download thread num
maxRetry: 3 # max retry times
```

The token can be got by: open the browser console on the iwara webpage, execute `localStorage.getItem("token")`, and the returned value is the token.

URL can be a video page or a user page.

URL list file is a text file, each line is a URL.

To download, either URL or URL list file is required.

Unfinished jobs are saved in `rootDir/jobs.list`, you can use `-r` to resume them.
Finished jobs are saved in `rootDir/history.list`.

Command line arguments have higher priority than config file values.