# iwara 下载工具

[English](readme.md), 中文说明

```shell
iwara.tv下载器支持功能：
- 多URL下载
- URL列表文件
- 断点续传
- 自定义下载目录
- 代理支持

使用方法：
  iwaradl [参数] [URL...]
  iwaradl [命令]

可用命令：
  completion  为指定shell生成自动补全脚本
  help        查看命令帮助
  serve       启动守护进程模式
  version     打印版本号

参数说明：
  -u  --email string        登录邮箱
  -p  --password string     登录密码
      --auth-token string   授权令牌
  -c, --config string       配置文件路径（默认为"config.yaml"）
      --debug               启用调试日志
  -h, --help                显示帮助信息
  -l, --list-file string    URL列表文件路径
      --max-retry int       最大重试次数（默认自动调整）
      --proxy-url string    代理服务器地址
  -r, --resume              恢复未完成的任务
      --root-dir string     视频存储根目录
      --thread-num int      并发下载线程数（默认自动调整）
      --use-sub-dir         使用用户名作为子目录
      --update-nfo          更新指定根目录下的nfo文件（--root-dir必须指定）
      --update-delay        nfo文件更新间隔时间，单位秒（默认: 1）

使用"iwaradl [命令] --help"查看具体命令帮助信息。
```

### 守护进程模式

启动 daemon：

```shell
iwaradl serve --port 23456 --config config.yaml
```

API 接口：

- `POST /api/tasks` 提交下载任务
- `GET /api/tasks` 查看全部任务
- `GET /api/tasks/{vid}` 查看单个任务
- `DELETE /api/tasks/{vid}` 删除单个待处理任务（仅 `pending` 可删除）

提交任务示例：

```shell
curl -X POST http://127.0.0.1:23456/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"urls":["https://www.iwara.tv/video/xxxx"]}'
```

### config.yaml

```yaml
rootDir: "D:\\MMD" # 存放视频的目录；mac/Linux下填完整路径，如/home/user/MMD
useSubDir: false # 是否根据作者创建子目录
email:  "" # 登录邮箱
password: "" # 登录密码
authorization: "" # 登录时用到的token，不含开头的"Bearer "
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
