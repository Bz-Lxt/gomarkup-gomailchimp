# Requirements.md — GoMailchimp

> **企业级多租户智能邮件营销（EDM）与高并发群发履约系统**
>
> | 项 | 值 |
> |---|---|
> | 文档版本 | v1.0（需求冻结版） |
> | 冻结时间 | 2026-08-23 17:30 (GMT+8) |
> | 负责 Agent | PM Agent / Alkaid-SOP v13.0 |
> | 权威性 | 本文件定义 **WHAT**；`docs/Roadmap.md` 定义 **WHEN**。二者冲突时以本文件为准 |
> | 原始 Prompt | `docs/.meta/original_prompt.md`（SSOT，禁止改写） |

---

## 1. PM 裁决摘要

**结论：✅ ACCEPT（接受立项）**

| 判据 | 结果 | 说明 |
|---|---|---|
| 完整性 | PASS | 需求描述具体到算法层，无缺失附件 |
| 平台独占 | PASS | Go + React，跨平台 |
| 规模 | **Tier 2 (10k–40k LoC)** | 估算 14k–17k LoC → 接受，**强制分期 Roadmap** |
| 外部依赖 | **Scenario A（可模拟）** | SMTP 发信可模拟；本项目更进一步使用真实 SMTP 服务端 Mailpit |
| 商业软件 | PASS | 全开源栈 |

### 1.1 规模估算依据

| 模块 | 预估 LoC | 文件数 |
|---|---:|---:|
| Go 后端（含 30+ 文件硬性要求） | 6,500 – 8,000 | 38 – 45 |
| Go 单元测试 | 1,200 – 1,600 | 12 – 16 |
| React 前端（拖拽编辑器为大头） | 5,000 – 6,500 | 55 – 70 |
| E2E 测试 / 脚本 / 配置 | 600 – 900 | 10 – 15 |
| **合计** | **≈ 13,300 – 17,000** | **≈ 115 – 146** |

落入 v13 Tier 2，因此 **Phase 1 必须先产出带 MVP / V1 / V2 明确边界的 `docs/Roadmap.md`，未获批准不得写业务代码**。

---

## 2. 业务目标

为独立站与 SaaS 团队提供一套 **可自托管** 的邮件营销履约平台，在**不损害发件域名信誉度**的前提下完成大批量触达，并闭环回收送达 / 打开 / 点击 / 退订 / 投诉五级转化数据。

系统的核心价值不是"把邮件发出去"，而是 **"在服务商的限速与信誉度约束下，尽可能快地、可观测地把邮件发出去"**。因此本项目的验收重心在于：限速器是否真实生效、退信是否真实隔离、追踪数据是否可归因。

---

## 3. 硬性交付门槛（Redline，违反即 STOP）

### 3.1 Docker 交付标准

- 全系统必须可通过 **`docker compose up --build -d` 一条命令**启动，禁止任何手工前置步骤（禁止手工建库、手工执行 SQL、手工装依赖）。
- 数据库 Schema 必须由**自动迁移**在容器启动时完成。
- 镜像必须同时支持 **linux/arm64（Apple Silicon）与 linux/amd64**，Phase 1 需对每个基础镜像执行 `docker pull --platform` 双架构验证并记录。
- 必须暴露以下 `localhost` 可访问入口（Dev 阶段随机端口，`/deploy` 阶段标准化到 8081+）：

| 入口 | 用途 |
|---|---|
| Web 控制台 | 前端 SPA（编辑器 / 活动台 / 漏斗大屏） |
| API 服务 | REST + SSE |
| 追踪网关 | 像素 `/o/...` 与重定向 `/c/...`，**独立于 API 端口** |
| Mailpit UI | 真实查看发出的邮件，作为交付可信度证据 |

- 所有容器必须设置 `TZ=Asia/Shanghai`；应用层时间统一使用 GMT+8，禁止裸用 UTC 落库。

### 3.2 美学标准（Dribbble Standard）

- 必须使用现代设计系统（**TailwindCSS + shadcn/ui**），禁止裸 HTML/CSS、禁止"工程师 UI"。
- 必须响应式，必须有完整反馈态：loading / empty / error / success。
- 拖拽编辑器与漏斗大屏是**门面模块**，视觉完成度直接决定验收，不允许功能可用但界面粗糙。

