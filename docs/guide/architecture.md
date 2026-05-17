# CCX 架构说明

本文档描述 CCX 当前的系统边界、核心模块和主要请求流。

## 系统概览

CCX 是一个单端口部署的 AI API 代理与协议转换网关：

- Go 后端负责路由、认证、调度、协议转换、日志与指标
- Vue 3 + Vuetify 前端提供 Web 管理界面
- 前端构建产物通过 `embed.FS` 嵌入后端二进制
- 统一入口同时承载 Web UI、管理 API 与代理 API

## 仓库结构

```text
ccx/
├── backend-go/                 # Go 后端
│   ├── main.go                # 路由与服务启动
│   └── internal/
│       ├── config/            # 配置与热重载
│       ├── handlers/          # HTTP 处理器
│       ├── providers/         # 上游适配
│       ├── converters/        # Responses 转换器
│       ├── scheduler/         # 多渠道调度
│       ├── session/           # 会话与 Trace 亲和性
│       ├── metrics/           # 指标、日志、持久化
│       └── middleware/        # 认证、CORS、压缩、日志
├── frontend/                  # Vue 3 + Vuetify 管理界面
└── .config/                   # 运行时配置与持久化数据
```

## 渠道类型与路由面

CCX 当前内建五类渠道，每类渠道拥有独立的调度、指标和日志空间：

| 渠道类型 | 代理入口 | 管理入口 | 说明 |
| --- | --- | --- | --- |
| Messages | `/v1/messages` | `/api/messages/channels/*` | Claude Messages 语义 |
| Chat | `/v1/chat/completions` | `/api/chat/channels/*` | OpenAI Chat Completions |
| Responses | `/v1/responses`、`/v1/responses/compact` | `/api/responses/channels/*` | Codex/OpenAI Responses |
| Gemini | `/v1beta/models/*` | `/api/gemini/channels/*` | Gemini 原生协议 |
| Images | `/v1/images/generations`、`/v1/images/edits`、`/v1/images/variations` | `/api/images/channels/*` | OpenAI Images |

大多数代理入口都支持 `/:routePrefix/...` 变体，用于为渠道配置附加自定义前缀。

## 核心请求流

```text
Client
  -> Auth Middleware
  -> Route Handler
  -> Channel Scheduler
  -> Provider / Converter
  -> Upstream API
  -> Metrics / Channel Logs
  -> Client Response
```

分阶段说明：
1. 中间件完成认证、CORS、压缩和请求日志处理。
2. 各协议 handler 解析请求并选择对应渠道类型。
3. 调度器根据渠道状态、优先级、促销期、Trace 亲和性和熔断状态选择上游。
4. Provider 负责将请求转换成上游协议并处理流式/非流式响应。
5. 指标和渠道日志记录请求生命周期，再返回统一响应。

## 核心模块职责

### `internal/config/`
- 维护 `.config/config.json`
- 支持热重载与配置备份
- 提供各渠道配置读写能力

### `internal/handlers/`
- 承载 Messages、Chat、Responses、Gemini、Images 代理与管理接口
- 处理模型查询、能力测试、日志查询、状态切换等管理请求

### `internal/providers/`
- 封装上游 API 的请求构造与响应处理
- 屏蔽 Claude、OpenAI、Gemini、Responses 等上游差异

### `internal/converters/`
- 主要服务于 Responses 场景
- 负责 Responses 与其他协议之间的结构转换

### `internal/scheduler/`
- 多渠道选路核心
- 管理优先级、促销期、Trace 亲和性、故障转移与恢复

### `internal/session/`
- 为 Responses API 提供 `previous_response_id` 驱动的会话跟踪
- 维护 Trace 亲和性所需的会话级信息

### `internal/metrics/`
- 记录渠道指标、历史统计和请求日志
- 支持独立的熔断状态、持久化和自动恢复调度

## 调度与高可用

每类渠道有自己的 `MetricsManager` 与日志存储，避免不同协议互相污染健康状态。

调度时会综合考虑：
- 渠道配置状态（`active` / `suspended` / `disabled`）
- 促销期
- 优先级
- Trace 亲和性
- 熔断与可用 key 状态
- 模型过滤规则

实际选路顺序为：Promotion 渠道 > Trace 亲和 > 普通 priority 顺序。Trace 亲和会让位给更高优先级且健康的候选渠道，因此普通置顶 / reorder 会把低优先级亲和流量迁移到置顶渠道；Promotion 则是临时强制优先，会在首次选择时绕过健康检查尝试促销渠道。

失败场景下会执行故障转移，并结合熔断状态和定时恢复逻辑控制重试范围。

## 可观测性

CCX 提供三类核心可观测信息：

1. **渠道指标**
   - 请求量、成功率、失败率、延迟
   - 全局统计与按模型统计历史数据

2. **渠道日志**
   - 每个渠道保留最近请求日志
   - 记录 `status`、`statusCode`、`requestSource`、`interfaceType`、`baseUrl`、`keyMask` 等字段
   - Images 请求额外记录 `operation`

3. **运行时状态**
   - 熔断状态
   - 黑名单 key 恢复
   - Promotion / Resume 等管理动作

### Images 日志 `operation`

Images 请求会在渠道日志中记录具体端点类型：
- `generations`
- `edits`
- `variations`

该字段仅对 Images 渠道有语义，其余协议为空。

## 构建与版本

- 根目录 `VERSION` 是唯一版本源
- `backend-go/Makefile` 在构建时读取 `../VERSION`
- 版本、构建时间和 Git 提交通过 `-ldflags` 注入 `backend-go/version.go`
- 前端构建产物嵌入到 `backend-go/frontend/dist`

## 扩展点

### 添加新上游能力
1. 在 `internal/providers/` 中扩展或新增上游适配实现
2. 必要时在 `internal/converters/` 中补充协议转换
3. 为对应渠道类型补齐 handler、管理接口和前端配置入口
4. 将指标、日志和模型过滤纳入统一链路

### 调整调度策略
- 修改 `internal/scheduler/`
- 如涉及健康状态或恢复机制，同时检查 `internal/metrics/`

### 扩展管理界面
- 修改 `frontend/src/components/` 与 `frontend/src/services/api.ts`
- 如新增图标或 Vuetify 组件，需同步更新前端插件注册

## 相关文档

- 入口说明：`README.md`
- 后端专项：`backend-go/README.md`
- 开发流程：`DEVELOPMENT.md`
- 环境变量：`ENVIRONMENT.md`
- 发布流程：`RELEASE.md`
