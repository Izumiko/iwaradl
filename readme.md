# iwara download tool
English, [中文说明](readme.zh_CN.md)

```shell
Usage: iwaradl [options] URL1 URL2 ...
Options:
  -c string
        config file (default "config.yaml")
  -l string
        URL list file
  -r    resume unfinished job
  --debug
        enable debug logging
```

### config.yaml
```yaml
rootDir: "D:\\MMD" # root dir for videos. or /home/user/MMD in linux
useSubDir: false # use user name as sub dir
authorization: "" # token for login, without leading "Bearer "
proxyUrl: "http://127.0.0.1:11081" # proxy url
threadNum: 4 # concurrent download thread num
maxRetry: 3 # max retry times
```

URL can be a video page or a user page.

URL list file is a text file, each line is a URL.

To download, either URL or URL list file is required.

Unfinished jobs are saved in `rootDir/jobs.list`, you can use `-r` to resume them.
Finished jobs are saved in `rootDir/history.list`.