### 3.3 文档先行

- 无 `docs/Roadmap.md` 不得写业务代码。
- 交付必须含 `docs/API.md`：每个端点的请求/响应示例、参数类型、错误码表（`global.md` 强制项）。

### 3.4 Mock 合法性

本项目任何 Mock 必须同时满足：**(a)** 真实实现路径存在且已接线；**(b)** Mock/Real 开关在 `README.md` §7 显式文档化。详见 §9。

---

## 4. 技术栈冻结

| 层 | 选型 | 冻结理由 |
|---|---|---|
| 后端语言 | **Go 1.22+** | 用户指定 |
| HTTP 框架 | **Gin** | 中间件生态成熟，追踪网关需要极轻量路由 |
| 关系库 | **PostgreSQL 16** | 多租户 RLS 能力、JSONB 存模板 AST、时序聚合 |
| 缓存/限速/队列 | **Redis 7** | 分布式令牌桶需原子性（Lua），队列需可靠性 |
| 队列实现 | **自研 worker pool + Redis List/ZSet** | 用户明确要求"手写 worker pool"，不使用 asynq 等黑盒 |
| SMTP 收信端（Dev/Test） | **Mailpit** | 真实 SMTP 协议服务端 + Web UI，非 Mock |
| ORM / 数据访问 | **sqlc**（生成类型安全代码）或 **GORM** | Phase 1 由 Architect 二选一并写入 Roadmap |
| 迁移 | **golang-migrate** | 容器启动自动执行 |
| 前端框架 | **React 18 + TypeScript + Vite** | 生态最适配拖拽编辑器 |
| UI | **TailwindCSS + shadcn/ui** | 满足 §3.2 |
| 拖拽 | **dnd-kit** | 现代、可访问性好，优于 react-dnd |
| 图表 | **ECharts (echarts-for-react)** | 用户指定 |
| 实时推送 | **SSE** | 漏斗大屏单向推送，比 WebSocket 更简单可靠 |
| 表格导入 | **SheetJS (前端预览) + excelize (后端解析)** | Excel/CSV 双格式 |

**冻结约束**：Phase 2/3 Agent 不得擅自更换以上选型；如需变更必须回到 PM 重新冻结。

---

## 5. 功能需求（FR）

标注说明：`[MVP]` = 最小可用闭环，`[V1]` = 完整交付目标，`[V2]` = 增强项。**验收基线针对 V1**。

### 5.1 多租户与权限

| ID | 需求 | 阶段 |
|---|---|---|
| FR-T1 | 租户（Tenant）隔离：所有业务表带 `tenant_id`，数据访问层强制注入租户过滤，禁止裸查询 | MVP |
| FR-T2 | 用户认证：JWT（Access + Refresh），密码 bcrypt/argon2 存储 | MVP |
| FR-T3 | 角色：Owner / Marketer / Viewer 三级 RBAC，Viewer 只读 | V1 |
| FR-T4 | 租户级发送配额与预算上限（日/月封数上限，触顶自动暂停活动） | V1 |
| FR-T5 | 越权防护：任何跨租户 ID 访问返回 404 而非 403（避免 ID 枚举） | MVP |

### 5.2 邮件可视化拖拽编辑器（Email Builder）

| ID | 需求 | 阶段 |
|---|---|---|
| FR-B1 | 组件库拖拽：**文本、图片、按钮、分割线**（用户明确列举，四者必须全部实现） | MVP |
| FR-B2 | 扩展组件：容器/分栏、间距、社交图标、页脚（含退订链接） | V1 |
| FR-B3 | 画布交互：拖入、拖动排序、选中、删除、复制、撤销/重做 | V1 |
| FR-B4 | 属性面板：字体、字号、颜色、对齐、内外边距、链接 URL、图片 alt | MVP |
| FR-B5 | **模板以 JSON AST 持久化**（非直接存 HTML），HTML 由后端渲染器统一产出 | MVP |
| FR-B6 | 动态占位符：支持 `{{ .UserName }}` 等 Go template 语法；编辑器提供可插入变量下拉，变量取自联系人字段 + 自定义属性 | MVP |
| FR-B7 | 占位符安全：渲染必须 HTML 转义，防止联系人字段注入脚本；未知变量渲染为空串并记录告警，不得 panic | MVP |
| FR-B8 | 邮件 HTML 兼容性：渲染产物使用 **table 布局 + inline CSS**（Outlook 兼容），CSS 内联由后端完成 | V1 |
| FR-B9 | 实时预览：桌面/移动双视图切换；测试发送（Send Test）到指定邮箱 | V1 |
| FR-B10 | 模板 CRUD + 版本快照（活动引用的是快照，模板后续修改不影响已发活动） | V1 |

