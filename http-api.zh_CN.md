# iwaradl HTTP API 文档

服务地址示例：

```text
http://127.0.0.1:23456
```

## 鉴权

- 所有 `/api/*` 接口都需要 Bearer Token 鉴权。
- 请求头：

```text
Authorization: Bearer <API_TOKEN>
```

- 鉴权失败时，服务返回 `401 Unauthorized`：

```json
{"error":"unauthorized"}
```

## 任务模型

任务状态值：

- `pending`
- `running`
- `completed`
- `failed`

进度语义：

- `progress` 是 `0~1` 范围内的浮点值
- 由已下载字节计算（`bytesComplete / bytesTotal`）

## 接口列表

### 1) 创建任务

`POST /api/tasks`

请求体：

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

字段说明：

- `urls`（`[]string`，必填）：待加入队列的 URL 列表。
- `options`（`object`，可选）：任务级运行参数。
  - `proxy_url`（`string`）：支持 `http/https/socks5`。
  - `download_dir`（`string`）：支持绝对/相对路径和模板变量。
  - `filename_template`（`string`）：输出文件名模板。
  - `cookie`（`string`）：仅当前任务使用的请求 Cookie。
  - `max_retry`（`int`）：当前任务重试次数。

路径规则：

- `download_dir` 为相对路径时，会拼接 `rootDir`。
- `download_dir` 为绝对路径时，直接使用。

成功响应 `201 Created`：

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

可能错误：

- `400`：JSON 格式错误
- `422`：`urls` 为空
- `422`：未产生有效新任务
- `422`：模板或参数不合法

### 2) 获取单个任务

`GET /api/tasks/{vid}`

成功响应 `200 OK`：

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

可能错误：

- `404`：任务不存在

### 3) 列出所有任务

`GET /api/tasks`

成功响应 `200 OK`：

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

### 4) 删除任务

`DELETE /api/tasks/{vid}`

- 仅允许删除 `pending` 状态任务。

响应：

- `204 No Content`：删除成功

可能错误：

- `404`：任务不存在
- `409`：任务状态不是 `pending`

## 模板变量（Go template 语法）

支持变量：

- `{{now}}`（默认格式 `YYYY-MM-DD`）
- `{{publish_time}}`（默认格式 `YYYY-MM-DD`）
- `{{title}}`
- `{{video_id}}`
- `{{author}}`
- `{{author_nickname}}`
- `{{quality}}`

可显式指定时间格式：

```text
{{now "2006-01-02"}}
{{publish_time "2006-01-02+15.04.05"}}
```

## 外部占位符兼容

API 同时支持将 [IwaraDownloadTool](https://github.com/dawn-lc/IwaraDownloadTool/wiki/%E8%B7%AF%E5%BE%84%E5%8F%AF%E7%94%A8%E5%8F%98%E9%87%8F) 的占位符自动转换为 Go 模板。

示例映射：

- `%#NowTime:YYYY-MM-DD#%` -> `{{now "2006-01-02"}}`
- `%#UploadTime:YYYY-MM-DD+HH.mm.ss#%` -> `{{publish_time "2006-01-02+15.04.05"}}`
- `%#TITLE#%` -> `{{title}}`
- `%#ID#%` -> `{{video_id}}`
- `%#AUTHOR#%` -> `{{author}}`
- `%#ALIAS#%` -> `{{author_nickname}}`
- `%#QUALITY#%` -> `{{quality}}`

因此 `download_dir` 与 `filename_template` 可同时使用两种写法。

## cURL 示例

创建任务：

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

获取任务：

```bash
curl http://127.0.0.1:8080/api/tasks/cgcW74i2Ga4a9w \
  -H "Authorization: Bearer <API_TOKEN>"
```

列出任务：

```bash
curl http://127.0.0.1:8080/api/tasks \
  -H "Authorization: Bearer <API_TOKEN>"
```

删除任务：

```bash
curl -X DELETE http://127.0.0.1:8080/api/tasks/cgcW74i2Ga4a9w \
  -H "Authorization: Bearer <API_TOKEN>"
```
