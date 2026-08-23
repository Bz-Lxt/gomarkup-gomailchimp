# QA_Record

## Round 1 — 2026-08-23 17:55 GMT+8

**Cost: ¥0**（全程 Mailpit / 离线，无 SES 调用）

### 执行环境

- `docker compose up --build -d` 后 `docker compose --profile qa run --rm qa`
- 宿主端口：控制台 27481 / API 27482 / 追踪 27483 / Mailpit 27484 / 退订 27485
- 初次 27181–27184 被同机 `gotravel` 占用，已改派

### 结果

| 项 | 结果 |
|---|---|
| Docker Build | PASS |
| Health Check | PASS `/healthz` `/readyz` |
| Auth + Viewer 403 | PASS |
| 列表 / 活动 / 管道 | PASS |
| 真实 SMTP 群发 | PASS 4/4 进入 Mailpit |
| 漏斗聚合 | PASS sent=4 delivered=4 |
| 打开像素 | PASS 200 GIF |
| 通道隔离单测 | PASS `TestChannelIsolation` |
| 令牌桶 / 退信状态机 / HMAC / 窄化重试 / 渲染转义 | PASS `go test ./...` |
| Playwright E2E | 未在容器内执行；浏览器手测登录→总览→活动台 |

### 修复

1. `pg_advisory_lock` 打在连接池随机连接上，tracker/sender 卡死无法听端口。改为 `sql.DB.Conn` 会话锁。
2. QA 冒烟用 `tenant["Name"]` 触发 KeyError，改为 `.get`。
3. 点击 token 含 `.`，网关路由改为 `/o/*token` `/c/*token`。

### 未在本轮压测的基线

500 封/分钟吞吐、网关 1000 QPS、重启丢件=0 未做满载压测，单测与 4 封闭环已覆盖逻辑路径。