### 5.3 联系人与客群

| ID | 需求 | 阶段 |
|---|---|---|
| FR-C1 | Excel/CSV 导入：字段映射向导（把表头映射到 email/name/自定义字段） | MVP |
| FR-C2 | 导入健壮性：**逐行校验**邮箱格式、必填、类型、长度边界；坏行进入错误报告可下载，不阻断整批（`global.md` [Robustness]） | MVP |
| FR-C3 | 导入去重：同租户内 email 唯一，重复行按"更新/跳过"策略处理 | MVP |
| FR-C4 | 大文件导入异步化：>5k 行走后台任务 + 进度条 | V1 |
| FR-C5 | 客群（List/Segment）管理，活动按客群选取目标 | MVP |
| FR-C6 | 订阅状态机：`subscribed / unsubscribed / bounced / complained / suppressed` | MVP |

### 5.4 营销活动（Campaign）控制台

| ID | 需求 | 阶段 |
|---|---|---|
| FR-M1 | 活动 CRUD：名称、发件人（From/Reply-To）、主题、模板快照、目标客群、发送通道策略 | MVP |
| FR-M2 | 发送策略三选一：**立即发送 / 定时发送 / 分批渐进式发送（Throttled Ramp-up）** | MVP |
| FR-M3 | 分批渐进式：可配置批次大小、批次间隔、每小时递增比例（IP 预热曲线），UI 需可视化展示预计完成时间 | V1 |
| FR-M4 | 活动状态机：`draft → scheduled → running → paused → completed / failed / cancelled`，非法转移必须拒绝 | MVP |
| FR-M5 | 运行时控制：暂停 / 恢复 / 取消（取消需保证已入队任务不再发送） | V1 |
| FR-M6 | 发送前预检（Preflight）：模板变量完备性、退订链接存在性、黑名单命中预估、配额是否足够 | V1 |

### 5.5 多通道高并发群发管道（SMTP Pipeline）★核心

| ID | 需求 | 阶段 |
|---|---|---|
| FR-P1 | **自研 Worker Pool**：可配置 worker 数（默认支持数百至上千 goroutine），任务从队列异步消费。禁止 `for range + 同步发送` 的朴素实现 | MVP |
| FR-P2 | 优雅关闭：收到 SIGTERM 后停止取新任务、等待在途任务完成（有超时）、未完成任务归还队列，**不得丢件** | MVP |
| FR-P3 | Provider 抽象接口：`SMTPProvider` / `SESProvider` / `MockProvider` 实现同一 `Sender` 接口 | MVP |
| FR-P4 | **动态负载均衡**：多 SMTP 账号/通道间按权重 + 实时健康度 + 剩余令牌分配任务 | V1 |
| FR-P5 | 熔断与健康度：通道连续失败达阈值自动熔断并半开探活，熔断期间任务改投其他通道 | V1 |
| FR-P6 | 连接复用：SMTP 连接池，避免每封邮件重建 TCP/TLS | V1 |
| FR-P7 | **窄化重试（Narrow Retry）**：仅对瞬时错误（SMTP 4xx、网络超时）指数退避重试（上限 3 次）；**认证失败（535）、地址无效（550）、格式错误一律不重试**，直接终态 | MVP |
| FR-P8 | 幂等性：任务携带唯一 `message_id`，重复消费不产生重复投递 | V1 |
| FR-P9 | 任务持久化：进程重启后 running 活动可恢复续发（v13 异步可靠性要求） | V1 |
| FR-P10 | 死信队列（DLQ）：超过重试上限的任务落 DLQ，控制台可查看与手动重投 | V1 |

