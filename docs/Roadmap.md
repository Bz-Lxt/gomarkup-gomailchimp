# Roadmap.md — GoMailchimp / Lumen Relay

> **WHEN** 文档。WHAT 以 `docs/Requirements.md` 为准。
>
> | 项 | 值 |
> |---|---|
> | 版本 | v1.0 |
> | 冻结时间 | 2026-08-23 17:39 (GMT+8) |
> | 负责 Agent | Chief Architect |
> | 预估规模 | Tier 2 · 14k–17k LoC |
> | 阶段顺序 | **Logic-First（Phase 3 → Phase 2）** |

---

## 0. Phase Order Decision

**决定：Logic-First（互换 SOP 默认的 Phase 2 / Phase 3）。**

**一句话理由：** 邮件拖拽编辑器与漏斗大屏都是 canvas/visualization 型 UI，组件树、属性面板、撤销栈、漏斗口径全部派生自模板 AST 与 `email_events` 事件模型；先定 schema/渲染器/状态机，再画 UI，避免返工。

---

## 1. 架构冻结

### 1.1 产品名与目录

| 目录 | 职责 |
|---|---|
| `backend/` | Go 单体代码库，三个进程：`api` / `sender` / `tracker` |
| `frontend-admin/` | 控制台 SPA（编辑器 / 活动 / 漏斗 / 联系人） |
| `frontend-user/` | 公网落地页（退订确认、投诉说明） |
| `frontend-mp/` | **不创建**。本产品无微信小程序端，记录于此以免结构歧义 |

### 1.2 ORM / 迁移

- ORM：**GORM**（开发速度 + 事务；核心查询手写 SQL 兜底）
- 迁移：**golang-migrate** SQL 文件，启动时 `pg_advisory_lock` 串行执行（防双副本 23505）
- 队列：自研 Redis List + ZSet 延迟队列，**不用 asynq**
- 限速：自研 Redis Lua 令牌桶，时钟源只用 `TIME`

### 1.3 进程与端口（Dev 随机，已二次探测空闲）

| 服务 | 容器端口 | 宿主端口 | 用途 |
|---|---|---|---|
| frontend-admin | 80 | **27481** | 控制台 |
| api | 8080 | **27482** | REST + SSE |
| tracker | 8081 | **27483** | `/o/{token}.gif` `/c/{token}` |
| mailpit UI | 8025 | **27484** | 收件箱证据 |
| frontend-user | 80 | **27485** | 退订落地页 |
| postgres / redis / smtp | 内部 | 不暴露 | 仅 compose 网络 |

`/deploy` 阶段再标准化到 8081+。

### 1.4 包分层（backend/internal）

```
clock / config / logger
domain          实体、状态机、错误
model           GORM 映射
repo            数据访问（强制注入 tenant_id）
auth            JWT / 密码
render          AST → table+inline HTML + 占位符
token           HMAC 追踪 token
limiter         令牌桶（memory 对照 + redis lua）
pipeline        queue / worker pool / scheduler / retry
provider        Sender 接口：smtp / ses / mock
bounce          分类、状态机、BounceSource
service         用例编排
httpx           统一响应、中间件
httpapi         路由与 handler
tracker         像素与 302 网关
```

---

## 2. MVP / V1 / V2 边界

### MVP（本轮 `/auto` 必须交付，可跑通闭环）

- 多租户 + JWT + Owner/Marketer/Viewer（Viewer 只读）
- 联系人 CSV/Excel 导入（逐行校验 + 错误报告）
- 模板 AST 四组件（文本/图片/按钮/分割线）+ Go template 占位符 + 安全转义
- 活动 CRUD + 立即/定时/分批策略 + 状态机
- 自研 Worker Pool + Redis 队列 + 优雅关闭
- 三维度 Redis 令牌桶 + Gmail/Outlook 通道隔离
- 追踪网关：像素打开 + 外链 302 + HMAC token
- 同步 SMTP 退信解析 + 黑名单必查 + RFC 8058 一键退订
- 基础漏斗大屏（SSE，五级漏斗）
- Docker 一键起，Mailpit 可见实信
- 核心引擎单测 + API smoke

