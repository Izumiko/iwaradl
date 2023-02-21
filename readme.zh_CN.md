# iwara 下载工具
[English](readme.md), 中文说明

```shell
用法: iwaradl [options] URL1 URL2 ...
选项:
  -c string
        配置文件名 (默认 "config.yaml")
  -l string
        待下载视频网址列表文件
  -r    继续未完成的下载任务
```

### config.yaml
```yaml
rootDir: "D:\\MMD" # 存放视频的目录；mac/Linux下填完整路径，如/home/user/MMD
useSubDir: false # 是否根据作者创建子目录
cookie: "" # 登录之后的cookie
proxyUrl: "http://127.0.0.1:11081" # 代理地址
threadNum: 4 # 同时进行的任务数
maxRetry: 3 # 最大尝试下载次数
```

视频网址可以是一个视频的页面，也可以是用户页面（将下载该用户所有投稿视频）。

视频网址列表文件是一个纯文本文件，每行一个网址。

使用时，命令行URL或者列表文件至少提供一个。

未完成的任务列表存放在`rootDir/jobs.yaml`，可以使用 `-r` 来继续。已完成的任务记录存放在`rootDir/history.list`中。