### 5.6 精准分布式限速器（Rate Limiter）★核心

| ID | 需求 | 阶段 |
|---|---|---|
| FR-R1 | **手写令牌桶（Token Bucket）**，禁止直接套用 `golang.org/x/time/rate` 作为最终答案（其为单进程，不满足"分布式"要求）；可作为单机对照实现 | MVP |
| FR-R2 | **分布式语义**：桶状态存 Redis，取令牌操作用 **Lua 脚本保证原子性**，多实例下总速率不超限 | MVP |
| FR-R3 | **多维度限速**：至少支持三维——① 收件方域名（gmail.com / outlook.com / 其他）② 发送通道（SMTP 账号 / SES）③ 租户。三者取最严约束 | MVP |
| FR-R4 | 限速配置：每维度可配 `每小时上限 / 每分钟上限 / 桶容量（突发）`，运行时热更新无需重启 | V1 |
| FR-R5 | **通道级隔离（用户明确场景）**：Gmail 通道触顶时该通道任务挂起/降速，**Outlook 等其他通道必须保持全速**，二者不得互相阻塞。此点为**验收硬指标**，需有测试证明 | MVP |
| FR-R6 | 挂起而非丢弃：触顶任务重新入延迟队列（Redis ZSet by score=可执行时间），到点唤醒 | MVP |
| FR-R7 | 限速可观测：控制台展示各维度当前令牌余量、被限速次数、预计恢复时间 | V1 |

### 5.7 追踪像素与反向重定向网关 ★核心

| ID | 需求 | 阶段 |
|---|---|---|
| FR-K1 | **打开追踪**：渲染时在 HTML `</body>` 前注入 1x1 透明 GIF/PNG，路由形如 `GET /o/{token}.gif`。网关**先返回图片字节再异步落库**，不得让统计写入阻塞响应 | MVP |
| FR-K2 | **点击追踪**：渲染时重写所有外链为 `GET /c/{token}`，命中后记录点击再 **302** 跳原 URL | MVP |
| FR-K3 | **Token 安全**：token 必须签名（HMAC）且携带 `tenant/campaign/recipient/link` 信息，**禁止裸自增 ID**，防止遍历刷量与跨租户污染 | MVP |
| FR-K4 | **开放重定向防护**：跳转目标必须来自签名 token 或租户白名单域，禁止接受 URL 查询参数直跳 | MVP |
| FR-K5 | **去重与机器打开过滤**：同 recipient 首次打开记 unique open，后续记 total open；识别并标记 Gmail Image Proxy / Apple MPP 等预取（UA/IP 段/极短时延特征），大屏上"机器打开"独立呈现，不污染真实打开率 | V1 |
| FR-K6 | 网关高性能：与主 API 分离部署（独立端口/独立容器可选），事件写入走缓冲批量落库 | V1 |
| FR-K7 | 网关容错：任何统计异常都不得影响图片返回或 302 跳转（追踪失败 ≠ 用户体验失败） | MVP |

### 5.8 退信（Bounce）、退订与黑名单 ★核心

| ID | 需求 | 阶段 |
|---|---|---|
| FR-D1 | **同步退信解析（真实路径）**：SMTP 会话返回的 5xx/4xx 状态码 + Enhanced Status Code (RFC 3463) 实时解析，分类为 `hard / soft / block` | MVP |
| FR-D2 | **异步退信摄取**：定义 `BounceSource` 接口，实现 ① IMAP 退信箱轮询 + DSN(RFC 3464) 解析 ② ESP Webhook（SES SNS 格式）③ Mock Feeder。三者产出统一 `BounceEvent` | V1 |
| FR-D3 | **状态机自动隔离**：hard bounce 立即 → `suppressed`（死信）；soft bounce 累计达阈值（如 3 次/7 天）→ `suppressed`；投诉（FBL）立即 → `complained` + 永久隔离 | MVP |
| FR-D4 | **黑名单隔离区**：全局 + 租户级 suppression list。**发送前必查**，命中直接跳过并计入 `skipped`，绝不投递 | MVP |
| FR-D5 | 隔离可解释：每条黑名单记录原因、来源、时间、原始 SMTP 响应，控制台可查、可人工解除（需二次确认） | V1 |
| FR-D6 | **一键退订（合规必选）**：邮件必须含 `List-Unsubscribe` 与 `List-Unsubscribe-Post` 头（**RFC 8058**），以及正文可见退订链接；退订请求必须在 **48 小时内**生效（本系统要求即时生效） | MVP |
| FR-D7 | 退订落地页：确认页 + 退订原因收集（可选），退订后写入状态机 | V1 |
| FR-D8 | 域名信誉度看板：实时展示硬退信率、投诉率、退订率，超过行业红线时**自动暂停活动并告警** | V1 |

