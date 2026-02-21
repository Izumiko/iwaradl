# iwaradl HTTP API

Base URL example:

```text
http://127.0.0.1:23456
```

## Auth

- All `/api/*` endpoints require bearer token auth.
- Header:

```text
Authorization: Bearer <API_TOKEN>
```

- If auth fails, server returns `401 Unauthorized`:

```json
{"error":"unauthorized"}
```

## Task Model

Task status values:

- `pending`
- `running`
- `completed`
- `failed`

Progress semantics:

- `progress` is a float in range `0~1`
- It is calculated from downloaded bytes (`bytesComplete / bytesTotal`)

## Endpoints

### 1) Create tasks

`POST /api/tasks`

Request body:

```json
{
  "urls": ["https://www.iwara.tv/video/xxxx", "https://www.iwara.tv/profile/xxx"],
  "options": {
    "proxy_url": "http://127.0.0.1:7890",
    "download_dir": "iwara/{{author_nickname}}",
    "filename_template": "{{publish_time}}-{{title}}-{{video_id}}-{{quality}}",
    "cookie": "...",
    "max_retry": 2
  }
}
```

Fields:

- `urls` (`[]string`, required): URLs to enqueue.
- `options` (`object`, optional): task-level runtime options.
  - `proxy_url` (`string`): supports `http/https/socks5`.
  - `download_dir` (`string`): supports absolute/relative path and template variables.
  - `filename_template` (`string`): output filename template.
  - `cookie` (`string`): request cookie used by this task only.
  - `max_retry` (`int`): retry count for this task.

Path behavior:

- If `download_dir` is relative, it is joined with `rootDir`.
- If `download_dir` is absolute, it is used directly.

Response `201 Created`:

```json
[
  {
    "vid": "cgcW74i2Ga4a9w",
    "status": "pending",
    "progress": 0,
    "created_at": "2026-02-20T12:34:56+08:00",
    "options": {
      "proxy_url": "http://127.0.0.1:7890",
      "download_dir": "D:\\MMD\\iwara\\摸鱼奎恩",
      "cookie_set": true,
      "max_retry": 2,
      "filename_template": "{{publish_time}}-{{title}}-{{video_id}}-{{quality}}"
    }
  }
]
```

Possible errors:

- `400`: invalid JSON
- `422`: `urls` is empty
- `422`: no valid new tasks
- `422`: invalid template or invalid options

### 2) Get one task

`GET /api/tasks/{vid}`

Response `200 OK`:

```json
{
  "vid": "cgcW74i2Ga4a9w",
  "status": "running",
  "progress": 0.42,
  "created_at": "2026-02-20T12:34:56+08:00",
  "options": {
    "proxy_url": "http://127.0.0.1:7890",
    "download_dir": "D:/MMD/iwara/摸鱼奎恩",
    "cookie_set": true,
    "max_retry": 2,
    "filename_template": "{{publish_time}}-{{title}}-{{video_id}}-{{quality}}"
  }
}
```

Possible errors:

- `404`: task not found

### 3) List all tasks

`GET /api/tasks`

Response `200 OK`:

```json
[
  {
    "vid": "id1",
    "status": "completed",
    "progress": 1,
    "created_at": "2026-02-20T12:34:56+08:00",
    "options": {
      "download_dir": "D:/MMD",
      "cookie_set": false,
      "max_retry": 3,
      "filename_template": "{{title}}-{{video_id}}"
    }
  }
]
```

### 4) Delete task

`DELETE /api/tasks/{vid}`

- Only `pending` tasks can be deleted.

Response:

- `204 No Content`: deleted

Possible errors:

- `404`: task not found
- `409`: task is not in `pending`

## Template Variables (Go template syntax)

Supported variables:

- `{{now}}` (default format `YYYY-MM-DD`)
- `{{publish_time}}` (default format `YYYY-MM-DD`)
- `{{title}}`
- `{{video_id}}`
- `{{author}}`
- `{{author_nickname}}`
- `{{quality}}`

You can pass explicit time format:

```text
{{now "2006-01-02"}}
{{publish_time "2006-01-02+15.04.05"}}
```

## External Placeholder Compatibility

The API also supports converting external placeholders from [IwaraDownloadTool](https://github.com/dawn-lc/IwaraDownloadTool/wiki/%E8%B7%AF%E5%BE%84%E5%8F%AF%E7%94%A8%E5%8F%98%E9%87%8F) to Go templates.

Examples:

- `%#NowTime:YYYY-MM-DD#%` -> `{{now "2006-01-02"}}`
- `%#UploadTime:YYYY-MM-DD+HH.mm.ss#%` -> `{{publish_time "2006-01-02+15.04.05"}}`
- `%#TITLE#%` -> `{{title}}`
- `%#ID#%` -> `{{video_id}}`
- `%#AUTHOR#%` -> `{{author}}`
- `%#ALIAS#%` -> `{{author_nickname}}`
- `%#QUALITY#%` -> `{{quality}}`

So both styles are accepted in `download_dir` and `filename_template`.

## cURL Examples

Create task:

```bash
curl -X POST http://127.0.0.1:8080/api/tasks \
  -H "Authorization: Bearer <API_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "urls": ["https://www.iwara.tv/video/cgcW74i2Ga4a9w"],
    "options": {
      "download_dir": "iwara/%#ALIAS#%",
      "filename_template": "%#UploadTime:YYYY-MM-DD#%-%#TITLE#%-%#ID#%-%#QUALITY#%"
    }
  }'
```

Get task:

```bash
curl http://127.0.0.1:8080/api/tasks/cgcW74i2Ga4a9w \
  -H "Authorization: Bearer <API_TOKEN>"
```

List tasks:

```bash
curl http://127.0.0.1:8080/api/tasks \
  -H "Authorization: Bearer <API_TOKEN>"
```

Delete task:

```bash
curl -X DELETE http://127.0.0.1:8080/api/tasks/cgcW74i2Ga4a9w \
  -H "Authorization: Bearer <API_TOKEN>"
```
