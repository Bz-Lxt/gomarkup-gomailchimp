# API Contracts — Contract Gate

> 时间：2026-08-23 17:54 (GMT+8)

## Mailpit SMTP — VERIFIED

对容器内 `mailpit:1025` 发出一封真实 SMTP 邮件（无 STARTTLS，Mailpit 接受任意 AUTH）。

| 项 | 实测 |
|---|---|
| 握手 | TCP 直连 + `smtp.NewClient`，无 TLS |
| 成功码 | `250` / `Accepted=true` |
| 失败形态 | `net/smtp` 错误字符串内嵌三位码（如 550/421）；`extractCode` 解析 |
| 4xx | 标 Transient，窄化重试 |
| 5xx / 535 | 不重试；535 标 AuthFailed |
| 证据 | 活动 `ae9784a2-...` 4 封入 Mailpit UI `http://localhost:27484`，漏斗 `sent=4 delivered=4` |
| 头 | `List-Unsubscribe` + `List-Unsubscribe-Post` 已出现在报文 |

## 追踪网关 — VERIFIED（打开）/ 部分（点击）

- `GET /o/{token}.gif` 返回 `image/gif` 1x1，HTTP 200。像素命中后异步记 `open`。
- 点击链路将外链改写为 `GET /c/{token}`。token 含 `.`，路由改为 `/c/*token` 以防截断。302 目标只来自 HMAC payload，禁止 query 直跳。

## AWS SES — UNVERIFIED

无 `AWS_ACCESS_KEY_ID`。`SESSender` 在缺凭据时拒绝臆造成功响应，返回官方文档中的 `InvalidClientTokenId` 形状：

来源：https://docs.aws.amazon.com/ses/latest/APIReference/API_SendRawEmail.html  
SNS 退信形状：https://docs.aws.amazon.com/ses/latest/dg/notification-contents.html

设置 `MAIL_PROVIDER=ses` 且填入密钥后才会走真实 `SendRawEmail`。

## BounceSource

| 实现 | 状态 |
|---|---|
| SMTP 会话码 | VERIFIED（与发送路径同一条） |
| Mock Feeder (SES SNS JSON) | 已接线，QA 离线用 |
| IMAP / Webhook | 接口在，无公网邮箱时 no-op |