### 5.9 实时漏斗大屏

| ID | 需求 | 阶段 |
|---|---|---|
| FR-F1 | **ECharts 漏斗图**，层级严格按用户指定：**发送总数 → 投递成功 → 邮件打开 → 链接点击 → 退订/投诉** | MVP |
| FR-F2 | 实时性：SSE 推送，数据延迟 ≤ 3s；断线自动重连 | MVP |
| FR-F3 | 辅助图表：发送速率时序曲线（含限速触顶标记）、通道健康度、Top 点击链接排行、收件域名分布 | V1 |
| FR-F4 | 大屏视觉：深色主题、动效过渡、关键指标卡片，达到 §3.2 标准 | V1 |
| FR-F5 | 指标口径必须在 UI 内可查（tooltip 说明 unique/total 区别、机器打开如何计算） | V1 |

### 5.10 平台工程要求

| ID | 需求 | 阶段 |
|---|---|---|
| FR-G1 | **统一 Logger**（结构化 JSON，`slog`），含 level 控制，生产自动屏蔽 debug。**禁止裸 `fmt.Println` / `log.Print` 散落**（`global.md` [Logging]） | MVP |
| FR-G2 | 请求链路 ID（trace_id）贯穿 HTTP → 队列 → 发送 → 追踪事件 | V1 |
| FR-G3 | `/healthz` `/readyz` 健康检查，供 compose healthcheck 使用 | MVP |
| FR-G4 | Prometheus 指标端点：队列深度、发送速率、成功/失败/限速计数、通道延迟 | V1 |
| FR-G5 | 配置全部走环境变量（12-Factor），提供 `.env.example` | MVP |
| FR-G6 | `docs/API.md`：端点清单 + 请求/响应示例 + 参数类型 + **错误码表**（`global.md` [Documentation]） | V1 |

---

## 6. 非功能需求（NFR）

| ID | 类别 | 要求 |
|---|---|---|
| NFR-1 | 安全 | 密码哈希（argon2id/bcrypt）、JWT 过期与刷新、SQL 参数化（禁止拼接）、CORS 白名单、上传文件类型与大小校验、追踪 token HMAC 签名 |
| NFR-2 | 安全 | 秘钥全部环境变量注入，仓库内**零硬编码凭据**；`.env` 必须 gitignore |
| NFR-3 | 健壮性 | 所有外部输入（CSV/Excel/API Body/SMTP 响应/DSN 报文）必须结构化校验，不得仅靠调用方检查（`global.md` [Robustness]） |
| NFR-4 | 可靠性 | 无单点丢件：入队持久化 + 消费确认 + 重启恢复 |
| NFR-5 | 性能 | 见 §7 量化基线 |
| NFR-6 | 可测试性 | 核心引擎（限速器/Worker Pool/状态机/渲染器/Token 签名）必须可单测，依赖走接口注入 |
| NFR-7 | 测试覆盖 | 后端覆盖 CRUD + 核心引擎单测；前端 Playwright E2E（`global.md` [Testing]） |
| NFR-8 | 时区 | 全链路 GMT+8，容器 `TZ=Asia/Shanghai`，落库时间统一口径 |
| NFR-9 | 成本 | **测试与 CI 全程 Mock/离线模式，预期外部花费 ¥0**（v13 成本安全测试） |

---

## 7. 可量化验收基线

