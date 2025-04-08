# iwara 下载工具

[English](readme.md), 中文说明

```shell
用法: iwaradl [options] URL1 URL2 ...
选项:
  -c string
        config file (default "config.yaml")
  -l string
        待下载视频网址列表文件
  -r    继续未完成的下载任务
  --debug
        启用调试日志输出
  --root-dir string
        存放视频的根目录
  --use-sub-dir
        是否根据作者创建子目录
  --auth-token string
        登录时用到的token
  --proxy-url string
        代理地址
  --thread-num int
        concurrent download thread number (default 3)
  --max-retry int
        max retry times (default 3)
```

### config.yaml

```yaml
rootDir: "D:\\MMD" # 存放视频的目录；mac/Linux下填完整路径，如/home/user/MMD
useSubDir: false # 是否根据作者创建子目录
authorization: "" # 登录时用到的token，不好含开头的"Bearer "
proxyUrl: "http://127.0.0.1:11081" # 代理地址
threadNum: 4 # 同时进行的任务数
maxRetry: 3 # 最大尝试下载次数
```

token获取方式如下：打开iwara网页的浏览器控制台，执行`localStorage.getItem("token")`，返回值即为token。

视频网址可以是一个视频的页面，也可以是用户页面（将下载该用户所有投稿视频）。

视频网址列表文件是一个纯文本文件，每行一个网址。

使用时，命令行URL或者列表文件至少提供一个。

未完成的任务列表存放在`rootDir/jobs.list`，可以使用 `-r` 来继续。已完成的任务记录存放在`rootDir/history.list`中。

命令行参数的优先级高于配置文件中的值。
