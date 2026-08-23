# DesignSpec — Lumen Relay

> 视觉规范。UI Agent 必须遵守。产品气质：**纸灯 / 驿站 / 夜间编辑室**，不是通用紫蓝 SaaS。

## 1. 气质

暖黑底、羊皮纸卡片、铜橙强调。像一间还亮着台灯的夜间编辑室：克制、可触摸、有纸张纤维。禁止默认 Inter + 靛蓝渐变。

## 2. 色彩

| Token | Hex | 用途 |
|---|---|---|
| ink | `#14110e` | 页面底 |
| ink-2 | `#1d1814` | 侧栏 / 顶栏 |
| paper | `#f4efe6` | 主文字 / 卡片 |
| paper-dim | `#c9bba8` | 次级文字 |
| brass | `#c45c26` | 主操作、漏斗强调 |
| brass-2 | `#e8a25a` | hover / 图表辅色 |
| moss | `#3d6b4f` | 成功 |
| rust | `#9b2c1a` | 危险 |
| line | `#2c261f` | 分割 |

## 3. 字体

- Display：`"Fraunces", "Songti SC", serif`（标题、漏斗数字）
- Body：`"IBM Plex Sans", "Source Han Sans SC", sans-serif`
- Mono：`"IBM Plex Mono"`（邮箱、token）

## 4. 组件

- 按钮：全圆角胶囊，主按钮 brass 实底，次按钮纸色描边。
- 输入：深底 + 细铜线 focus。
- 卡片：`#1d1814`，1px `#2c261f`，轻内阴影。
- Toast / Dialog：禁止原生 alert；右上角可关闭，5s 消失。
- 空态：插画式文案 + 单一 CTA，禁止空白灰框。

## 5. 布局

- 左侧 240px 导航（移动端抽屉）。
- 内容区 `w-full`，禁止 `max-w-*` 限宽（登录卡与 Dialog 例外）。
- 断点：768 / 480。

## 6. 漏斗大屏

深色全幅。五级漏斗：发送 → 投递 → 打开（真实/机器分列）→ 点击 → 退订/投诉。数字用 Fraunces。SSE 断线条幅。

## 7. 编辑器

左组件库、中画布（600px 纸色邮件）、右属性。拖拽幽灵半透明。占位符芯片一键插入 `{{ .UserName }}`。