> v13 要求：有行业基准的领域必须写成**可测量**指标，不得用形容词。
> 分为「系统性能基线（必须实测）」与「业务健康红线（系统需具备监控与自动熔断能力）」两类。

### 7.1 系统性能基线（Phase 4 QA 必须实测并记录数值）

| 指标 | 目标值 | 测量方式 |
|---|---|---|
| 群发吞吐（Mailpit 通道，单容器） | **≥ 500 封/分钟** | 1 万封任务压测，取稳态速率 |
| 单封端到端处理延迟 P95 | **< 200 ms** | 出队 → SMTP 返回 |
| 追踪网关响应 P95 | **< 50 ms** | 像素与 302 两条路径分别测 |
| 追踪网关吞吐 | **≥ 1000 QPS** | 单容器压测 |
| CSV 导入速度 | **≥ 5000 行/秒**（校验+入库） | 5 万行样本 |
| 限速精度误差 | **< ±2%** | 设 600/min，实测 10 分钟总量偏差 |
| **通道隔离正确性** | **Gmail 通道触顶时，Outlook 通道吞吐下降 < 5%** | FR-R5 专项测试，**硬性验收项** |
| 重启恢复 | **丢件数 = 0** | 发送中途 `docker restart`，校验总投递数 |
| 幂等性 | **重复消费 0 重复投递** | 人为重投队列消息 |
| 优雅关闭 | 在途任务 100% 完成或 100% 归还 | SIGTERM 测试 |
| API P95 延迟 | **< 300 ms**（非报表类） | 常规 CRUD |
| 后端单测覆盖率 | **≥ 60%**，核心引擎包 **≥ 80%** | `go test -cover` |
| 每轮 QA 外部花费 | **¥0** | QA_Record 记录 |

### 7.2 业务健康红线（系统必须监控 + 触线自动暂停）

参照 Gmail/Yahoo 2024 起生效的发件人要求与 EDM 行业基准：

| 指标 | 行业基准 | 系统红线（触发自动暂停 + 告警） |
|---|---|---|
| 送达率 | ≥ 95% | < 90% |
| 硬退信率 | < 2% | **> 5%** |
| 投诉率（Spam Complaint） | < 0.1% | **> 0.3%** |
| 退订率 | < 0.5% | > 2% |
| 打开率（参考） | 行业均值 ≈ 21% | 仅展示，不熔断 |
| 点击率（参考） | 行业均值 ≈ 2.6% | 仅展示，不熔断 |

**验收方式**：Mock Feeder 注入超红线的退信/投诉事件，验证系统在 **60 秒内**自动暂停活动并产生告警记录。

---

## 8. 矛盾与歧义裁决表（PM 冻结，下游 Agent 不得推翻）

| # | 冲突/歧义 | 裁决 |
|---|---|---|
| **C1** | Prompt 中追踪 URL 写作 `https://api.com{{.TaskID}}.png` 与 `https://api.com`，均缺失路径段/参数 | 判定为书写截断，非需求。**冻结路由为**：打开 `GET /o/{token}.gif`；点击 `GET /c/{token}`。`token` 为 HMAC 签名串（FR-K3） |
| **C2** | "不被拉黑" 与 "高速发送数万封" 是**相互对立的优化目标** | 不追求绝对速度。验收以 **§7.1 限速器精度与通道隔离正确性**为准，而非"发得多快"。渐进式 ramp-up（FR-M3）是二者的和解方案 |
| **C3** | 像素打开率天然失真：Gmail Image Proxy 预取、Apple MPP 默认预加载，会造成打开率虚高 | **不假装准确**。实现 FR-K5 机器打开识别，大屏上 `真实打开 / 机器打开 / 合计` 三列并存，tooltip 说明口径。**隐瞒此失真视为数据造假** |
| **C4** | "解析退信回执" 需要真实 MX 收信通道，Docker 单机 demo 无公网域名 | **双通道**：① 同步 SMTP 5xx/4xx 解析（**100% 真实**，Mailpit 可构造拒收）；② 异步 `BounceSource` 接口 + IMAP/Webhook 真实实现 + Mock Feeder。Mock 合法性见 §9 |
| **C5** | "分布式限速器" 暗示多实例，但 compose demo 默认单实例 | 限速器实现必须是分布式正确的（Redis Lua）。compose 提供 `--scale sender=3` 验证方式并写入 README §7，用多实例压测证明总速率不超限 |
| **C6** | Prompt 未提及 RFC 8058 一键退订，但这是 2024 年起 Gmail/Yahoo 对批量发件人的**强制要求**，缺失会直接导致"被拉黑"——与 Prompt 首要目标冲突 | **PM 补入为 MVP 必选项**（FR-D6）。理由：不实现它就无法达成 Prompt 明确写出的"不被拉黑"目标，属需求内在推论而非范围扩张 |
| **C7** | "企业级多租户" 但未定义租户边界与角色 | 冻结为 §5.1：`tenant_id` 硬隔离 + Owner/Marketer/Viewer 三角色 + 跨租户访问返回 404 |
| **C8** | "30+ Go 文件 / 5000–8000 行" 是**约束**还是**目标**？ | 视为**下限约束**。不得为凑行数造冗余代码；若合理实现低于下限，以功能完整性为准并在审计说明 |