### V1（本轮尽量做完，作为验收基线）

- 渐进式 ramp-up、通道熔断与加权负载均衡、SMTP 连接池
- DLQ、重启续发、幂等 message_id
- BounceSource：IMAP / SES webhook / Mock Feeder
- 机器打开识别、信誉度看板与自动暂停
- 完整大屏（速率曲线、通道健康、Top 链接）
- `docs/API.md`、Playwright E2E
- §7 量化基线实测

### V2（加分，不挡验收）

- Prometheus 指标页、审计日志 UI、模板市场、`--scale sender=3` 演示脚本

---

## 3. 任务分解（Logic-First）

| ID | 任务 | 阶段 | 负责 |
|---|---|---|---|
| A1 | Git / ignore / compose 骨架 / 双架构镜像校验 | P1 | Architect |
| L1 | 时钟、日志、配置、错误码、GORM 模型、迁移 | P3 | Logic |
| L2 | Auth + 租户仓储 + seed | P3 | Logic |
| L3 | 联系人导入 / 客群 / 黑名单 | P3 | Logic |
| L4 | 模板 AST 渲染器 + HMAC token | P3 | Logic |
| L5 | 令牌桶（单测 + Redis Lua） | P3 | Logic |
| L6 | Worker Pool + 队列 + Provider（smtp/ses/mock） | P3 | Logic |
| L7 | 活动编排、预检、调度、退信状态机 | P3 | Logic |
| L8 | 追踪网关 + SSE 漏斗聚合 | P3 | Logic |
| L9 | HTTP API + Dockerfile + Contract Gate | P3 | Logic |
| U1 | DesignSpec + shadcn 骨架 | P2 | UI |
| U2 | 登录 / 联系人 / 活动控制台 | P2 | UI |
| U3 | 拖拽 Email Builder | P2 | UI |
| U4 | 漏斗大屏 + 退订落地页 | P2 | UI |
| Q1 | 单测 + smoke + Playwright（¥0） | P4 | QA |
| D1 | 审计 + /learn | P5 | Auditor |

---

## 4. 镜像与平台校验记录

Phase 1 对以下镜像执行 `docker pull --platform linux/arm64` 与 `linux/amd64`：

| 镜像 | 用途 |
|---|---|
| `golang:1.23-alpine` | 后端构建 |
| `node:22-alpine` | 前端构建 |
| `nginx:1.27-alpine` | 前端运行 |
| `postgres:16-alpine` | 主库 |
| `redis:7.4-alpine` | 队列/限速 |
| `axllent/mailpit:v1.21.4` | 真实 SMTP + UI |

校验结果（2026-08-23 17:42 GMT+8）：上表 6 个镜像 `linux/arm64` 与 `linux/amd64` **全部 OK**。

---

## 5. Contract Gate 清单（Phase 3 强制）

1. 对 Mailpit 发 **一封真实 SMTP 邮件**，记录握手、状态码、错误格式 → `docs/.meta/api_contracts.md`
2. SES：无密钥则契约标 `UNVERIFIED`，响应结构照抄官方文档并注明来源，禁止臆造
3. 追踪网关对本机发出的测试信做一次像素命中 + 一次点击 302

---

## 6. 完成定义（DoD）

- `docker compose up --build -d` 后可在 localhost 打开控制台、API、追踪网关、Mailpit
- 种子账号能完成：导入 → 编辑模板 → 立即群发 → Mailpit 可见 → 打开/点击回流 → 漏斗更新
- Gmail 通道触顶时 Outlook 通道吞吐下降 < 5%（单测或压测证明）
- QA 每轮 Cost = ¥0
- Phase 5 PASS 后执行知识收割
