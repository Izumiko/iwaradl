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
  genlist     Generate a filtered Iwara video URL list
  help        Help about any command
  serve       start iwara downloading daemon
  version     Print the version number

Flags:
  -u  --email string              email
  -p  --password string           password
      --api-token string          token for daemon HTTP API authentication
      --auth-token string         authorization token
  -c, --config string             config file (default "config.yaml")
      --debug                     enable debug logging
  -h, --help                      help for iwaradl
  -l, --list-file string          URL list file
      --filename-template string  output filename template
      --max-retry int             max retry times (default -1)
      --proxy-url string          proxy url
  -r, --resume                    resume unfinished job
      --root-dir string           root directory for videos
      --thread-num int            concurrent download thread number (default -1)
      --use-sub-dir               use user name as sub directory
      --update-nfo                update nfo files in root directory (--root-dir flag required)
      --update-delay              delay in seconds between updating each nfo file (default: 1)

Use "iwaradl [command] --help" for more information about a command.
```

### Generate video list (`genlist`)

`genlist` fetches video list pages from Iwara, filters videos by date/likes/views/duration, then writes final video URLs to a text file.

Example:

```shell
iwaradl genlist --sort date --page-limit 3 --date-limit 14 --output videolist.txt
iwaradl genlist --rating all --filter-like0 200 --filter-like-inc 20 --filter-duration 120
```

Main flags:

- `--sort`: `date`, `trending`, `popularity`, `views`, `likes`
- `--page-limit`: number of pages to fetch, must be `> 0`
- `--date-limit`: only keep videos from last N days, must be `> 0`
- `--rating`: `all`, `general`, `ecchi`
- `--filter-like0`: base minimum likes, must be `>= 0`
- `--filter-like-inc`: extra required likes per day, must be `>= 0`
- `--filter-views`: minimum views, must be `>= 0`
- `--filter-duration`: minimum duration in seconds, must be `> 0`
- `--output`: output file path, cannot be empty

### Daemon mode

Start daemon:

```shell
iwaradl serve --bind 127.0.0.1 --port 23456 --config config.yaml
```

`--api-token` (or env `IWARADL_API_TOKEN`) is required in daemon mode.
`--bind` defaults to `127.0.0.1`.

API endpoints:

- `POST /api/tasks` add download tasks
- `GET /api/tasks` list all tasks
- `GET /api/tasks/{vid}` get one task
- `DELETE /api/tasks/{vid}` delete one pending task

Details see [API doc](http-api.md).

Create task example:

```shell
curl -X POST http://127.0.0.1:23456/api/tasks \
  -H "Authorization: Bearer <YOUR_API_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "urls":["https://www.iwara.tv/video/xxxx"],
    "options":{
      "download_dir":"daily",
      "proxy_url":"http://127.0.0.1:7890",
      "max_retry":2,
      "filename_template":"{{title}}-{{video_id}}-{{quality}}"
    }
  }'
```

`options.download_dir` supports both relative and absolute paths. Relative paths are joined with `rootDir`.
`options.download_dir` also supports the same template variables, e.g. `iwara/{{author_nickname}}`.

Filename template variables:

- `{{now}}` (YYYY-MM-DD)
- `{{publish_time}}` (YYYY-MM-DD)
- `{{title}}`
- `{{video_id}}`
- `{{author}}`
- `{{author_nickname}}`
- `{{quality}}`

### config.yaml

```yaml
rootDir: "D:\\MMD" # root dir for videos. or /home/user/MMD in linux
useSubDir: false # use user name as sub dir
email:  "" # email for login
password: "" #  password for login
authorization: "" # token for login, without leading "Bearer "
apiToken: "" # token used by daemon HTTP API auth
proxyUrl: "http://127.0.0.1:11081" # proxy url
filenameTemplate: "{{title}}-{{video_id}}" # output filename template
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