---

## 9. 外部依赖与 Mock 策略（Redline 4 合规声明）

| 依赖 | 分类 | 真实实现路径 | 离线替代 | 开关 |
|---|---|---|---|---|
| SMTP 发送 | Scenario A | `net/smtp` + STARTTLS/SSL，可配任意真实 SMTP 账号 | **Mailpit（真实 SMTP 服务端）**，协议真实，仅收件方在容器内 | `MAIL_PROVIDER=smtp\|ses\|mock` |
| AWS SES | Scenario A | AWS SDK v2 `SendRawEmail` | `MockSESProvider`，返回符合 SES 响应结构的模拟结果 | 同上 |
| SES Bounce Webhook (SNS) | Scenario A | 真实 HTTP 端点 + SNS 签名校验 | Mock Feeder 按真实 SNS/DSN 报文结构注入 | `BOUNCE_SOURCE=imap\|webhook\|mock` |
| IMAP 退信箱 | Scenario A | `go-imap` 轮询 + RFC 3464 DSN 解析 | Mailpit 内构造退信邮件供解析 | 同上 |

**合法性论证**：
1. **真实路径存在且已接线** —— 每个 Provider 都是同一 `Sender`/`BounceSource` 接口的实现，切换仅改环境变量，无代码分叉。
2. **开关文档化** —— 必须在 `README.md` §7「API 模拟与切换指南」中逐项说明。
3. **Contract Gate（Phase 3 强制）** —— Mailpit SMTP 通道必须先发**一封真实邮件**验证握手、STARTTLS、状态码与错误格式，结果记入 `docs/.meta/api_contracts.md`。无 AWS 凭据时，SES 契约标记为 `UNVERIFIED` 并**禁止臆造响应结构**（必须照抄官方文档结构并注明来源）。

**违规定义**：若某功能只有 Mock 而无真实实现接口，或开关未文档化 → 判定为**伪造交付**，Phase 5 直接 FAIL。

---

## 10. 数据模型概览（Phase 1 细化）

核心实体：`tenants` · `users` · `contacts` · `contact_lists` · `list_memberships` · `templates` · `template_versions` · `campaigns` · `campaign_recipients`（任务表，承载状态机）· `send_channels` · `channel_stats` · `email_events`（open/click/bounce/unsub/complaint 统一事件流）· `suppressions`（黑名单）· `bounce_records` · `import_jobs` · `audit_logs`

关键设计约束：
- 所有业务表带 `tenant_id` 且建复合索引 `(tenant_id, ...)`。
- `email_events` 为**只追加**事件表，漏斗指标由其聚合（或物化视图）得出，**禁止在多处维护计数字段造成口径分裂**。
- `campaign_recipients` 是发送状态机的唯一真值：`pending → queued → sending → sent → delivered / bounced / failed / skipped`。
- 时间列统一 GMT+8 口径（NFR-8）。

---

## 11. 范围边界（Out of Scope）

明确**不做**，防止 Scope Drift：

