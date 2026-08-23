# API.md — Lumen Relay

Base: `http://localhost:27482/api/v1`  
认证：`Authorization: Bearer <access>`。跨租户 ID 一律 **404**。

## 信封

成功：`{ "data": ..., "meta": { "total", "page", "per_page" } }`  
失败：`{ "error": { "code", "message" } }`

## 错误码

| code | HTTP | 含义 |
|---|---|---|
| unauthorized | 401 | 未登录 / token 无效 |
| forbidden | 403 | Viewer 写操作 |
| not_found | 404 | 资源不存在或跨租户 |
| validation_error | 422 | 参数 / 状态机非法 |
| conflict | 409 | 冲突 |
| quota_exceeded | 429 | 配额 |
| internal_error | 500 | 内部错误 |

## 端点

### POST /auth/login

```json
{ "email": "owner@lumen.local", "password": "Owner123!" }
```

```json
{ "data": { "tokens": { "access_token": "jwt", "refresh_token": "jwt", "expires_in": 7200 }, "user": { "email": "owner@lumen.local", "role": "owner" } } }
```

### POST /auth/refresh

`{ "refresh_token": "..." }` → 同 login。

### GET /me

返回当前用户与租户。

### GET /contacts?q=&page=&per_page=

分页联系人。

### POST /lists

`{ "name": "核心客群" }` → 201

### POST /contacts/import  (multipart)

`file` + `list_id`。坏行进入 `error_csv`，不阻断整批。

### POST /templates

```json
{ "name": "唤醒", "subject": "Hi, {{ .UserName }}", "ast": { "width": 600, "blocks": [{ "type": "text", "html": "Hi, {{ .UserName }}" }] } }
```

### POST /campaigns

```json
{ "name": "八月", "from_name": "北极星", "from_email": "hello@lumen.local", "subject": "灯还亮着", "list_id": "uuid", "template_ver_id": "uuid", "strategy": "immediate" }
```

### POST /campaigns/:id/action

`{ "action": "start|pause|resume|cancel|schedule" }`

### GET /campaigns/:id/funnel

五级漏斗快照。`unique_opened` / `machine_open` 分列。

### GET /campaigns/:id/stream

SSE `event: funnel`。可用 `?access_token=`（EventSource 无法带 Header）。

### GET /pipeline/stats

`{ "send", "delay", "dlq", "provider" }`

### POST /public/unsub

`{ "token": "<hmac>" }`

### 追踪网关 `http://localhost:27483`

- `GET /o/{token}.gif` → 1x1 GIF，异步记打开
- `GET /c/{token}` → 记录点击后 302，目标来自签名 token（禁止开放重定向）