- ❌ 自建 MTA / MX 收信服务器（不实现 SMTP 服务端）
- ❌ DKIM/SPF/DMARC 的 DNS 记录自动配置（仅提供检查清单文档与配置校验工具）
- ❌ A/B 测试、自动化旅程编排（Marketing Automation Journey）、AI 文案生成
- ❌ 短信/推送/WhatsApp 等非邮件渠道
- ❌ 计费与订阅付费
- ❌ 真实公网域名信誉度对接（Google Postmaster Tools API）
- ❌ 多语言 i18n（界面中文为主）

如需以上能力，另行立项。

---

## 12. 阶段划分建议（Roadmap 由 Phase 1 细化定稿）

> Tier 2 项目强制分期。以下为 PM 建议边界，Chief Architect 可细化但不得删减 MVP 项。

| 阶段 | 边界 | 交付判据 |
|---|---|---|
| **MVP** | 多租户+认证、联系人导入、模板 AST 与渲染（含占位符+四类组件）、活动 CRUD、Worker Pool、Redis 令牌桶（三维度+通道隔离）、追踪网关（像素+302+签名）、黑名单与同步退信、一键退订、基础漏斗大屏、Docker 一键起 | 能完整跑通「导入 → 编辑 → 群发 → Mailpit 收到 → 打开/点击回流 → 漏斗更新」闭环 |
| **V1** | 渐进式发送、通道负载均衡与熔断、连接池、DLQ、重启恢复、异步退信摄取、机器打开识别、信誉度看板与自动熔断、完整大屏、RBAC、API.md、E2E 测试 | 达成 §7 全部量化基线 |
| **V2** | Prometheus/Grafana、模板市场、更多编辑器组件、多实例水平扩展演示、审计日志 UI | 加分项，不影响验收 |

### 12.1 Phase Order 建议：**Logic-First（建议 Phase 2 与 Phase 3 互换）**

**理由**：邮件拖拽编辑器是典型的 **editor/canvas 型 UI**，其组件树、属性面板、撤销栈**全部派生自模板 AST 数据模型**；同理漏斗大屏派生自 `email_events` 事件模型。若先做 UI 再定 schema，编辑器与渲染器必然返工。

**依据 SOP v13 §PHASE 1 规则**：「IF the UI is an editor/timeline/canvas/visualization whose component structure is derived from the data model → SWAP Phase 2 and Phase 3」。

Chief Architect 必须在 `docs/Roadmap.md` 中确认或推翻此建议，并给出一句话理由（未记录则默认 UI-First）。

---

## 13. 风险登记

| # | 风险 | 影响 | 缓解 |
|---|---|---|---|
| R1 | 拖拽编辑器工作量被低估（历史上是此类项目最大黑洞） | 进度 | MVP 严格限定四类组件；扩展组件推 V1 |
| R2 | 邮件 HTML 跨客户端兼容（Outlook 用 Word 渲染引擎） | 质量 | 强制 table 布局 + inline CSS，模板经 Mailpit 源码核验 |
| R3 | 分布式令牌桶 Lua 脚本时钟漂移 | 正确性 | 统一以 Redis `TIME` 命令为时钟源，禁用应用侧时间 |
| R4 | 压测 500 封/分钟在 Mailpit 上可能受其自身写盘限制 | 验收 | 提供 `MAIL_PROVIDER=mock`（内存 sink）做纯管道压测，Mailpit 做协议真实性验证，两者分别记录 |
| R5 | 打开率口径争议（C3） | 审计 | 已冻结三列并存口径 + tooltip 说明 |
| R6 | 多租户越权漏洞 | 安全 | 数据访问层统一强制注入 tenant 过滤 + 专项越权测试用例 |
| R7 | 30+ 文件的目录组织混乱 | 可维护性 | Phase 1 冻结目录结构（`internal/` 分层：domain / service / repo / provider / pipeline / api） |

---

## 14. 需求冻结声明

本文件经 PM Agent 评估后冻结。下游 Agent（Architect / UI / Logic / QA / Auditor）**均以本文件 + `docs/.meta/original_prompt.md` 为唯一需求真值**。

任何范围变更必须回到 `/pm` 重新冻结，禁止在实现阶段自行增删功能。
