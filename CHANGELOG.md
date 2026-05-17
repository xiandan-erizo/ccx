## [v2.7.0] - 2026-05-17

### 新增

- **Responses Compact 本地压缩** - 当 responses 渠道上游为非原生 Responses 类型（openai/claude/gemini）时，`/v1/responses/compact` 端点自动切换为本地 compact 模式：将对话历史格式化为 transcript，通过现有 converter 管线发送普通请求让模型生成摘要，再包装为 Responses 格式返回。支持流式/非流式跟随客户端、session 历史读取与 compact 结果写回、大输入截断保护。原生 responses 上游若返回 404/405/501 也会自动回退本地 compact
- **SessionManager 新增只读查询与压缩会话创建** - 新增 `GetSessionByResponseID` 通过 responseID 只读查找 session；新增 `CreateCompactedSession` 创建压缩后的轻量会话并记录映射
- **ResponsesProvider 提取公共请求构建方法** - 新增 `ConvertBodyToProviderRequest` 公共入口，接受 bodyBytes 参数复用现有 URL/转换/认证逻辑，供 compact 等场景调用

### 改进

- **Messages 渠道模型列表兼容国内 Claude 协议入口** - 当 `base_url` 以 `/anthropic`、`/claude`、`/messages` 结尾时（如 `https://api.deepseek.com/anthropic`），模型列表获取自动尝试三段候选 URL：当前路径 → 剔除协议尾段 → 纯域名根路径，解决国内服务商模型接口不在兼容协议子路径下的问题。管理端"获取模型"同步适配。

## [v2.6.99] - 2026-05-16

### 修复

- **Windows 兼容性修复** - 将 `syscall.Kill` 替换为 `os.FindProcess`，修复 Windows 平台上的编译兼容性问题

## [v2.6.98] - 2026-05-16

### 修复

- **对话持久化字段补全** - `persistedConversation` 补齐 `RequestCount`、`Models`、`CurrentChannel`、`ChannelName`、`LastModel`、`LastRequestID` 字段，修复服务重启后对话卡片显示 "Channel 0" 和请求次数归零的问题
- **NEXT 渠道 chip 可读性优化** - 为 ConversationCard 的 NEXT 渠道 chip 添加专用高对比度样式，避免橙色文字在浅色背景上不醒目

### 新增

- **会话标题持久化与实时补全** - 对话追踪器新增 `.config/conversation_state.json` 本地持久化，保存 `title`、AI 生成标题与最新用户消息兜底摘要，服务重启后驾驶舱卡片仍可保留标题和创建时间；用户每轮输入后实时用最新消息摘要补全卡片标题。

### 改进

- **Failover 不可重试错误日志输出上游响应体** - 当 failover 判定为不可重试错误（内容审核、参数校验等）时，日志中增加上游返回的 body 内容（截断至 4KB），便于快速定位 400 错误的具体原因（Closes #65）
- **会话调度看板（Conversation Dashboard）** - 新增前端"Sessions"Tab 和后端 API，管理员可实时观察当前所有经过网关的活跃对话（按 kind:userID 聚合），并为单个对话自定义渠道优先级序列（含 failover 顺序）。支持拖拽排序、点击置顶、降级操作，覆盖规则 30 分钟 TTL 自动过期。
- **对话追踪器（ConversationTracker）** - 后端新模块 `internal/conversation/`，自动追踪所有成功请求的对话元数据（模型、渠道、请求次数、状态），1 小时无活动标记 idle，2 小时后自动清理
- **渠道序列覆盖（OverrideManager）** - 支持为单个对话设置完整的渠道调度序列，调度优先级：X-Channel > 促销期 > 手动覆盖 > Trace 亲和 > 默认排序

## [v2.6.97] - 2026-05-15

### 修复

- **Chat 渠道 Gemini 端点注入 thought_signature** - 当 Chat 渠道 serviceType 为 openai 但实际对接 Gemini 上游时，自动为 function calling 的 tool_calls 注入 thought_signature，避免 Gemini 3 模型返回 400 错误

## [v2.6.96] - 2026-05-15

### 修复

- **前端暗色模式对话框按钮文字对比度优化** - 改善 dark mode 下对话框按钮文字的可读性
- **BuildChannelView 补充 vision 相关字段返回** - 修复 noVision/noVisionModels/visionFallbackModel 字段在构建渠道视图时未正确返回的问题

## [v2.6.95] - 2026-05-15

### 新增

- **Vision 能力路由** - 支持根据请求是否包含图片内容动态跳过不支持 vision 的渠道/模型；渠道级 `noVision` 标记整渠道不支持图片输入，`noVisionModels` 标记特定模型不支持，`visionFallbackModel` 支持同渠道内模型降级；前端 AddChannelModal 模型映射行增加眼睛图标 toggle 和可选 fallback 输入，ChannelCard 显示 noVision 指示器

## [v2.6.94] - 2026-05-14

### 修复

- **修复 Responses→OpenAI Chat 历史中孤立 `tool` 消息导致 DeepSeek 拒绝请求** - 当历史消息中存在没有前置匹配 `tool_calls` 的 `function_call_output` / `tool_result` 时，转换层会将该孤立 tool 输出降级为普通 user 文本保留上下文，避免上游返回 `Messages with role 'tool' must be a response to a preceding message with 'tool_calls'`；同时补充 converter 与 DeepSeek thinking matrix 测试，兼容 OpenAI Chat content parts 断言。

## [v2.6.93] - 2026-05-14

### 修复

- **修复 Responses 渠道 converter 路径未注入 thinking/reasoning 参数** - 当 Responses 渠道 serviceType 为 chat/openai/claude 时（走 converter 转换路径），`reasoningParamStyle` 配置不生效，转发到上游的请求体中缺少 thinking 或 reasoning 参数；现在 converter 路径也会根据配置正确注入，同时支持无 ReasoningMapping 时透传客户端原始 reasoning 并按 style 转换格式（Closes #59）

## [v2.6.92] - 2026-05-14

### 修复

- **修复 MiniMax 等上游流式输出缺少 `[DONE]` 终止符导致 `stream disconnected before completion`** - 当 OpenAI 兼容上游（如 MiniMax）在流式响应结束时不发送 `data: [DONE]`，Responses handler 现在会自动检测并补发 `response.completed` 事件，确保客户端正常接收完整流（Closes #39）

### 新增

- **新增 `thinking` 思考参数风格** - 渠道 `reasoningParamStyle` 新增 `thinking` 选项，配置后将 reasoning effort 转换为 `{"thinking": {"type": "enabled"}}` 格式，支持京东 CodingPlan / GLM-5 等需要该格式开启思考模式的上游（Closes #54）

### 修复

- **修复 Chat 渠道 Gemini 上游 function calling 缺少 thought_signature 导致 400 错误** - Gemini 3 模型要求多轮 function calling 时 assistant message 的 tool_calls 必须包含 `thought_signature`，Chat handler 现在会自动为缺失该字段的 tool_calls 注入 dummy 值（`skip_thought_signature_validator`）跳过验证，已有真实 signature 则保留原值
- **修复编辑渠道时思考参数风格（reasoningParamStyle）未正确保存** - Messages、Responses、Gemini、Images 四类渠道的更新函数缺少对 `reasoningParamStyle` 字段的赋值，导致前端编辑后该配置被静默丢弃；仅 Chat 渠道此前已正确处理

## [v2.6.91] - 2026-05-14

### 修复

- **修复 Codex 字符串 apply_patch 代理调用回写** - 当 Codex 客户端以字符串简写形式声明 `apply_patch` 工具时，响应回写阶段会将上游 function_call 重新映射为 Codex 可执行的原生工具调用，避免客户端无法识别代理函数。

## [v2.6.90] - 2026-05-14

### 新增

- **新增 `codexNativeToolPassthrough` 渠道开关** - 透传分支中将 Codex 原生工具（apply_patch、namespace、web_search、local_shell、computer_use）转换为 OpenAI function 格式，使上游模型可调用；与 `codexToolCompat`（剥离工具）互斥，优先级更高；修复 Issue #52 中 Codex Desktop 原生工具无法被 deepseek 等上游调用的问题
- **导出 `ConvertRawToolsToOpenAI` 函数** - converters 包新增导出包装，供透传分支复用已有的工具转换逻辑
- **前端渠道编辑新增开关** - AddChannelModal 中在 Codex 工具兼容开关前新增 "Codex 原生工具透传" 开关，含英文/印尼语/中文三语翻译

### 修复

- **修复 Codex apply_patch proxy 调用回写不可执行** - 当 Codex 以字符串数组形式暴露 `apply_patch_add_file` / `apply_patch_batch` 等 proxy 工具时，ccx 现在会在响应回写阶段将上游 function_call 反向映射为 Codex 客户端可执行的 `custom_tool_call name=apply_patch`，避免客户端报 `unsupported call`。

## [v2.6.89] - 2026-05-13

### 修复

- **修复 /v1/models 聚合接口对 Gemini 渠道的鉴权和解析** - 修复 Models 聚合接口在请求 Gemini 渠道时鉴权和响应解析异常的问题
- **修复流式空 thinking 内容触发重试** - 当流式响应中 thinking 内容为空时正确触发重试机制，避免下游收到无效响应
- **对齐前端 Reasoning Content 回传开关** - 修复前端 Reasoning Content 回传开关状态与后端不一致的问题

### 其他

- **移除本地测试脚本** - 清理仓库中遗留的本地测试脚本文件

## [v2.6.88] - 2026-05-13

### 修复

- **Fuzzy 模式下非流式空响应自动 failover** - 此前流式路径已通过 `PreflightStreamEvents` + `ErrEmptyStreamResponse` 拦截上游 200 + 内容为空的响应，但非流式路径仅在 JSON 解析失败时才走 failover；当上游返回 `HTTP 200` + 合法 JSON + 语义空内容（如 `choices[0].message.content=""`、Claude `content=[]`、Responses `output=[]`、Gemini `candidates=[]` 等）时直接透传给客户端，导致 Claude Code / Codex 等下游立即中断。新增 `common/empty_response.go` 定义 `ErrEmptyNonStreamResponse` 与 `IsClaudeResponseEmpty` / `IsChatResponseEmpty` / `IsResponsesResponseEmpty` / `IsGeminiResponseEmpty` 四个协议级判空函数；`messages` / `chat` / `responses` / `gemini` 四个非流式 `handleSuccess` 在响应转换成功后、Header 写出前先判空，仅 Fuzzy 模式下返回 `ErrEmptyNonStreamResponse`，由 `TryUpstreamWithAllKeys` 现有 failover 分支自动切换 Key/BaseURL/渠道；判空保留 tool_use / server_tool_use / redacted_thinking / function_call / reasoning / refusal / Gemini SAFETY/RECITATION/promptFeedback.blockReason 等语义非空场景，避免误触发 failover；chat handler Claude 分支直接在原生 `*types.ClaudeResponse` 上判空（而非转换后的 Chat map），避免 `convertClaudeResponseToChat` 忽略 `server_tool_use` / `redacted_thinking` 导致的误判；`isResponsesItemEmpty` 在 `[]interface{}` content / summary 分支补齐非文本 part 类型检测（与 `isChatMessageEmpty` 保持一致）；严格模式行为不变；新增 `empty_response_test.go` 共 36+ 个表驱动用例覆盖各协议判空规则
- **修复 Codex 兼容 apply_patch_batch 工具 schema 缺失 items 字段导致 400** - `codex_tools.go` 中 `applyPatchBatchSchema` 的 `operations.items.properties.hunks` 声明为 `"type":"array"` 但未声明 `items` 子 schema，OpenAI 官方等严格校验的上游会返回 `invalid_function_parameters 400: Invalid schema for function 'apply_patch_batch': ... array schema missing items.`；抽取公共 `applyPatchHunksSchema` 供 `update_file` 单文件代理与 `batch` 代理共用，保证两处 schema 结构一致并补齐嵌套 `lines` 数组的 items 定义；新增 `TestApplyPatchSchemaHunksHaveItems` 锁定两处 schema 的 `hunks` / `hunks.items.lines` 均声明了 `items`，防止回归

## [v2.6.87] - 2026-05-13

### 新增

- **Claude 协议上游 thinking ↔ reasoning_content 双向转换** - 针对 mimo（xiaomimimo）等使用 Claude 协议但内部要求 OpenAI 风格 `reasoning_content` 回传的上游，`UpstreamConfig` 新增 `passbackReasoningContent` 开关；开启后 `ClaudeProvider.ConvertToProviderRequest` 将 assistant 消息中的 `thinking` 内容块提取为消息级 `reasoning_content` 字段，`ConvertToClaudeResponse` 与 `HandleStreamResponse` 将上游响应中的 `reasoning_content`（含 SSE `thinking_delta`）转回 Claude `thinking` 块，修复 mimo-v2.5-pro 因历史消息缺少 `reasoning_content` 返回 `400 "The reasoning_content in the thinking mode must be passed back to the API."` 的问题；新增 `claude_reasoning_test.go` 与 `mimo_reasoning_test.go` 覆盖非流式/流式及开关关闭时的透传行为
- **前端新增「回传 Reasoning Content」开关** - Messages 渠道 + `serviceType=claude` 时在 `AddChannelModal` 的 Gemini Thought Signature 配置之后显示新开关；同步更新 `Channel`/`ChannelFormLike` 类型定义、`buildChannelPayload` 序列化、`App.vue` 克隆逻辑及 zh-CN/en/id 三语言文案；`channel_metrics_handler.go` 在 `messages` 渠道列表响应中回传 `passbackReasoningContent`，避免编辑时开关状态丢失
- **`ccx --version` 命令支持** - `backend-go/main.go` 在 `main()` 入口最前面增加参数短路分支，识别到 `--version` / `-v` / `version` 时直接打印 `Version` / `BuildTime` / `GitCommit` 并 `os.Exit(0)`，不再继续加载 `.env`、初始化配置和启动 HTTP 服务；版本信息仍通过 `Makefile` 的 `-ldflags` 从根目录 `VERSION` 文件注入，行为与 `[Server-Info]` 启动横幅保持一致

### 修复

- **修复 Responses→Chat content parts 纯文本时 messages 为 null** (Issue #39) - `responsesContentToOpenAIChatParts` 此前仅在 content 数组包含图片时返回 parts，纯文本场景错误返回 nil，导致下游 `messages.content` 字段缺失，MiniMax 等上游报 `messages is empty`；移除 `hasImage` 判断，改为按解析出的 part 数量决定是否返回，覆盖 `type:text` / `type:input_text` / content 字符串 / input 字符串四种 input 形态；新增 `openai_converter_issue39_test.go` 与 `responses_to_chat_issue39_test.go` 锁定回归

## [v2.6.86] - 2026-05-12

### 新增

- **能力测试超时时间调整为 30s** - 将能力测试超时时间从默认值调整为 30 秒，优化测试流程

### 修复

- **Responses→Chat 转换补齐 assistant tool_calls content 字段** - cucloud 等上游镜像对 OpenAI Chat 消息的 content 字段做了必填校验，assistant 消息仅含 tool_calls 时缺少 content 会导致 JSON 反序列化失败；转换时自动补齐空 content
- **编辑渠道弹窗能力测试前先保存表单变更** - 点击能力测试按钮时，如果表单有未保存的修改（modelMapping、baseUrl 等），先自动保存再触发测试；无修改时直接测试，跳过多余保存请求
- **修复 Codex 兼容开关文案挤占开关布局** - 修复前端 Codex 工具兼容开关文案过长导致开关组件布局错位

### 其他

- **go fmt 格式化对齐** - 统一 Go 源码格式
- **添加 .mcp.json 和 .serena/ 到 gitignore** - 补充工具配置文件的 gitignore 规则

## [v2.6.85] - 2026-05-12

### 新增

- **支持 X-Channel 请求头指定目标渠道** - 新增 `X-Channel` 请求头，可通过渠道名直接定位目标渠道，跳过促销/亲和/优先级自动选择逻辑，便于单渠道调试测试；`SelectChannel` 新增 `channelName` 参数，非空时直接按名称匹配；`HandleMultiChannelFailover` / `compact` / `models` 路径同步更新签名；补充 `TestSelectChannelByName` 测试覆盖
- **Codex 自定义工具兼容层** - Responses 渠道支持将 Codex CLI 的 `custom`/`namespace`/`web_search`/`local_shell`/`computer_use` 工具及字符串简写转换为 Chat Completions 兼容的 function 代理工具；`apply_patch` 拆分为 5 个结构化代理工具（add/delete/update/replace/batch）；响应侧自动将 upstream function call remap 回 `custom_tool_call` 格式；流式和非流式路径均支持；支持 `custom_tool_call`/`custom_tool_call_output` 历史回放

### 变更

- **统一 Codex 工具兼容开关为 `codexToolCompat`** - 将原有 `stripCodexClientTools`（Responses 透传剥离）和 PR #43 的 `codexToolsCompat`（Chat 上游转换）合并为单一 `codexToolCompat` 开关；行为按上游 `serviceType` 自动分支：Responses 透传时剥离 Codex 专属工具，Chat/Claude/Gemini 上游时转换为 function 代理；旧配置 `stripCodexClientTools` 自动迁移；前端仅在 Responses 渠道显示该开关
- **扩展 Responses 请求工具类型支持** - `ResponsesRequest.Tools` 新增 `RawTools` 字段保留原始 `tools` 数组（含字符串简写），自定义 Marshal/Unmarshal 确保字符串工具不丢失；Codex 兼容层支持字符串工具名和 `web_search`/`local_shell`/`computer_use` 类型转为通用 function 代理

## [v2.6.84] - 2026-05-12

### 新增

- **新增 Responses Codex 工具兼容开关** - Responses 渠道新增 `stripCodexClientTools` 配置与前端开关，用于显式兼容不支持 Codex CLI 0.130+ 工具结构的旧版上游；默认保持原样透传，避免影响原生支持新版协议的上游；该开关同时覆盖原生 Responses 透传与 Responses 转 Chat/Claude/Gemini 路径
- **扩充模型优先级排序规则** - 前端 `AddChannelModal.vue` 的 `modelPriorityPatterns` 覆盖 2026-05 主流模型：Claude 4.7/4.6/4.5、GPT-5.5/5.4/5.3-codex/5.2/5.1/5、Gemini 3.1/3/2.5、Grok 4.3/4.2/4.1、GLM-5.1/5/4.7/4.6、Qwen 3.6/3.5/3-Max/3-Coder、DeepSeek V4/V3.2、Kimi K2.6/K2.5、MiniMax M2.7/M2.5；保持「pro/codex/max > 主版本 > mini/nano」与「新 > 旧」顺序
- **上游错误日志使用 4KB 长摘要** - `backend-go/internal/handlers/common/failover.go` 新增 `truncateErrorSummary`（4KB 上限并附 `...(truncated)`），`upstream_failover.go` 的上游错误详情摘要日志改用该函数，便于排查协议/schema 类问题；指标/原因字段仍走 200 字符的 `truncateMessage`
- **双通道日志输出** - stdout 始终输出精简格式（simplify tools + compact arrays + 缩进，不截断），日志文件始终写入原始完整 JSON；`logger.Setup` 将 stdout 与文件 writer 拆分，所有请求/响应日志点统一双写

### 修复

- **允许 Responses 工具协议错误跨渠道重试** - 对 `tools[n].tools`、工具参数 schema 等上游工具结构兼容错误放行 failover，避免第三方镜像 400 直接阻断后续渠道
- **修正模型映射匹配方向** - `config_utils.go` 的 `GetMappedModel` / `GetReasoningEffort` 不再做反向包含匹配（移除 `strings.Contains(m.source, model)`），仅保留 `model contains source`，避免短别名（如 `gpt`）误匹配到长目标键导致映射错配
- **truncateMessage 截断函数 UTF-8 安全** - `truncateMessage` 和 `truncateErrorSummary` 改用 `[]rune` 截断，避免多字节字符被切断产生乱码；`truncateMessage` 上限从 200 提升至 800 字符
- **保留 Responses 配对工具历史，仅降级孤立输出** - `normalizeStatelessResponsesToolHistory` 改为仅降级无对应 `function_call` 的孤立 `function_call_output`，配对的工具调用/输出历史保持原样透传
- **兼容无 session 的重放式工具历史** - 无 `previous_response_id` 的重放式请求不再因缺少 session 而丢失工具历史
- **修复文件日志缺失全量记录** - `logger.Setup` 将全局 `log.Printf` 输出目标从仅 stdout 改为 `io.MultiWriter(stdout, file)`，确保 300+ 处 `log.Printf` 调用均写入 `app.log`；`Console=false` 时不再丢弃日志改为仅写文件

## [v2.6.83] - 2026-05-11

### 新增

- **支持切换思考参数风格** - Chat 渠道新增思考参数风格切换能力，并调整前端配置位置以便管理

### 修复

- **保留 Responses 转 Chat 图片输入** - 修复 Responses 转 Chat 协议转换时图片输入丢失的问题
- **统一默认代理访问密钥** - 统一配置中的默认代理访问密钥，避免前后端默认值不一致

## [v2.6.82] - 2026-05-10

### 修复

- **修复能力测试协议路由** - 修正能力测试中协议路由逻辑，确保测试请求使用正确的协议路径
- **单模型测试仅保留请求模型** - 能力测试在单模型场景下仅保留用户请求的模型，避免多余模型干扰测试结果
- **保留同源模型重定向测试** - 能力测试保留同源模型的重定向验证，确保重定向场景正确覆盖
- **移除测试完成后的总耗时显示** - 清理能力测试完成后多余的总耗时输出

## [v2.6.81] - 2026-05-10

### 修复

- **修复前端环境类型声明** - 补充 Vue 单文件组件模块声明，并将运行时全局变量声明调整为 ambient 形式，避免全局类型声明在 Vite/Vue 类型检查中失效

## [v2.6.80] - 2026-05-10

### 重构

- **重构前端认证 Store** - 将认证状态管理从 setup store 切换为 option store，保留认证状态、getter、action 与持久化字段行为，减少 Pinia 持久化插件类型推断问题

## [v2.6.79] - 2026-05-10

### 新增

- **支持 max 思考强度** - 前端渠道高级参数的 reasoning effort 选项、类型定义与请求序列化支持 `max`，并补充 payload 测试覆盖

### 修复

- **虚拟协议能力测试使用实际上游协议** - 重试和重定向验证在虚拟协议场景下按渠道的实际上游协议构造探测请求，避免用虚拟协议路径误测；补充 HTTP 与重定向测试覆盖

## [v2.6.78] - 2026-05-10


### Changed

- **前端内存优化（第三批：SVG / Tooltip 常驻资源收敛）** - 针对 100+ 渠道场景下 heap 基线过高（实测 254MB 快照、live 约 1GB）的深度治理
  - `ChannelOrchestration.vue`：活动波形图由「每渠道 150 个独立 `<linearGradient>`」改为全组件共享 7 个成功率档位 gradient（`ccx-act-g0..g6`），所有 `<rect>` 通过 `url(#id)` 引用；同时移除外层 `<g>` 包裹、`bar.v === 0` 时跳过 rect 渲染（零请求段不落 DOM）。106 渠道场景下 `SVGLinearGradientElement` 从 15,901 降到 7、`SVGStopElement` 从 31,801 降到 14、`SVGGElement` 从 15,901 降到 0，快照总大小 254MB → 123MB
  - `ChannelOrchestration.vue`：bar 数据模型 `{x, y, width, height, radius, color}` 改为 `{x, y, width, height, radius, g, v}`，`g` 为 7 档 gradient id、`v` 为可见位，视觉输出与原 7 档色板一致
  - `ChannelOrchestration.vue`、`ChannelStatusBadge.vue`：metrics-display 统计 tooltip 与状态徽章 tooltip 改为 hover/focus 懒挂载（外层 div 监听 `mouseenter`/`mouseleave`/`focusin`/`focusout` 维护 `hovered` / `hoveredMetricsChannel` ref，`<v-tooltip v-if=... activator="parent">` 仅在激活时创建 overlay）。从每渠道常驻 2 个 overlay 降为全局最多 1 个，去除 Vuetify 3,000+ 个 `{activator, persistent, ...}` 响应式对象

- **前端 GC / 内存优化（第二批）** - 降低长时间运行时的内存增长速率和 GC 抖动频率
  - 新增 `useGlobalTick` composable：相同 `intervalMs` 的多订阅者共用一个 `setInterval`（5 个 5s 组件合并为 1 个定时器）；`visibilitychange` 时自动暂停所有 timer，恢复时若已超期立即补触发一次
  - `stores/channel.ts`：自动刷新定时器切换到 `registerGlobalTick`；`mergeChannelsWithLocalData` 抽到 `@/utils/channelMerge` 便于测试，用 `Map<index, Channel>` 预索引将合并复杂度从 O(N²) 降到 O(N)；`apiKeys` / `disabledApiKeys` / `modelMapping` 通过 `Object.freeze` 跳过 Vue 深度 Proxy 化
  - `ChannelOrchestration.vue`、`GlobalStatsChart.vue`、`KeyTrendChart.vue`、`ModelStatsChart.vue`、`ChannelLogsDialog.vue`：独立 `setInterval` 全部替换为 `useGlobalTick`
  - `ModelStatsChart.vue`：新增 `chartRef` + silent 模式下 `updateSeries`，避免每 5s 整图重绘
  - `ChannelOrchestration.vue`：bars 缓存加 `markRaw` 防止未来被误 Proxy 化
  - `App.vue`：修复 `mediaQuery` 事件监听泄漏（`onUnmounted` 补 `removeEventListener`）

### Added

- **能力测试新增「重定向验证」独立结果区域** - 渠道若配置了 ModelMapping，点击能力测试时会额外用该渠道类型的原生探测模型走一遍 ModelMapping 重定向并发往上游，直接验证"重定向规则+上游"链路是否可用，结果在能力测试对话框上方独立展示，不影响下方 4 协议原生兼容性测试
  - 后端：新增 `runRedirectVerification()` 并发编排器和 `executeRedirectModelTest()` 单模型测试函数；仅测试命中 ModelMapping 的探测模型，未命中则 `RedirectTests` 为空；复用现有 dispatcher RPM 限流、流式检测和渠道日志；渠道日志记录重定向后的实际模型名（与代理运行时一致）
  - 后端：`CapabilityTestJob`、`CapabilityTestResponse`、`CapabilitySnapshot` 新增 `RedirectTests []RedirectModelResult` 字段，snapshot clone/build/merge 同步处理
  - 后端：`buildCapabilityCacheKey` 和 `buildCapabilityExecutionLookupKey` 新增 `modelMappingHash` 参数，新增 `hashModelMapping()` 辅助函数（按 key 字典序 SHA-1 截 16 位 hex），ModelMapping 变更时缓存自动失效
  - 前端：`CapabilityTestDialog.vue` 状态栏下方新增重定向验证独立区域，徽标展示 `原模型 → 目标模型` 映射关系及成功/失败图标，悬停 tooltip 显示延迟、流式支持、错误详情
  - 前端：`api.ts` 新增 `RedirectModelResult` 接口；`vuetify.ts` 按需导入 `mdiArrowRightThin` 图标；`messages.ts` 三种语言补充 `capability.redirectTestTitle/redirectTestDescription/redirectTestEmpty/redirectedTo`
  - 测试：`capability_cache_key_test.go` 新增两条用例验证 ModelMapping hash 差异导致缓存 key 不同、`hashModelMapping` 对 key 顺序不敏感

- **前端单元测试补齐** - 为本批改动中的纯逻辑新增 32 个单元测试（总计 155 通过）
  - `channelMerge.test.ts`：冻结幂等性、预索引、5 分钟 latency 有效期边界、1000 项 O(1) 查找（11 tests）
  - `expandSparseSegments.test.ts`：稀疏展开、reuse 复用、防 API 数据污染回归（9 tests）
  - `useGlobalTick.test.ts`：共享 timer、visibility 暂停/补触发、组件 unmount 自动退订、回调抛错隔离（12 tests）
  - `package.json` 新增 `test` / `test:watch` 脚本

### Fixed

- **修复上游 model_not_found 503 错误被误判为可重试故障** - 上游 new-api 对 `model_not_found` 返回 HTTP 503，原逻辑将其视为临时故障触发全量 failover（所有 key/channel 耗尽后才返回 503）。新增 `isModelRoutingError` 识别 `model_not_found` 错误码，并通过 `normalizeUpstreamErrorStatus` 将最终 5xx 状态码归一化为 404 返回客户端；failover 仍允许跨 channel 尝试（不同上游实例可能支持该模型）

## [v2.6.77] - 2026-05-05

### 修复

- **黑名单过期 token 错误** - 将过期 token 相关的错误信息加入熔断黑名单，避免因上游返回的 token 过期错误触发不必要的故障转移

## [v2.6.76] - 2026-05-04

### 修复

- **移除 Gemini 和 Passthrough 路径中 responses input 的 status 字段** - `stripStatusFromResponsesInput` 处理也覆盖 Gemini→Responses 转换器和 Passthrough 通道，防止上游收到 `Unknown parameter: input[n].status` 报错

## [v2.6.75] - 2026-05-04

### Fixed

- **修复 Messages→Responses thinking 请求字段兼容** - Claude `thinking` block 转 Responses `input` 的 `reasoning` item 时不再写入仅响应侧使用的 `status` 字段，避免上游报 `Unknown parameter: input[n].status`
- **修复 Gemini→Responses thinking 请求字段兼容** - Gemini `thought` part 转 Responses `input` 的 `reasoning` item 时不再写入 `status` 字段，消除上游 `Unknown parameter: input[n].status` 报错
- **修复 Passthrough 路径 input status 泄漏** - `normalizeResponsesInputForPassthrough` 清除所有 input item 上的 `status` 字段，防止客户端回传 response output 时 `status` 泄漏到上游

## [v2.6.74] - 2026-05-04

### Added

- **Responses 渠道内置源模型列表新增 mini 模型** - 前端渠道编辑弹窗的 Responses 内置源模型选项中新增 mini 模型

### Fixed

- **修正 Chat→Responses 缓存 usage 口径** - OpenAI/Responses 风格的 `cached_tokens` 不再转换为 Claude 顶层 `cache_read_input_tokens`，并在协议转换输出中将 cache read 从 `input_tokens` 扣除，避免 `total_tokens` 重复累计缓存命中 token
- **修正 Responses→Chat 工具调用消息顺序** - 修复 responses 协议转换时工具调用消息排序不正确的问题，确保多轮工具调用按正确顺序排列
- **补充渠道日志弹窗缺失的 i18n** - 替换 `ChannelLogsDialog` 中硬编码的中文字符串（连接/首字/总计时长标签、请求状态文本），改用 i18n 键值；`toLocaleTimeString` 改为使用系统 locale 而非硬编码 `zh-CN`
- **清除前端剩余硬编码中文并移除死代码** - 替换 `ChannelOrchestration` 中硬编码的 tooltip 文本；移除 `system` store 中未使用的 `systemStatusText`/`systemStatusDesc` 方法及 `RETRO_THEME.name` 字段

## [v2.6.73] - 2026-05-04

### Added

- **OpenAI Chat 渠道新增非标准 role 规范化选项** - 新增 `normalizeNonstandardChatRoles` 配置项，启用后将非标准 Chat role（如 `developer`、`function` 等）自动改写为 `user`，提高与不支持扩展 role 的上游兼容性；前端渠道编辑弹窗增加对应开关
- 新增 `backend-go/internal/converters/chat_roles.go` 实现 role 规范化逻辑

### Fixed

- **修复 Chat 协议转换中工具调用链丢失问题** - 修复 responses 协议转换时 tool call chain 被截断的问题，确保多轮工具调用链完整保留

### Changed

- **提取 `buildChatCompletionRequestBody` 统一请求体构造逻辑** - 将 openai/gemini/default 三个分支中重复的 model 映射和参数注入逻辑抽离为独立函数，消除重复代码

## [v2.6.72] - 2026-05-03

### Added

- **全面支持 DeepSeek thinking 内容跨协议透传** - 在 `chat`、`messages`、`responses`、`gemini` 四条链路中完整支持 `reasoning_content` / `thinking` 字段的双向透传：
  - Chat API：Claude → OpenAI 转换时 `thinking_delta` 输出为 `delta.reasoning_content`；OpenAI 上游的 `reasoning_content` 回传时注入为 Claude `thinking` block
  - Messages API：OpenAI 上游 `reasoning_content` 转换为 Claude `thinking` content block
  - Responses API：新增 `claude_to_responses` 转换器，支持 Claude thinking → Responses reasoning 转换
  - 新增全链路 DeepSeek thinking 矩阵测试覆盖

### Fixed

- **修复 Failover Fuzzy 模式下 5xx 误判为不可重试错误** - `ShouldRetryWithNextKey` 的参数校验类不可重试错误检查（`invalid_request` 等）现在仅对 4xx 客户端错误生效，5xx 服务端错误允许 failover 到下一个渠道；内容审核类错误仍在任何状态码下阻止 failover，避免重复发送相同违规请求

### Changed

- **统一各协议上游响应日志输出** - 抽取公共响应头/响应体日志函数，统一 `messages`、`responses`、`gemini`、`chat`、`images` 非流式响应日志，并补齐流式响应头及 `chat`、`gemini`、`images` 上游流式原始内容日志

## [v2.6.71] - 2026-05-02

### 修复

- **统一 Chat/Codex 能力测试的模型探测顺序与 gpt-5.5 对齐** - 修正 `capability_probe_models` 中 Chat 和 Codex 协议的模型探测顺序，使前后端一致使用 gpt-5.5 作为优先探测模型

## [v2.6.70] - 2026-04-30

### 新增

- **引入 httptrace 生命周期追踪优化上游请求状态上报** - 新增 `RequestLifecycleTrace` 回调结构体，支持连接建立和首字节到达事件；封装 `SendRequestWithLifecycleTrace` 注入 `httptrace.ClientTrace`，upstream failover 使用生命周期回调替代事后状态更新，使日志状态更精确

### 文档

- **规范发布公告空分组输出** - 改进发布流程中空分组的输出格式

## [v2.6.69] - 2026-04-28

### Changed

- **为 Images 渠道日志补充具体端点标识** - 后端 `ChannelLog` 新增 `operation` 字段并透传到前端日志弹窗，Images 请求现在可直接区分 `generations`、`edits`、`variations`，便于排查不同图片端点的路由命中与重试情况

### Fixed

- **增强 Images 本地失败诊断日志并保护敏感信息** - 为 `multipart` 校验失败、JSON 参数校验失败和本地构建上游请求失败补充分阶段诊断日志，仅输出 `operation`、`content-type`、`body_bytes`、`stage/reason` 与脱敏 key 等上下文，不记录原始 `multipart` body、文件名、prompt 原文或未脱敏凭证
- **修复手动登录成功后系统状态未及时同步** - 前端在手动输入 access key 并完成 `refreshChannels()` 后，会立即根据最新刷新结果同步 `systemStatus` 为 `running` 或 `error`，避免界面状态继续停留在 `Connecting`

## [v2.6.68] - 2026-04-28

### Added

- **新增 OpenAI Images edits/variations 代理入口** - 补齐 `/v1/images/edits` 与 `/v1/images/variations`（含 `routePrefix` 变体），支持沿用 Images 渠道的多 key failover、metrics、channel logs 与 `#` BaseURL 语义，并新增 multipart 请求重写与回归测试覆盖 `model` 映射、文件字段保留和流式标记识别
- **新增独立 Images 渠道与 OpenAI Images 代理入口** - 新增 `imagesUpstream` 配置、`ChannelKindImages` 调度类型、`/v1/images/generations` 与 `/api/images/channels/*` 管理接口，支持独立的 key 管理、排序、状态切换、promotion、metrics/history、logs、ping 与 models 查询，避免图片请求与 Chat 渠道混用指标和熔断状态
- **前端接入 Images 渠道页签与基础运维能力** - 在管理界面新增 Images 标签页，接入 dashboard、CRUD、排序、promotion、日志、图表和模型查询；Images 渠道默认走 OpenAI 兼容语义，并隐藏未实现的 capability test 入口

### Changed

- **精简能力测试弹窗的冗余状态提示与来源文案** - 移除“共享结果 / 当前执行状态 / 测试范围”等低信息量提示、空态引导与 loading 副文案，删除前端未实际区分来源的 `snapshotSource` 字段，仅保留运行模式、兼容协议、进度与更新时间，降低 `CapabilityTestDialog` 与相关 i18n 文案的视觉噪音和误导性。
- **扩展调度器、指标迁移与回归测试以支持第五类渠道** - `scheduler`、`channel_metrics_handler`、SQLite metrics key 迁移、模型查询 fallback 与相关 handler/scheduler 回归测试统一纳入 Images 渠道，保持与 messages / responses / chat / gemini 一致的隔离和恢复语义
- **将能力测试 RPM 从渠道配置迁移到测试弹窗** - 在能力测试对话框新增默认值为 `10`、范围为 `1–60` 的 RPM 输入；前端请求与后端能力测试入口同步接收并钳制 `rpm`，同时从渠道级配置、payload 与管理视图中移除 `channel.rpm`，避免将测试速率持久化到渠道配置。
- **扩展能力测试为多协议并发启动与独立轮询** - 能力测试弹窗允许分别启动多个协议测试，前端按协议维护 `jobId` 引用并恢复多个活跃任务的轮询，避免后续启动覆盖已有协议状态与进度显示。
- **保留能力测试多协议恢复与继续执行语义** - 取消后恢复旧任务时继续保留已选协议，未开始的协议仍可继续加入测试，减少多协议测试被中断后的状态丢失与重复操作。

### Fixed

- **避免 multipart 图片请求在开发日志中输出原始二进制体** - `multipart/form-data` 的 Images 请求在开发环境下仅记录省略提示和请求头，避免日志污染与大体积二进制输出影响排查效率
- **修复编辑渠道时目标模型查询触发表单重载** - 编辑弹窗在静默保存当前渠道后不再对同一渠道执行 `loadChannelData` 回填，并让同一渠道的 watcher 更新保持 `noop`，避免点击目标模型名时重复请求 `/models` 且清空用户已选的源模型名；同时补充对应回归测试覆盖同渠道静默保存场景。
- **补齐能力测试 RPM 的前后端边界保护** - 前端发起请求前统一将 `rpm` 约束到 `1–60`，后端在缺省或越界时回退到安全值（默认 `10`，最大 `60`），保证能力测试速率行为一致且可控。
- **修复能力测试 snapshot 跨协议覆盖丢失** - `replaceFromJob` 从全量替换改为协议级合并，确保多协议独立 job 的 `ProtocolJobIDs` 和 `Tests` 不会互相覆盖，重新打开对话框能正确恢复所有协议的任务状态。
- **修复能力测试取消时重复 DELETE 导致状态刷新被跳过** - 对 `activeEntries` 中的 jobId 去重后再执行取消，取消与状态查询各自独立 try/catch，避免 legacy 多协议 snapshot 恢复后因 409 错误中断 UI 状态更新。
- **修复跨 Tab 恢复运行中能力测试时轮询/取消 404** - snapshot 新增按协议保存原始 `jobId/channelKind/channelId` 引用，前端轮询、取消与重试统一按协议使用原始 job 路径，确保跨协议标签页恢复相同 identity 的运行中任务时仍能继续跟踪和取消。

## [v2.6.67] - 2026-04-22

### Changed

- **统一渠道状态管理暴露层与恢复编排** - 为 channels 列表、dashboard、metrics/history、status/promotion、resume 与 ping 抽取共享 view / handler / transition helper，收口 messages / responses / chat / gemini 四类渠道管理接口的重复实现，并保持状态语义、运行时状态与返回结构一致
- **补齐渠道状态与连通性回归测试** - 新增和更新 handlers/config/scheduler/metrics/transitions 相关测试，覆盖统一状态视图、Chat 自动 suspended、promotion/status 接口、自动恢复编排，以及 chat / gemini / responses / messages 的 ping 路径
- **支持模型过滤增强为包含/排除规则** - `supportedModels` 现支持精确匹配、`prefix*`、`*suffix`、`*contains*` 以及 `!` 排除规则；非法中间通配如 `foo*bar` 在前端会被拦截为无效输入，后端会跳过该条规则而不影响其他合法规则生效
- **修复 Responses Compact 多渠道模型过滤绕过** - `/v1/responses/compact` 多渠道选择现在会携带原始请求模型参与 `supportedModels` 过滤，并补充后端调度/handler 回归测试与前端规则校验测试

### Fixed

- **修复自动恢复错过 UTC 槽位后不补跑** - 为定时自动恢复新增基于持久化上次检查时间的启动补偿检查与运行中兜底检测；当服务启动、容器恢复或宿主机从睡眠恢复后，仅在确实错过最近一个 UTC `00:00:01` / `08:00:01` / `16:00:01` 恢复槽位时才会补跑，并始终按真实错过的槽位时间执行恢复判定，避免提前恢复仍处于冷却中的 key；同时补充调度时序与状态持久化单元测试覆盖最近槽位、错过槽位与跨重启判定

## [v2.6.66] - 2026-04-20

### Fixed

- **补充日额度耗尽错误的 key 拉黑识别** - 将 `USAGE_LIMIT_EXCEEDED`、`DAILY_LIMIT_EXCEEDED` 及 `daily usage limit exceeded` 等错误码/消息统一识别为额度耗尽，触发自动拉黑 key 而非仅临时失败，并补充 HTTP 与 SSE 回归测试覆盖 Responses/流式场景
- **区分客户端取消与失败日志终态** - 将客户端主动取消的请求标记为独立 `cancelled` 终态，并补充后端回归测试与前端状态样式，避免在渠道日志中误显示为失败
- **统一进行中日志高亮样式** - 将进行中请求的视觉强调集中到状态码徽章，移除行级左侧高亮，保持渠道日志列表的对齐一致性与可读性

### Other

- **忽略 Python 字节码缓存目录** - `.gitignore` 新增 `__pycache__/`，避免本地探测脚本生成的缓存目录污染工作区

## [v2.6.65] - 2026-04-20

### Added

- **编辑弹窗新增静默保存后执行能力** - 编辑已有渠道时，点击“目标模型名”或“能力测试”会先验证并静默保存当前表单，再基于保存后的最新配置继续执行；`rpm`、`proxyUrl`、`baseUrl`、`apiKeys` 等修改无需手动保存即可生效
- **渠道日志实时显示请求生命周期状态** - 扩展 `ChannelLog` 结构支持请求状态追踪（pending/connecting/first_byte/streaming/completed/failed），在请求各阶段实时更新日志状态，前端显示状态标签、各阶段耗时（连接耗时、首字节耗时、总耗时）和进行中请求的脉动动画效果
- **渠道日志默认自动刷新** - 前端日志对话框打开时自动开始 3 秒轮询，移除手动刷新按钮，关闭对话框时自动停止查询

### Changed

- **能力测试协议顺序与 Claude 探测优先级对齐** - 默认能力测试执行顺序统一调整为 `messages → responses → chat → gemini`，前后端排序与首屏占位保持一致；同时将 Claude 探测模型优先级更新为优先探测 `claude-opus-4-7`，减少展示顺序与实际执行顺序不一致
- **能力测试 RPM 输入移至测试按钮旁** - 编辑渠道时将能力测试 RPM 控件移动到弹窗右上角测试按钮旁，并收窄输入框宽度；新增渠道时不再显示该控件，提升编辑场景的就近操作效率
- **扩展按渠道模型查询的临时连接参数** - 前端模型查询请求与四类后端 `/channels/:id/models` 入口新增支持 `proxyUrl`、`insecureSkipVerify` 与 `customHeaders`，新增渠道场景也可直接带临时连接参数获取模型列表
- **公共 `/v1/models` 复用渠道代理与自定义请求头** - 聚合模型列表与模型详情查询现在会沿用渠道配置中的 `proxyUrl` 和 `customHeaders`，与正式转发链路保持一致
- **统一 Base URL 等价去重与请求预览语义** - 为前后端新增共享 canonical Base URL 规则，将根域名与默认版本前缀 URL（如 `/v1`、`/v1beta`）视为等效并保留最短形式，同时继续保留 `#` 作为独立语义；同步影响渠道新增/编辑、快速输入解析、payload 构建与预期请求 URL 预览
- **兼容等效 Base URL 的历史指标与图表聚合** - 后端配置、运行时指标 key、历史统计与图表聚合改为按等效 Base URL 兼容读取，避免用户在 `root` 与 `/v1` 之间切换后出现历史访问记录和图表数据断裂，并补充前后端回归测试
- **优化渠道日志记录机制** - 新增 `CreatePendingLog`、`UpdateLogStatus`、`CompleteLog` 函数支持日志生命周期管理，`ChannelLogStore` 新增 `Update` 方法支持通过 `requestID` 更新已存在的日志条目
- **新增 UTC 0/8/16 时段自动恢复黑名单 key** - 为因余额/额度类原因自动拉黑的 key 增加基于 UTC `00:00:01`、`08:00:01`、`16:00:01` 的定时恢复编排，恢复后将 key 切入 `half_open` 探测而非直接回到 `closed`，并跳过 1 小时内刚自动封禁的 key 以顺延到下个时段
- **自动恢复时按渠道状态最小激活** - 当渠道因 active key 为空而处于 `suspended` 时，若本轮恢复出了可用 key，则自动恢复为 `active`；`disabled` 渠道保持不变，避免误激活手动禁用渠道
- **补充恢复编排与熔断回归测试** - 新增 metrics/scheduler 单测覆盖 UTC 时段计算、可自动恢复 reason 筛选、多 BaseURL half-open 迁移与 suspended 渠道激活规则

### Fixed

- **修复 Responses 桥接误把会话标识映射到 `user` 字段** - Claude Messages 转发到 Responses 上游时，不再把 `metadata.user_id`、`X-Claude-Code-Session-Id` 或 `X-Client-Request-Id` 回填到 Responses `user` 字段，统一仅用于 `prompt_cache_key`，避免桥接请求携带非预期用户身份
- **修复渠道日志可读性与进行中状态展示** - 日志列表为进行中请求增加透明占位状态码徽章，内联展示 `keyMask` 与 `baseUrl` 便于排查路由命中，并将连接/首字节/总耗时统一改为秒级格式，提升高频扫描可读性
- **修复编辑渠道切换时弹窗临时输入残留** - `AddChannelModal` 在关闭弹窗、切换新增/编辑模式和切换编辑渠道时统一重置自定义请求头输入框，并同步清理新 API Key、模型映射等临时表单状态，避免上一个渠道的草稿残留到下一个渠道
- **修复合并后的指标 serviceType 签名不一致** - 统一自动恢复与熔断相关代码、测试对新的 metrics identity/serviceType 签名的调用，修复 worktree 合并后 `MoveKeyToHalfOpen`、`GetKeyCircuitState` 与对应回归测试的编译错误，涉及 `internal/metrics` 与 `internal/scheduler`
- **修复客户端取消请求的日志终态缺失** - 在客户端取消分支（`context.Canceled`）中补充 `CompleteLog` 调用，避免日志永久停留在进行中状态
- **修复日志并发读写安全问题** - `ChannelLogStore.Get` 方法返回深拷贝而非共享指针，避免 HTTP 序列化与日志更新并发时的数据竞争
- **修复前端进行中请求误标为失败** - 仅在 `status === 'failed'` 时显示错误背景，`statusCode === 0` 时显示 `-` 而非 `ERR`，避免 pending/connecting 请求被误判为失败
- **修复连接阶段时间戳记录时机** - 将 `ConnectedAt` 时间戳记录移到收到上游响应后，确保连接耗时准确反映 DNS 解析、TCP 握手和 TLS 协商的真实耗时
- **修复高并发场景下终态日志丢失** - `CompleteLog` 在 `Update` 失败时补写终态日志，避免长请求在环形缓冲淘汰后静默丢失生命周期记录
- **修复渠道删除时的日志索引污染** - 在 `ChannelLog` 中记录创建时的渠道索引，补写日志时验证索引匹配，避免在渠道删除导致索引漂移时将日志写入错误的渠道
- **修复终态日志补写条件失效** - 将渠道日志 `Update` 返回值改为区分“正常找到 / 环形缓冲淘汰 / 渠道删除”，并在索引漂移场景下跨索引查找请求，仅在确认被淘汰时补写终态日志，避免高并发下终态丢失且不再污染被删除或移位后的渠道日志
- **修复删除后已淘汰请求的终态误判** - 为渠道日志新增独立的在途请求索引 `requestLocations`，仅跟踪未完成请求；`Update` 现在基于在途索引区分“仍在途但已被环形缓冲淘汰”与“渠道删除后请求已失效”，避免长请求先淘汰后删渠道时把终态日志错误回填到移位后的渠道或幽灵日志桶
- **修复删除渠道时残留在途索引污染** - `RemoveAndShift` 现在会先统一重写全部 `requestLocations`：删除指向被删渠道的在途请求索引，并将其后渠道的在途请求索引整体前移；这样即使请求日志已先被环形缓冲淘汰，删除渠道后也不会因旧索引残留而把终态日志误补写到复用后的渠道桶
- **修复移位后淘汰请求的终态回填索引** - `Update` 现在返回在途请求的当前实际渠道索引；`CompleteLog` 在请求已移位且日志已被环形缓冲淘汰时，会按最新索引回填终态日志，而不再使用调用方持有的旧索引，避免日志写回失效索引或污染后续复用的渠道桶

### Other

- **前端开发依赖升级** - 升级 `eslint` `10.2.0 → 10.2.1`、`typescript` `6.0.2 → 6.0.3`、`vue-tsc` `3.2.6 → 3.2.7`

## [v2.6.64] - 2026-04-16

### Fixed

- **修正 Responses prompt total 保留逻辑避免缓存命中率误判** - 在非流式与直连 Responses 流式 handler 中改为仅基于 patch 前原始 usage 保留 `PromptTokensTotal`，避免把 patched `input_tokens` 误当总 prompt tokens 导致 dashboard 缓存命中率虚高到 100%，并补充 handlers/providers 回归测试
- **统一 Responses 来源的缓存命中率统计口径** - 在内部 usage 统计中保留 Responses/OpenAI 风格的总 prompt token 数，并在 dashboard metrics 聚合前归一化为未命中输入 token，修复 messages→responses 以及 direct responses 渠道缓存率被重复计入分母、前端显示约减半的问题；同时兼容 `input_tokens_details.cached_tokens` 回退并补充 bridge、stream、metrics、handler 回归测试

## [v2.6.63] - 2026-04-16

### Changed

- **能力测试请求接入渠道日志并补充来源标识** - 将模型能力测试与单模型重试产生的真实上游请求写入现有渠道日志，新增 `requestSource` 字段区分正式代理流量与能力测试流量，并在前端日志弹窗中显示“能力测试”标识，便于统一排查失败状态、耗时和错误详情
- **补充能力测试日志接入回归测试** - 新增能力测试成功/失败日志记录测试，以及渠道日志来源字段 helper 测试，覆盖后端日志来源默认值与显式写入路径

### Fixed

- **补齐 Responses 缓存命中 token 映射兼容** - 当上游 `usage` 未返回 `cache_read_input_tokens` 时，改为从 `input_tokens_details.cached_tokens` 回填 Claude Usage 的缓存读取 token，覆盖非流式与 SSE 流式响应，并补充对应回归测试

## [v2.6.62] - 2026-04-16

### Fixed

- **放宽熔断恢复阈值并补充 failover 错误摘要日志** - 将 half-open 恢复成功门槛从两次探针下调为一次，同步在切换到下一个 API Key 前记录截断后的上游错误摘要，并更新 dashboard 回归测试与熔断相关指标行为以匹配新的恢复语义

## [v2.6.61] - 2026-04-15

### Fixed

- **全黑名单 Key 场景下的模型列表与编辑态回退一致性** - 聚合 `/v1/models` 与编辑弹窗在活跃 API Key 全部被拉黑后，允许临时借用 `disabledApiKeys` 获取模型列表，并在回退时保留 routePrefix 隔离与运行时全黑名单渠道支持，同时保持正常调度不使用已拉黑 key
- **补充黑名单借 key 管理场景回归测试** - 新增管理场景 key 选择与模型列表聚合 fallback 的后端回归测试，覆盖 active key 优先、disabled key 回退与无 key 失败路径

## [v2.6.60] - 2026-04-15

### Fixed

- **能力测试全 Key 拉黑时回退逻辑不一致** - 修复当所有活跃 API Key 被拉黑后，能力测试入口直接报 no_api_key 而不尝试借用被拉黑 key 的问题，与 buildTestRequestWithModel 中已有的回退逻辑保持一致

## [v2.6.59] - 2026-04-15

### Fixed

- **能力测试全 Key 拉黑时重试 panic** - 修复当渠道所有 API Key 均被拉黑时，能力测试重试逻辑触发 panic 的问题

### Other

- **前端补丁依赖升级** - 升级前端 patch 级别依赖包

## [v2.6.58] - 2026-04-15

### Added

- **完整强化版熔断器** - 实现显式三态熔断状态机（closed/open/half_open），支持指数退避、单探针恢复和失败分类
  - 引入失败分类机制（retryable/non_retryable/quota/client_cancel），只有可重试故障触发熔断
  - 实现 half-open 单探针恢复机制，成功 1 次即完全恢复
  - 实现指数退避机制（30s base, 10min max），避免频繁重试
  - 新增 circuit_states 表持久化熔断状态，服务重启后保留
  - 调度器禁止 open 渠道被 fallback 选回，解决连续 500 仍打到坏渠道的问题
  - 升级 upstream_failover 和 responses/compact 使用新 breaker 状态机
  - Dashboard API 返回完整 breaker 字段（circuitState/halfOpenSuccesses/breakerFailureRate/nextRetryAt）
  - 前端状态徽章支持 breaker-open/half-open 显示，恢复按钮识别自动熔断
  - 涉及文件：`internal/metrics/channel_metrics.go`, `internal/metrics/persistence.go`, `internal/metrics/sqlite_store.go`, `internal/scheduler/channel_scheduler.go`, `internal/handlers/common/upstream_failover.go`, `internal/handlers/responses/compact.go`, `internal/handlers/channel_metrics_handler.go`, `frontend/src/components/ChannelStatusBadge.vue`, `frontend/src/components/ChannelOrchestration.vue`
- **新增跨接口一致性与桥接回归测试** - 增加统一会话标识提取测试、Messages→Responses 身份映射测试，以及 handlers 层跨 messages / responses / chat / gemini 的 affinity 一致性测试，覆盖缓存键与用户标识回退逻辑

### Changed

- **为 Messages/Responses 渠道增加 metadata.user_id 规范化开关** - 新增默认开启的 `normalizeMetadataUserId` 渠道配置与前端开关；请求入口保留原始 `metadata.user_id`，仅在发往上游前按渠道决定是否将 JSON 对象字符串扁平化，兼容需要透传原始对象和依赖旧扁平格式的不同上游；同步更新渠道列表/dashboard 返回与前后端回归测试
- **统一四协议的会话亲和与缓存身份提取** - 为 Messages / Responses / Chat / Gemini 入口统一引入会话标识提取优先级，新增 `X-Claude-Code-Session-Id` 与 `X-Client-Request-Id` 支持，并保留旧提取函数兼容现有调用
- **补全 Messages→Responses 桥接字段映射** - 在 Claude Messages 转 Responses 上游请求时补齐 `prompt_cache_key`、`user`、`top_p`、`tool_choice`、`parallel_tool_calls` 等字段，提升跨接口缓存复用与参数传递完整性

### Fixed

- **补齐 401 字符串认证错误的自动拉黑识别** - 当上游仅返回 `401` + 字符串 `error`/`message`（如 `{"error":"无效的API Key"}`）且缺少 `type`/`code` 时，非流式与 SSE 流式自动拉黑逻辑现在也会识别为 `authentication_error`，避免无效 key 仅触发 failover 而未被持久化拉黑；同步补充回归测试
- **收敛 half-open 探针并发窗口、健康判定与指标口径** - 调整 `upstream_failover.go` 与 `responses/compact.go` 的探针释放时序为“先记账、后释放”，避免 half-open 状态下并发请求重复抢占探针；将空 API Key 列表渠道统一判定为不健康；将 `IsKeyHealthy()` 的到期状态推进收敛到写锁内，避免读锁下写状态；并修正渠道聚合 `successRate/errorRate` 使用总请求统计、仅让 `breakerFailureRate` 使用 breaker 窗口，保证看板指标与真实请求结果一致，同时去除 closed / 非 breaker 相关热路径上的同步熔断状态持久化写入，降低默认持久化模式下的请求开销

### Removed

- **清理已下架模型引用** - 移除代码中对官方已下架的 `gpt-5.1-codex-max` 模型的所有引用

## [v2.6.57] - 2026-04-14

### Fixed

- **补齐字符串错误体的余额不足拉黑识别** - 非流式与 SSE 流式拉黑检测现在都支持从字符串 `error` / 顶层 `message` 中识别“额度不足”语义，能力测试链路同步复用该判定，避免此类上游错误漏判导致 key 未自动拉黑
- **修复能力测试与渠道健康检查的版本化 BaseURL 拼接** - 统一能力测试与 Responses/Chat/Gemini 渠道 ping/health-check 的端点构建逻辑，正确识别已包含 `/v1` 或 `/v1beta` 的 baseURL（如 `.../codex/v1`），避免重复追加版本前缀导致 `/v1/v1/...` 404，并补充相关回归测试

## [v2.6.56] - 2026-04-13

### Fixed

- **保留删除渠道后的历史日志** - 删除渠道时改为仅调整内存日志索引，避免清空其他渠道日志，并补充删除与仪表盘链路回归测试
- **补充余额不足错误码识别** - 非流式拉黑检测支持从 `code` 字段识别 `INSUFFICIENT_BALANCE`，并让能力测试与自动余额拉黑语义保持一致，避免误禁用 key
- **对齐紧凑 Responses 与活动日志记录** - 为 Responses compact 尝试和上游 failover 统一复用共享日志辅助逻辑，确保活动指标中出现的尝试也能在前端日志视图中看到

## [v2.6.55] - 2026-04-12

### Fixed

- **拆分 Vue 类型声明为 ambient 文件** - 将 Vue shims 拆分为独立的 ambient 声明文件，兼容 TypeScript 6.0 的类型解析要求

## [v2.6.54] - 2026-04-12

### Fixed

- **优化能力测试对话框模型列表排版** - 改善能力测试对话框中模型列表的排版布局和 tooltip 样式显示
- **缩小能力测试对话框操作按钮尺寸** - 减小能力测试对话框底部操作按钮的尺寸，提升视觉一致性

### Other

- **添加 .planning/ 到 .gitignore** - 将 `.planning/` 目录加入 Git 忽略列表

## [v2.6.53] - 2026-04-12

### Added

- **补充四协议互转矩阵测试** - 新增 Responses 请求矩阵、Messages 响应矩阵，以及 Messages/Chat/Gemini/Responses 入口的非流式 handler matrix 测试，覆盖四种上游协议的主要请求与响应路径

### Fixed

- **修复 Chat 非流式透传读空响应体** - `chat` handler 在非流式默认透传分支中重置已读取的 `resp.Body`，避免 `PassthroughJSONResponse` 二次读取时返回空响应体

## [v2.6.52] - 2026-04-12

### Changed

- **抽取能力测试模型结果子组件** - 将 `CapabilityTestDialog` 中移动端与桌面端重复的模型 tooltip、badge、空状态与 retry 渲染提取为独立 `CapabilityModelResults` 组件，统一模型区交互与样式来源，降低后续状态改动的分叉风险
- **恢复渠道时同步恢复被拉黑 Key** - `resume` 系列管理接口在重置渠道熔断状态时，同步恢复对应渠道 `DisabledAPIKeys` 中的全部 Key，并返回 `restoredKeys` 数量，避免仅清除熔断却遗漏渠道级 Key 恢复

### Fixed

- **对齐能力测试与代理空流判定** - 能力测试改为复用 provider 规范化后的流事件和代理侧 `PreflightStreamEvents` 预检逻辑，要求上游必须返回实际文本或语义内容才判定成功，并将流读取超时统一为 30 秒，避免空 SSE 流被误判为模型可用

## [v2.6.51] - 2026-04-10

### Changed

- **优化亲和调度优先级让渡规则** - 当存在更高优先级且健康的可用渠道时，trace affinity 不再强制命中旧绑定渠道，优先回到当前最优可用渠道，减少低优先级历史绑定长期占用流量
- **重构能力测试状态模型与前端展示** - 能力测试后端为 job/protocol/model 三层新增 lifecycle、outcome、reason、runMode 等状态语义字段，统一取消、重试、缓存命中与恢复任务的状态表达；前端同步切换到新状态模型，优化轮询控制、局部重测、协议/模型行级展示与错误提示文案，减少 completed/failed/cancelled 混淆
- **补充能力测试状态聚合回归测试** - 为 capability job 聚合、取消语义与 reason 映射新增单测，验证 partial / cancelled / timeout / not_run 等关键状态形状

### Fixed

- **修复 403/429 余额语义误分类** - 后端拉黑逻辑对 403/429 仅在错误消息明确表达余额或额度不足时才标记为 `insufficient_balance`，避免将普通 permission denied 等授权错误误判为永久失效；同步补充中英文额度文案与权限错误回归测试
- **修复新增渠道表单状态串扰** - 调整 AddChannelModal 对 `channel` 变更的监听逻辑，避免编辑态关闭或触发能力测试时错误切回快速添加，并确保重新新增渠道时 `baseURL` 等表单字段被正确重置
- **补充弹窗状态回归测试** - 新增 watcher 级别测试覆盖新增重置、编辑回填与编辑态清空 channel 不误切模式等场景

## [v2.6.50] - 2026-04-08

### Fixed

- **修复能力测试错误处理和状态映射** - 后端移除无 API Key 时返回的 `all` 伪协议，改为直接返回 `failed` 状态；前端增加已知协议白名单过滤，优化 `failed` 状态映射保留结果视图，补全印尼语 i18n 翻译

### Changed

- **规范化前端字号行高和按钮样式** - 统一字号规范（0.875rem、1.125rem）和行高规范（1.3、1.4、1.5、1.6），移除中间值；为按钮添加 `line-height: 1.5` 改善文字垂直居中；提取对话框标题样式到全局 CSS

## [v2.6.49] - 2026-04-07

### Fixed

- **修复非正天数参数验证缺陷** - 修复 7d/30d 时间范围支持的 3 个关键问题

### Changed

- **调整指标数据保留期默认值和范围** - 扩展渠道指标历史查询支持 7d 和 30d，统一全局统计与渠道统计的时间范围选项

## [v2.6.48] - 2026-04-07

### Changed

- **移除前端批量测试功能** - 删除批量测试入口按钮、对话框组件、相关多语言文案与专用图标，避免在管理界面触发大面积上游访问
- **精简前端能力测试代码路径** - 清理仅供批量测试使用的前端 capability job 类型与 API 包装，保留单渠道能力测试与常规渠道延迟测试所需能力

- **统一全局 Tooltip 样式** - 将分散在各组件中的 tooltip 样式（`fuzzy-tooltip`、`status-tooltip`）合并为全局 `ccx-tooltip` 类（复古像素主题），所有 `v-tooltip` 统一使用 `content-class="ccx-tooltip"` 避免 Vuetify 默认灰色；拉黑密钥 chip 颜色由 `error` 改为 `warning`

## [v2.6.47] - 2026-04-06

### Fixed

- **补注册 cash-remove 图标** - 修复前端缺少 `cash-remove` 图标注册导致组件渲染异常的问题

## [v2.6.46] - 2026-04-05

### Fixed

- **routePrefix 下 responses 渠道协议转换失败** - 修复 `ResponsesProvider.buildProviderRequestBody` 对请求路径的硬编码比较（`== "/v1/messages"`），改为 `HasSuffix` 匹配，使带 routePrefix 前缀的路由（如 `/:prefix/v1/messages`）也能正确触发 Claude→Responses 协议转换
- **恢复拉黑 Key 后统计重复计数** - 修复 `ConfigManager.RestoreKey` 未从 `HistoricalAPIKeys` 移除已恢复 key 的问题，避免 key 同时存在于 active 和 historical 列表导致指标聚合重复
- **Go 代码格式规范化** - 修正 `chat/channels.go` 和 `messages/channels.go` 中 `gin.H` 字面量的对齐格式（`go fmt`）

## [v2.6.45] - 2026-04-03

### Fixed

- **自动补全消息内容数组** - 修复对 Claude Code 新版请求中 `message.content` 为 `null` 或省略时的解析兼容性，避免 Messages 请求在进入转换链路前被错误拒绝

## [v2.6.44] - 2026-03-31

### Fixed

- **补全 Claude cache usage 统计** - 修复 Claude/Responses 协议转换与 `/v1/responses` handler 对 `cache_creation_input_tokens`、`cache_creation_5m_input_tokens`、`cache_creation_1h_input_tokens`、`cache_read_input_tokens` 的透传与 `total_tokens` 计算，确保流式与非流式场景下 usage 总量包含缓存读写成本
- **新增 usage 回归测试覆盖** - 为 converters、responses handler 与 provider bridge/stream 补充缓存 usage 断言，防止后续协议转换再次遗漏 TTL 维度缓存字段

## [v2.6.43] - 2026-03-29

### Fixed

- **保留上游请求 URL 尾部井号** - 修复后端请求转发与前端渠道保存时对 base URL 的规范化逻辑，避免将以 `#` 结尾的上游地址错误裁剪，确保特殊路由或占位配置可被原样保留

## [v2.6.42] - 2026-03-21

### Fixed

- **清洗 Read.pages 空串** - 规范化 Read 工具请求时过滤空字符串页码参数，避免将空 `pages` 传给 provider 导致校验或兼容性问题

## [v2.6.41] - 2026-03-18

### Changed

- **messages->responses system 指令过滤** - `/v1/messages` 转发到 responses 上游时，若 `system` 数组首项为 Claude Code 注入的 `x-anthropic-billing-header` 计费头，则不再映射到 `instructions`，仅保留真实 system 指令，避免协议元信息污染上游提示词

## [v2.6.40] - 2026-03-18

### Changed

- **metadata.user_id 通用 JSON 对象支持** - 扩展规范化逻辑支持任意 JSON 对象格式，优先处理 Claude Code 标准字段（`device_id/account_uuid/session_id`），对非标准格式按字母序拼接为 `key_value` 格式，确保更广泛的上游兼容性

### Fixed

- **修复 invalid_request 误拦截认证错误** - `isSchemaValidationError` 检查 `error.code` 排除 `invalid_api_key` 等认证错误，仅拦截真正的 schema/参数错误，保留多 key 场景下的 failover 能力

## [v2.6.39] - 2026-03-18

### Fixed

- **metadata.user_id JSON 对象兼容** - Claude Code v2.1.78 将 `metadata.user_id` 从扁平字符串改为 JSON 对象字符串，部分上游（如 anyrouter）严格校验导致请求失败；代理层自动检测并动态拼接为扁平格式（仅包含非空字段，如 `user_{device_id}` 或 `user_{device_id}_session_{sid}`）
- **schema/参数错误不再触发 failover** - 上游返回 `invalid_request_error`、`schema_validation_error` 等不可恢复的 4xx 错误时，不再切换渠道重试，避免同一份坏请求打挂所有同类渠道

## [v2.6.38] - 2026-03-15

### Added

- **能力测试取消功能** - 测试进行中可点击"取消测试"按钮立即中止，后端通过 context cancel 终止 goroutine 和 HTTP 请求，未完成模型标记为 skipped
- **复用成功结果** - 重新测试时自动传入上次 jobId，后端仅重测失败/跳过的模型，已成功的结果直接复用
- **单模型插队重测** - 点击失败/跳过的模型 badge 可触发该模型的单独重测，无需重跑整个流程

### Fixed

- **修复能力测试对话框空指针访问** - 在测试对话框中添加可选链操作符，防止 null 访问导致的崩溃

## [v2.6.37] - 2026-03-15

### Changed

- **能力测试对话框改为 prop 驱动** - CapabilityTestDialog 从 `defineExpose` + `ref.updateJob()` 命令式驱动改为 `capabilityJob` prop + `watch` 响应式驱动，消除 ref 可能为 null 导致数据丢失的问题
- **关闭编辑渠道弹窗时自动关闭能力测试弹窗** - 避免 Vuetify dialog 叠加后 overlay 栈错乱导致关闭按钮和 ESC 失效

## [v2.6.36] - 2026-03-14

### Fixed

- **能力测试 job store 内存泄漏** - 新增定时 GC（每 30 分钟），清理已完成/失败且超过 2 小时的 job，防止长期运行后无限增长
- **修复缓存命中时的死锁** - `createCapabilityJobFromResponse` 在 `getOrCreateByLookupKey` 持锁期间被调用，内部再次调用 `bindLookupKey` 加同一把锁导致死锁；移除函数内的 `bindLookupKey` 调用，由 `getOrCreateByLookupKey` 统一负责 lookupKey 绑定
- **缓存命中时 lookupKey 未绑定** - 修复缓存命中分支中 `createCapabilityJobFromResponse` 传空 lookupKey 的问题，同一渠道重复命中缓存时正确复用 job
- **jobID 极低概率碰撞** - 改用 `crypto/rand` 生成 16 字符随机 hex，消除纳秒级时间戳碰撞风险

### Changed

- **能力测试全量模型扫描** - 移除协议首次成功后跳过后续模型的早退逻辑，所有候选模型均会被逐一测试
  - 后端：去掉 `protocolStatus` 早退机制，改用 `protocolTimedOut` 仅标记超时强制中断；协议最终状态（completed/failed）在全部模型测完后统一更新
- **修复协议延迟统计虚高** - 收尾阶段改用每个协议自身的开始/结束时间计算延迟（`protocolEndTime`），避免串行执行时后续协议的耗时被计入先完成的协议

## [v2.6.35] - 2026-03-14

### Changed

- **能力测试 Round-Robin 串行调度** - 将并发竞争模型改为串行 round-robin 编排，跨协议交错调度（messages[0] → chat[0] → gemini[0] → responses[0] → messages[1]...），协议首次成功后自动跳过后续模型
  - 后端：新增 `runRoundRobinTests()` 编排器和 `executeModelTest()`，全局超时按 `max(interval, perModelTimeout)` 累加，协议早退出机制
  - 后端：新增 `skipped` 模型状态（`CapabilityModelStatusSkipped`），`ModelTestResult` 增加 `Skipped` 字段确保缓存重建不丢失状态
  - 后端：收尾逻辑区分「未实际测试」与「测试后全部失败」，回填残留模型结果，守护零时间延迟计算
  - 前端：新增 skipped 状态展示（灰色删除线 badge + `mdi-skip-next` 图标），三语言 i18n 支持
  - 前端：`failed` 任务也展示详细协议/模型结果，不再跳转到 error 页面

## [v2.6.34] - 2026-03-13

### Added

- **能力测试 RPM 配置** - 渠道新增 RPM 字段并在编辑页可配置，仅影响模型能力测试的发包节奏

### Changed

- **能力测试调度间隔** - 调度器按每个请求的 RPM 计算放行间隔，RPM 无效时默认回退 10

## [v2.6.33] - 2026-03-12

### 新增

- **能力测试对话框视觉优化** - 优化能力测试对话框视觉层级与交互体验

### 修复

- **移动端渠道编排模式标签** - 移动端隐藏渠道编排模式标签，优化移动端显示
- **渠道状态徽章对齐** - 修复渠道状态徽章垂直对齐问题

## [v2.6.32] - 2026-03-12

### 新增

- **多语言管理界面与文档** - 新增多语言管理 UI 与本地化文档，补充翻译内容
- **桌面端语言切换器** - 增加桌面端语言切换入口，完善多语言体验
- **能力测试展示完整模型结果** - 能力测试改为按 500ms 节流启动同协议下的全部候选模型请求，并汇总返回每个模型的可用性、流式支持、延迟与错误信息
  - 后端：`backend-go/internal/handlers/capability_test_handler.go` 新增 `modelResults` / `successCount` / `attemptedModels`，保留协议级摘要并记录模型级启动与完成时间
  - 前端：`frontend/src/components/CapabilityTestDialog.vue` 升级为”协议摘要 + 模型明细”视图，支持展示成功数/总数与各模型测试详情
  - 类型与文案：同步更新 `frontend/src/services/api.ts` 与 `frontend/src/i18n/messages.ts`
- **扩展能力测试模型列表** - 为各协议添加更多候选测试模型，提升测试覆盖率
- **移动端响应式优化** - 能力测试对话框与渠道列表针对移动端进行全面优化
  - 能力测试改用流式徽章布局，简化模型结果展示
  - 添加 tooltip 显示详细信息（延迟/错误原因）
  - 移动端采用卡片布局替代表格，避免横向滚动
  - 优化渠道卡片和列表的移动端响应式布局
  - 移动端渠道名称自动截断至 12 字符

### 修复

- **移动端头部与语言回退** - 修正移动端头部样式与语言回退逻辑
- **频道状态标记遮挡** - 修复暂停状态徽标遮挡频道名称的问题

### 优化

- **能力测试响应长度** - 将测试响应 token 限制从 20 提升到 100，获得更完整的测试结果

## [v2.6.31] - 2026-03-11

### 修复

- **清理快速添加中的 pricing 页面路径** - 修复前端快速添加功能中 pricing 页面路径配置问题

## [v2.6.30] - 2026-03-09

### 修复

- **修复工具调用协议转换中的两个 P2 级别 Bug**
  - 修复空 tool input 生成非法 JSON 参数问题：`marshalJSONString` 对 `nil` 返回 `"{}"` 而非空字符串，避免上游函数调用请求解析失败
  - 修复 Gemini 同名并行函数调用产生重复 tool_call_id 问题：使用 `函数名_索引` 格式生成唯一 ID，确保 tool result 能正确关联到具体调用
- **Responses API 工具调用与工具定义互转补全** - 修复 Responses 协议在 Claude / OpenAI Chat / Gemini 之间转换时 function tools、function_call、function_call_output 字段丢失或格式不一致的问题
  - 为 `ResponsesRequest` 补充 `tools`、`tool_choice`、`parallel_tool_calls`、`max_output_tokens` 字段，并统一映射到各上游请求
  - 新增 `responses_tools.go` 提取工具定义转换逻辑，统一生成 Claude/OpenAI/Gemini 所需的函数工具格式
  - 修复 `parseResponsesInput` / `parseInputToItems` 对 `function_call` 与 `function_call_output` 的字段保留，避免多轮工具链路断裂
  - 修复 Gemini / OpenAI Chat / Claude 响应转 Responses 时 `call_id`、`name`、`arguments`、`output` 的结构化映射
  - 补充转换器与 handler 单测，覆盖工具定义透传、function_call/function_call_output 往返与 handler 解析场景

### 新增

- **协议互转工具调用测试覆盖** - 补充所有缺失的协议间 tool 互转测试，新增 3 个测试文件共 16 个测试用例
  - Gemini ↔ Claude 工具转换测试（`gemini_claude_tool_test.go`，6 个测试）
  - Responses ↔ OpenAI 工具转换测试（`responses_openai_tool_test.go`，5 个测试）
  - OpenAI ↔ Claude 工具转换测试（`openai_claude_tool_test.go`，5 个测试）
  - 覆盖 function_call/tool_use/tool_calls 的双向转换、多工具调用、混合内容、往返验证等场景

## [v2.6.29] - 2026-03-09

### 优化

- **模型列表更新** - 更新前端渠道添加弹窗中的默认模型列表，使用最新模型版本
  - Chat 渠道：更新为 codex、gpt-5.x 系列（gpt-5、gpt-5.4、gpt-5.3-codex、gpt-5.2-codex、gpt-5.2）
  - Gemini 渠道：调整模型顺序，优先展示 gemini-3.x 系列，精简预览版本列表
  - 目标模型占位符：统一更新为最新模型示例（gpt-5.4、gemini-3.1-pro、claude-opus-4-6）

## [v2.6.28] - 2026-03-08

### 新增

- **支持的模型常用过滤器** - 在"支持的模型"输入框下方新增常用过滤器胶囊（claude-*、gpt-5*、grok-4*、gemini-3*），所有渠道类型统一显示，点击自动追加，已选状态有视觉区分

### 优化

- **目标模型优先级排序** - 按模型系列重要性分桶排序（Claude Opus/Sonnet → GPT-5.x → Grok-4.x → Gemini-3 → GLM/Kimi 等），同桶内按版本号自然降序，未匹配模型兜底排序
- **模型重定向卡片布局调整** - Fast 模式开关和输出冗长度移至映射输入区域下方；模型映射添加行支持窄屏自动换行
- **模型重定向示例文案更新** - 更新各渠道类型的重定向示例和占位符，使用最新模型名称（如 gpt-5.4、claude-sonnet-4-5）

### 移除

- 移除模型重定向区域内重复的 Fast 模式说明文字

## [v2.6.27] - 2026-03-07

### 新增

- **能力测试结果缓存** - 后端新增内存缓存，避免短时间内重复执行耗时的能力测试请求
  - 初始 TTL 5 分钟，每次缓存命中自动续期 5 分钟，最大生存期 15 分钟
  - 缓存 Key 基于渠道类型、ID 和协议列表，并发安全（sync.RWMutex）
  - 仅缓存有成功结果的测试，惰性淘汰过期条目

### 优化

- **能力测试请求优化** - 降低测试请求的思考强度和超时时间，大幅缩短测试耗时
  - 默认超时从 15 秒降至 10 秒
  - Messages 协议明确关闭思考（`thinking.type: disabled`）
  - Chat 协议设置 `reasoning_effort: "none"`
  - Responses 协议设置 `reasoning.effort: "none"`
  - Gemini 协议设置 `thinkingLevel: "low"`
- **高级选项字段条件渲染** - 提取 `channelAdvancedOptions` 工具模块，统一管理渠道高级选项的支持判断与归一化
  - 不支持的渠道类型（claude/gemini）表单中隐藏高级选项输入控件，保存时自动清空对应字段
  - 新增完整单元测试覆盖
- **服务类型标签修正** - OpenAI → OpenAI Chat，Responses (原生接口) → Responses (Codex)

## [v2.6.26] - 2026-03-07

### 新增

- **渠道能力测试功能** - 新增渠道协议兼容性测试，支持并发测试 Messages/Chat/Gemini/Responses 四种协议，检测流式支持和延迟
  - 后端：新增 `/api/{type}/channels/{id}/capability-test` 端点，并发测试多协议兼容性，返回详细测试结果（成功/失败、延迟、流式支持、错误分类）
  - 多模型降级测试：每个协议支持多个候选模型（逗号分隔），按优先级依次尝试，一旦某个模型成功就停止，提高测试成功率
  - 前端：新增 `CapabilityTestDialog` 组件，展示测试结果和兼容协议列表，显示测试成功的模型名称，支持一键复制渠道到兼容的 Tab
  - 编辑渠道弹窗：右上角添加"能力测试"按钮，编辑时可直接测试渠道能力
  - 错误分类：支持 timeout、rate_limited、http_error_XXX 等错误类型，Tooltip 显示详细错误信息
  - 测试模型配置：
    - Messages: `claude-opus-4-6,claude-opus-4-5-20251101,claude-sonnet-4-6,claude-sonnet-4-5-20250929`
    - Chat: `gpt-5.4,gpt-5.3-codex,gpt-5.2`
    - Gemini: `gemini-3.1-pro-preview,gemini-3-pro-preview,gemini-3-pro,gemini-3-flash-preview,gemini-3-flash`
    - Responses: `gpt-5.4,gpt-5.3-codex,gpt-5.2`
  - 影响文件：
    - `backend-go/internal/handlers/capability_test_handler.go` - 新增能力测试处理器，支持多模型降级
    - `backend-go/internal/handlers/capability_probe_models.go` - 测试模型统一定义，支持多候选模型
    - `backend-go/main.go` - 注册能力测试路由
    - `frontend/src/components/CapabilityTestDialog.vue` - 新增测试结果对话框，显示测试模型
    - `frontend/src/components/AddChannelModal.vue` - 右上角添加测试按钮
    - `frontend/src/App.vue` - 测试流程和结果处理
    - `frontend/src/services/api.ts` - 能力测试 API 调用，新增 testedModel 字段

- **4 协议完整互转支持** - 实现 Claude Messages、OpenAI Chat、Gemini、Responses 四种协议的完整双向转换矩阵（12 条转换路径）
  - 前端：所有渠道类型（messages/chat/responses/gemini）现可选择全部 4 种上游服务类型（claude/openai/gemini/responses）
  - 后端转换器：新增 `GeminiResponsesConverter`、`gemini_to_responses.go`、`responses_to_gemini.go` 实现 Gemini ↔ Responses 双向转换
  - Gemini handler：添加 `responses` 上游支持（请求构建、认证、流式/非流式响应转换）
  - Responses handler：流式处理中按 `upstreamType` 分支，`gemini` 上游调用 `ConvertGeminiStreamToResponses`
  - 影响文件：
    - `frontend/src/components/AddChannelModal.vue` - serviceTypeOptions 扩展
    - `internal/converters/factory.go` - 注册 GeminiResponsesConverter
    - `internal/converters/gemini_responses_converter.go` - 新增
    - `internal/converters/gemini_to_responses.go` - 新增
    - `internal/converters/responses_to_gemini.go` - 新增
    - `internal/handlers/gemini/handler.go` - 添加 responses case
    - `internal/handlers/gemini/stream.go` - 添加 streamResponsesToGemini
    - `internal/handlers/responses/handler.go` - 流式转换分支
    - `internal/providers/responses.go` - Gemini URL 动态构建

### 修复

- **互转回归测试补强** - 新增 Gemini ↔ Responses 互转回归测试与前端 URL 预览组合测试，覆盖纯工具调用、多个 function call、usage 迟到扣减，以及 `responses + claude/openai/gemini`、`chat/gemini + responses` 等组合，降低四协议互转后续回归风险
- **Responses → Gemini 工具项读取错误（P1）** - `parseResponsesInput` 现在对 `function_call` 和 `function_call_output` 类型保留完整的 itemMap 作为 Content，`responsesItemToGeminiContents` 支持从顶层和嵌套 content 字段读取 name/arguments/call_id/output，确保工具调用链不会因字段丢失而断裂
- **call_id 与函数名映射不一致（P1）** - Gemini → Responses 转换（非流式和流式）现在使用函数名作为 `call_id`，Responses → Gemini 转换中 `function_call_output` 使用 `name` 字段（而非 `call_id`）作为 `FunctionResponse.Name`，确保工具结果可以稳定匹配 Gemini 函数调用
- **流式完成状态错误（P2）** - `generateCompletedEvent` 现在使用传入的 `finishReason` 调用 `geminiFinishReasonToResponsesStatus` 进行状态映射，不再硬编码 `"completed"`，正确处理 MAX_TOKENS/SAFETY 等场景
- **前端 URL 预览与后端能力不一致（P3）** - `responses` 渠道下选择 `serviceType=gemini` 时，URL 预览现在正确显示 `/models/{model}:generateContent` 端点，而非错误回退到 `/chat/completions`
- **Responses → Gemini 流式转换 SSE 解析** - `ConvertResponsesToGeminiStream` 现在正确处理逐行输入的 SSE 事件（`event:` 和 `data:` 分行），在状态中缓存 `eventType`，避免因事件类型缺失导致流式输出为空
- **Gemini 流式响应发送增量文本** - `response.output_text.delta` 处理中现在发送增量 delta 而非累计文本，避免客户端拼接时出现重复内容
- **Gemini URL 自动添加 v1beta 前缀** - Responses provider 构建 Gemini URL 时，当 `baseURL` 不含版本号后缀且未标记 `#` 跳过时，自动添加 `/v1beta` 前缀，避免 404 错误
- **Responses → Gemini 流式工具调用丢失** - 添加 `CurrentFuncName` 和 `CurrentFuncArgs` 状态字段，在 `response.output_item.added`、`response.function_call_arguments.delta`、`response.function_call_arguments.done` 事件中正确收集工具调用，确保最终 chunk 包含完整 function calls
- **Gemini → Responses 流式 usage 顺序错误** - 将 `usageMetadata` 处理移至 `generateCompletedEvent` 调用前，避免 usage 信息在 `response.completed` 事件中缺失
- **Gemini 流式最终 chunk 重复全文** - `buildGeminiFinalChunk` 不再包含文本内容（文本已通过 delta 发送），最终 chunk 仅包含 finishReason、usage 和 function calls，避免客户端末尾重复显示全文
- **前端 URL 预览 responses 服务类型错误** - 修复非 responses 渠道选择 `serviceType=responses` 时，URL 预览错误显示 `/chat/completions` 的问题，现正确显示 `/responses` 端点
- **Gemini → Responses 流式转换丢失工具调用（P1）** - 在 `geminiToResponsesStreamState` 中添加 `FunctionCalls` 字段，流式处理中检测 `functionCall` 并收集到状态，在 `generateCompletedEvent` 中输出为 `function_call` 类型的 output item
- **纯工具调用被误判为空响应（P1）** - 添加 `hasResponsesFunctionCall` 函数检测 `response.completed` 事件中的工具调用，预检逻辑现在识别纯工具调用（无文本）为有效响应，避免触发重试/切换渠道
- **call_id 不一致导致工具调用关联失败（P1）** - Gemini → Responses 转换中，`function_call` 和 `function_call_output` 现在都使用函数名作为 `call_id`，确保同一次工具调用的请求和响应可以稳定关联
- **前端允许但后端不支持的配置（P2）** - Messages 渠道现在不再显示 `responses` 上游选项，避免用户配置后端不支持的组合

## [v2.6.25] - 2026-03-03

### 修复

- **Gemini 流式响应 cachedContentTokenCount 处理** - 修复 Gemini 流式 usage 聚合逻辑在 `cachedContentTokenCount` 迟到时返回错误 input token 值的问题。当后续 chunk 包含缓存扣除信息时，`input_tokens` 现在会正确更新为扣除后的值（如 100 - 80 = 20），而非保持之前较大的值（100）
  - 影响文件：`providers/gemini.go`
  - 新增测试：`TestGeminiHandleStreamResponse_CachedContentTokenCountReducesInputTokens`

## [v2.6.24] - 2026-03-02

### 修复

- **Dashboard 响应包含完整渠道配置字段** - `GetChannelDashboard` 返回值新增 `customHeaders`、`proxyUrl`、`supportedModels` 字段，确保前端展示 Dashboard 时能获取完整渠道信息

## [v2.6.23] - 2026-03-02

### 修复

- **上游 401 错误误触发重新登录** - `GetChannelModels` 端点透传上游 401 状态码时，前端统一 401 处理逻辑会误判为管理 API 认证失败并清除登录状态。现将上游 401 包装为 400 BadRequest 返回，避免前端误判
  - 影响文件：`handlers/chat/channels.go`、`handlers/messages/channels.go`、`handlers/responses/channels.go`、`handlers/gemini/channels.go`
  - 同步更新 `messages/channels_test.go` 中 `TestGetChannelModels_UpstreamReturns401` 断言：期望由 401 改为 400

## [v2.6.22] - 2026-03-02

### 新增

- **渠道模型列表查询 API** - 添加 `POST /api/{type}/channels/:id/models` 端点（支持 messages/responses/chat/gemini 四种渠道类型），通过后端代理获取上游模型列表，解决前端直连上游时的 CORS 跨域问题和 API Key 泄露风险
  - 请求体 `{"key": "sk-xxx", "baseUrl": "https://..."}` - 单个 key，始终传递表单当前 baseUrl（确保检测反映最新配置，而非已保存的旧值）
  - 前端对每个 key 并发独立请求，各自维护独立状态（loading/success/fail），合并所有成功 key 的模型列表去重展示
  - `apiType` 与 `modelsApiType` 解耦：`apiType` 决定在哪个渠道组数组查找 id，`modelsApiType` 按 serviceType 决定请求协议（Bearer/x-goog-api-key、/v1/models vs /v1beta/models），支持 gemini 渠道组内配置 openai/claude serviceType 的渠道
  - Gemini 特殊处理：使用 `/v1beta/models` 端点和 `x-goog-api-key` 认证头；后端将 `{"models": [{"name": "models/gemini-..."}]}` 转换为 OpenAI 兼容格式 `{"object": "list", "data": [{"id": "gemini-..."}]}`，若响应无 `models` 字段则透传原始响应
  - 后端单次请求 10s 超时
  - 新增 `messages/GetChannelModels` 单元测试（6 个用例：非法 ID、空 key、渠道不存在、上游 200、上游 401、临时 baseUrl）

### 安全

- **SSRF 防护（云元数据）** - 模型列表查询 API 新增 `utils.ValidateBaseURL()` 验证，硬编码拦截云元数据服务（169.254.169.254），防止云凭证泄露。允许其他内网地址（支持 Ollama、内网部署等场景）

### 重构

- **统一 Dashboard 端点** - 将 4 种 dashboard 端点（`/api/messages/channels/dashboard`, `/api/chat/channels/dashboard`, `/api/gemini/channels/dashboard` 及 `?type=responses`）统一为 `/api/messages/channels/dashboard?type=messages|responses|chat|gemini`，消除代码重复，符合 DRY 原则
  - 后端：扩展 `GetChannelDashboard()` 支持 chat/gemini 类型，删除 `chat/dashboard.go` 和 `gemini/dashboard.go`
  - 前端：简化 API 服务调用，删除 `getChatChannelDashboard()` 和 `getGeminiChannelDashboard()`，统一使用 `getChannelDashboard(type)`

### 改进

- **模型重定向源模型名支持自由输入** - 将源模型名输入框从 `v-select`（仅选列表）改为 `v-combobox`（可选可输入），与目标模型名保持一致
- **源模型名输入验证实时反馈** - 新增 `validateSourceModelName` 函数，拦截非法输入：自定义模型名超过 50 字符（内置选项不受限）、含空格、含非法字符（仅允许字母、数字、`-_.:/@+`）；验证绑定到 `@update:model-value` 事件，输入时实时显示错误提示并禁用"添加"按钮

### 修复

- **前端代码质量** - 移除 `AddChannelModal.vue` 中未使用的 `apiType` 变量和无效的 ESLint 指令，消除 lint 警告

## [v2.6.21] - 2026-03-02

### 新增

- **Gemini 3 系列模型重定向支持** - 新增 Gemini 3 系列模型（gemini-3.0-flash、gemini-3.0-pro 等）的请求重定向，自动路由到正确的 Gemini API 端点
- **移动端导航下拉菜单** - 手机端（< 600px）将四个 API 类型链接收进下拉菜单，节省 header 空间；平板端隐藏 brand-text 避免溢出

### 修复

- **登录界面密钥提示更新** - 输入框 label 改为「管理访问密钥」，安全提示说明优先使用 `ADMIN_ACCESS_KEY`、未设置时回退到 `PROXY_ACCESS_KEY`；移除安全提示 info 图标节省空间
- **CCH 按钮仅在 Claude 渠道显示** - CCH 计费头移除开关仅在 Messages 渠道页面显示，其他类型渠道不再展示无关按钮

## [v2.6.20] - 2026-03-01

### 优化

- **Dashboard 接口响应体积优化** - 针对 `/api/messages/channels/dashboard` 每 2 秒刷新时大量 0 值内容的问题进行优化
  - 后端：`ChannelRecentActivity.Segments` 改为稀疏 Map 格式 (`map[int]*ActivitySegment`)，只返回有请求的段，无请求时返回空 map 而非 150 个空对象
  - 后端：`ActivitySegment`、`MetricsResponse`、`KeyMetricsResponse`、`TimeWindowStats` 结构体添加 `omitempty` 标签，0 值字段不再输出到 JSON
  - 前端：新增 `expandSparseSegments()` 辅助函数，自动将稀疏 Map 展开为完整数组，兼容旧版数组格式
  - 效果：无请求时单个渠道的 `recentActivity` 响应体积从 ~7KB 减少到 ~50 bytes

- **移动端加载性能优化** - 针对手机端打开慢的问题进行多维度优化
  - 后端：新增 Gzip 中间件 (`backend-go/internal/middleware/gzip.go`)，采用白名单模式仅压缩静态资源（`/assets/`、`.js`、`.css` 等），避免意外压缩未来新增的流式端点
  - 前端：Vuetify 组件按需导入，首屏 JS 从 1,374 KB 降至 506 KB (-63%)
  - 前端：图表组件 (`GlobalStatsChart`、`KeyTrendChart`) 改为异步加载，ApexCharts 库 (582 KB) 延后到用户展开图表时才加载
  - 前端：优化分包策略，vue-vendor / vuetify / charts 独立分包
  - 依赖：升级 gin-gonic/gin v1.11.0，新增 gin-contrib/gzip v1.2.5

## [v2.6.19] - 2026-02-28

### 修复

- **渠道最后成功/失败时间戳持久化** - 修复渠道超过 24 小时无请求后重启时"最后成功"/"最后失败"时间戳丢失的问题
  - `persistence.go`：新增 `KeyLatestTimestamps` 结构体，`PersistenceStore` 接口增加 `LoadLatestTimestamps(apiType)` 方法
  - `sqlite_store.go`：实现 `LoadLatestTimestamps`，用单条 `GROUP BY + MAX(CASE WHEN)` SQL 从全量历史记录中查出每个 key 的最后成功/失败时间
  - `channel_metrics.go`：`loadFromStore` 末尾调用 `loadHistoricalTimestamps()` 补全超出 24h 窗口的时间戳；修复 24h 内无任何记录时提前返回导致历史时间戳不被加载的 bug；对 24h 内无记录但历史有请求的渠道创建空壳 `KeyMetrics` 携带时间戳（`RequestCount=0`，不影响统计）

## [v2.6.18] - 2026-02-26

### 移除

- **清理 LoadBalance 死代码** - 调度器完全基于优先级/促销/Trace亲和/健康状态选择渠道，`LoadBalance` 字段从未被读取
  - 后端：删除 `Config` 结构体中 `LoadBalance`/`ResponsesLoadBalance`/`GeminiLoadBalance`/`ChatLoadBalance` 四个字段，删除 `Set*LoadBalance` 方法、`validateLoadBalanceStrategy` 验证函数、四个 `UpdateLoadBalance` Handler、两条 API 路由，清理 Dashboard/Health/Metrics 响应中的 loadBalance 返回
  - 前端：删除 `ChannelsResponse`/`ChannelDashboardResponse` 接口中的 `loadBalance` 字段，删除四个 `update*LoadBalance` API 方法，清理 store 中所有 loadBalance 状态初始化、赋值和重置逻辑

## [v2.6.17] - 2026-02-25

### 新增

- **渠道模型白名单过滤** - 为渠道配置支持的模型列表，调度器自动跳过不支持当前请求模型的渠道
  - 后端：`UpstreamConfig` 新增 `SupportedModels` 字段，`SupportsModel()` 支持精确匹配和通配符前缀匹配（如 `gpt-4*`），`SelectChannel` / `getActiveChannels` 按模型过滤渠道，四类 Handler 调用链传入 model 参数
  - 前端：渠道编辑表单新增"支持的模型"Combobox（Chips 输入），`Channel` 接口支持 `supportedModels` 字段
  - 空列表表示支持所有模型，向后兼容
- **渠道级代理（Proxy）支持** - 为每个渠道配置独立的 HTTP/SOCKS5 代理，用于通过代理访问特定上游服务（网络隔离、地域限制等场景）
  - 后端：`UpstreamConfig` 新增 `ProxyURL` 字段，`GetStandardClient`/`GetStreamClient` 支持代理配置，`SendRequest` 传递代理参数并记录脱敏日志
  - 前端：渠道编辑表单新增代理 URL 输入框，`Channel` 接口支持 `proxyUrl` 字段
  - Ping 适配：Messages 和 Gemini 渠道连通性测试均通过渠道代理发送

### 修复

- **Chat 渠道 Ping 对 Claude serviceType 的支持** - Claude API 没有 `/v1/models` 端点，改用 `OPTIONS /v1/messages` 进行健康检查
- **Chat Claude 响应 finish_reason 映射** - 正确映射 Claude `stop_reason` 到 OpenAI `finish_reason`（`max_tokens`→`length`, `tool_use`→`tool_calls`）

### 文档

- **新增 OpenAI Chat Completions 端点设计文档** (`docs/chat-completions-design.md`) - 详细设计第四类用户侧 API (`POST /v1/chat/completions`)，涵盖后端 Config/CRUD/Handler/Scheduler/Metrics 扩展、协议转换器（OpenAI Chat ↔ Claude/Gemini）、FailedKeysCache 和 TraceAffinity 按 apiType 隔离、前端 Chat Tab 集成等完整方案
- **移除 Credential Pool 设计文档** (`docs/credential-pool-design.md`) - 密钥池方案暂缓，后续视 Chat Completions 端点落地情况再决定是否推进

### 变更

- **Go 依赖升级** - `golang.org/x/net` 升级到 v0.50.0（直接依赖），支持 SOCKS5 代理功能

---

# 版本历史

> **注意**: v2.0.0 开始为 Go 语言重写版本，v1.x 为 TypeScript 版本

---

## [v2.6.16] - 2026-02-18

### 新增

- **渠道日志接口类型标识** - `ChannelLog` 新增 `interfaceType` 字段，记录请求来源接口类型（Messages/Responses/Gemini），前端日志列表以彩色标签展示

---

## [v2.6.15] - 2026-02-14

### 修复

- **Messages API 空响应误判 tool_use/thinking 响应** - `PreflightStreamEvents` 预检测仅通过 `delta.text` 判断内容是否为空，导致纯 tool_use（工具调用）、thinking（思考）、server_tool_use 等非文本 content block 响应被误判为空响应并触发不必要的重试；新增 `hasNonTextContentBlock` 检测，遇到非文本 content block 时立即放行

---

## [v2.6.14] - 2026-02-14

### 新增

- **自动移除 cch= 计费头参数** - Messages API 预处理阶段自动剥离 system 数组中的 `cch=xxx;` 参数，保留 `cc_version`/`cc_entrypoint` 等其他计费信息，避免上游计费混乱
- **前端 CCH 全局开关** - 操作栏新增 CCH 切换按钮，支持热配置切换（默认启用）

---

## [v2.6.13] - 2026-02-13

### 新增

- **全局流量图按模型堆叠面积曲线** - 流量模式改为按实际模型（如 claude-opus-4-6、claude-haiku-4-5）显示堆叠面积曲线，tooltip 显示各模型请求数/失败数，移除冗余的 ModelStatsChart 组件
- **渠道流量图改为堆叠面积图** - Key+Model 分组方式不变，流量模式改为堆叠面积并显示 legend

### 修复

- **渠道 Token/Cache 图多模型时无数据** - 按模型拆分时 Token/Cache 字段未传递，补齐 KeyModelHistoryDataPoint 的 Token 数据
- **渠道 Token/Cache 图多模型 Y 轴刻度不一致** - 所有 Input series 共享左侧 Y 轴，所有 Output series 共享右侧 Y 轴

---

## [v2.6.12] - 2026-02-12

### 新增

- **渠道自定义请求头支持** - 允许为每个渠道配置自定义 HTTP 请求头，在发送请求到上游时附加或覆盖（关闭 [#4](https://github.com/BenedictKing/ccx/issues/4)）

---

## [v2.6.11] - 2026-02-12

### 改进

- **渠道流量图按 Key+Model 组合显示多条曲线** - 当同一 Key 请求多个模型时，每个 Key+Model 组合显示独立曲线，便于分析不同模型的流量分布
- **全局流量图简化为单曲线** - 流量模式只显示请求总量曲线，失败率通过红色背景色带表示（与渠道流量图风格统一）

---

## [v2.6.10] - 2026-02-12

### 改进

- **日志和流量图显示重定向后的实际模型** - 当渠道配置了模型映射时，日志显示 `原始模型 → 实际模型`，流量图按实际使用的模型分组统计

---

## [v2.6.7] - 2026-02-11

### 修复

- **渠道流量条历史最大值增加指数衰减** - 避免历史峰值过后所有柱子都变得很矮，半衰期 5 分钟

### 改进

- **version-bump 技能增加 CHANGELOG 前置检查** - 检查 `[Unreleased]` 区块是否存在及是否包含实际变更内容，无区块时警告并询问是否跳过，有区块但无内容时中止流程，避免发布空版本记录

---

## [v2.6.6] - 2026-02-10

### 新增

- **模型维度使用统计** - 新增按模型分组的请求量和 Token 消耗时间序列统计
  - `RequestRecord` 和 `PersistentRecord` 新增 `Model` 字段，记录每次请求的模型名
  - SQLite schema migration（user_version 0→1）自动添加 `model` 列和索引
  - 新增 `GET /api/{messages|responses|gemini}/models/stats/history` API，返回按模型分组的历史数据点
  - 前端新增 `ModelStatsChart.vue` 多曲线面积图组件，支持请求量/Token 双视图切换
  - 集成到 `App.vue` 全局统计区域，与 GlobalStatsChart 并列展示

- **渠道快速日志** - 新增渠道级别的请求日志查看功能（内存环形缓冲区，每渠道保留最近 50 条）
  - 新增 `ChannelLogStore` 内存环形缓冲区，按 channelIndex 存储，纯内存，重启丢失
  - `TryUpstreamWithAllKeys` 在每次上游尝试后自动采集日志（含状态码、耗时、错误摘要、是否重试）
  - 新增 `GET /api/{messages|responses|gemini}/channels/:id/logs` API
  - 前端新增 `ChannelLogsDialog.vue` 弹窗组件，支持状态码颜色标识、展开错误详情、3 秒自动刷新
  - 渠道操作菜单新增"日志"入口（mdi-history 图标）
  - 渠道卡片新增独立日志按钮，与三点菜单并列，方便快速访问

### 修复

- **ModelStatsChart 并发请求竞态** - 使用请求版本号机制，确保只有最新请求的结果会更新数据，旧请求返回时自动丢弃，避免数据闪回
- **渠道删除后日志串台** - `ChannelLogStore` 新增 `ClearAll` 方法，删除渠道时清空整个日志存储避免索引错位；`DeleteUpstream` handler 传入 `channelScheduler` 参数

---

## [v2.6.5] - 2026-02-08

### 修复

- **OpenAI/Gemini 上游流式响应缺少 `message_start`/`message_stop` 事件** - 修复通过 OpenAI/Gemini 上游代理时，Anthropic SDK 报 `Unexpected event order, got content_block_start before "message_start"` 的问题
  - OpenAI/Gemini provider 的 `HandleStreamResponse` 在协议转换时未生成 Claude Messages API 规范要求的 `message_start` 和 `message_stop` 事件
  - 新增 `buildMessageStartEvent()` 公共函数，在第一个 `content_block_start` 之前自动发送 `message_start`
  - 流结束时统一发送 `message_delta`（含 `stop_reason`）+ `message_stop`
  - 修复 OpenAI provider `finish_reason == "stop"` 时未发送 `message_delta` 的遗漏
  - 新增 `finish_reason == "length"` → `max_tokens` 的映射
  - 涉及文件：`backend-go/internal/providers/openai.go`, `backend-go/internal/providers/gemini.go`

---

## [v2.6.4] - 2026-02-07

### 新增

- **上游空响应自动重试** - 上游返回 HTTP 200 但流式响应内容为空或几乎为空时，自动触发 failover 重试
  - 空响应定义：OutputTokens == 0（完全无输出）或 OutputTokens == 1 且内容仅为 `{`（截断的 JSON 开头）
  - 智能预检测：在发送 HTTP 200 Header 之前缓冲上游事件并检查实际输出内容
  - Messages API：新增 `PreflightStreamEvents()` 预检测函数，延迟 `SetupStreamHeaders()` 调用
  - Responses API：在 `handleStreamSuccess()` 中新增 scanner 预读取逻辑
  - Failover 集成：空响应触发 `ErrEmptyStreamResponse`，标记 Key 失败并计入熔断指标，继续尝试下一个 Key/BaseURL/渠道
  - 预检测超时 30s 保守放行，正常响应延迟约 100-200ms（等到第一个有效 content_block_delta）
  - 涉及文件：`backend-go/internal/handlers/common/stream.go`, `backend-go/internal/handlers/common/upstream_failover.go`, `backend-go/internal/handlers/responses/handler.go`

---

## [v2.6.3] - 2025-02-05

### 变更

- **渠道删除时保留历史指标数据** - 删除渠道时不再主动清理指标数据，让数据自然过期
  - 移除三个渠道删除处理器中的 `DeleteChannelMetrics()` 调用
  - SQLite 数据将在配置的保留期后自动删除（`METRICS_RETENTION_DAYS`，默认 30 天）
  - 内存指标将在 48 小时无活动后自动清理
  - 保持全局历史统计数据完整性，不再因删除渠道而丢失
  - **注意**：若用相同 BaseURL + APIKey 重建渠道，可能继承近期健康状态/统计（受内存清理窗口与服务重启影响，熔断状态不持久化）
  - 涉及文件：`backend-go/internal/handlers/messages/channels.go`, `backend-go/internal/handlers/responses/channels.go`, `backend-go/internal/handlers/gemini/channels.go`

---

## [v2.6.2] - 2026-02-04

### 新增

- **渠道配置复制功能** - 在渠道右侧弹出菜单中新增"复制配置"选项
  - 点击后将渠道的所有 BaseURL 和 API Key 按行分隔复制到系统剪贴板
  - 方便用户分享配置或快速创建新渠道
  - 支持活跃渠道和备用池渠道
  - 涉及文件：`frontend/src/components/ChannelOrchestration.vue`

- **DeleteChannelMetrics 测试覆盖** - 新增共享 MetricsKey 删除场景的单元测试
  - `TestDeleteChannelMetrics_SharedMetricsKeyPreserved` - 验证共享 metricsKey 被保留
  - `TestDeleteChannelMetrics_AllExclusiveKeysDeleted` - 验证独占 metricsKey 全部删除
  - `TestDeleteChannelMetrics_PreconditionWarning` - 验证前置条件违反时的行为
  - 涉及文件：`backend-go/internal/scheduler/channel_scheduler_test.go`

### 修复

- **删除渠道时共享 MetricsKey 数据丢失** - 修复删除渠道时误删其他渠道共享指标数据的问题
  - 问题：当两个渠道使用相同的 (BaseURL, APIKey) 组合时，删除其中一个渠道会导致另一个渠道的统计数据也被清除
  - 原因：Metrics 按 `hash(baseURL + apiKey)` 存储，删除时直接删除 MetricsKey，未检查是否有其他渠道共享
  - 修复：在 `DeleteChannelMetrics()` 中增加共享检测逻辑，只删除不被其他渠道使用的独占 MetricsKey
  - 新增 `collectUsedCombinations()` 辅助方法收集其他渠道的组合
  - 新增 `isUpstreamInConfig()` 前置条件守卫，检测渠道是否仍在配置中
  - 涉及文件：`backend-go/internal/scheduler/channel_scheduler.go`

- **DeleteByMetricsKeys 返回值语义不清晰** - 补充方法注释说明返回值语义
  - 返回持久化存储删除的记录数，未配置存储或删除失败时返回 0
  - 涉及文件：`backend-go/internal/metrics/channel_metrics.go`

- **前端复制配置 timeout 未清理** - 修复组件卸载时 `copyTimeoutId` 未清理的问题
  - 在 `onUnmounted` 钩子中添加 `clearTimeout(copyTimeoutId)` 清理逻辑
  - 涉及文件：`frontend/src/components/ChannelOrchestration.vue`

---

## [v2.6.1] - 2026-02-01

### 修复

- **低质量渠道 message_start 事件 usage 修补** - 修复低质量渠道模式下 `message_start` 事件中虚假 `input_tokens` 未被修补的问题
  - 问题：当 `lowQuality=true` 且 `input_tokens >= 10` 时，`PatchMessageStartInputTokensIfNeeded` 函数会跳过修补，导致虚假值（如 25599）被直接返回
  - 修复：在条件判断中增加 `!lowQuality` 检查，确保低质量渠道始终调用 `PatchTokensInEvent` 进行 5% 偏差检测
  - 涉及文件：`backend-go/internal/handlers/common/stream.go`

- **message_start 事件 output_tokens 误修补** - 修复 `output_tokens=1` 被错误修补为 `0` 的问题
  - 问题：`patchUsageFieldsWithLog` 的常规修补逻辑在 `estimatedOutput=0` 时仍会将 `output_tokens=1` 修补为 `0`
  - 修复：在常规 output_tokens 修补条件中增加 `estimatedOutput > 0` 检查，避免用无效估算值覆盖正常的初始值
  - 涉及文件：`backend-go/internal/handlers/common/stream.go`

---

## [v2.5.13] - 2026-01-31

### 修复

- **Gemini functionDeclaration parameters 类型修复** - 修复 Gemini API 返回 400 错误的问题
  - 问题：当 Claude 工具的 `InputSchema` 为 nil、缺少 `type` 字段或缺少 `properties` 字段时，Gemini API 拒绝请求
  - 新增 `normalizeGeminiParameters()` 辅助函数，确保 parameters schema 符合 Gemini 要求：
    - `parameters` 必须有 `type: "object"` 字段
    - `parameters` 必须有 `properties` 字段（即使为空对象）
  - 涉及文件：`backend-go/internal/providers/gemini.go`

---

## [v2.5.12] - 2026-01-30

### 新增

- **渠道置顶/置底功能** - 在渠道编排菜单中新增一键调整渠道位置的操作
  - 在渠道右侧弹出菜单中添加"置顶"和"置底"选项
  - 第一个渠道不显示"置顶"，最后一个渠道不显示"置底"
  - 操作后立即保存到后端，复用现有 `saveOrder()` 函数
  - 解决渠道数量较多时拖拽排序不便的问题
  - 涉及文件：
    - `frontend/src/components/ChannelOrchestration.vue` - 添加菜单项和处理函数
    - `frontend/src/plugins/vuetify.ts` - 添加 `arrow-collapse-up/down` 图标

- **隐式缓存读取推断** - 当上游未明确返回 `cache_read_input_tokens` 但存在显著 token 差异时，自动推断缓存命中
  - 检测 `message_start` 与 `message_delta` 事件中 `input_tokens` 的差异
  - 触发条件：差额 > 10% 或差额 > 10000 tokens
  - 将差额自动填充到 `CacheReadInputTokens` 字段，使 token 统计更准确
  - **下游转发支持**：推断的 `cache_read_input_tokens` 会写入 `message_delta` 事件并转发给下游客户端
  - 新增 `StreamContext.MessageStartInputTokens` 字段记录初始 token 数
  - 新增 `inferImplicitCacheRead()` 函数在流结束时执行推断
  - 新增 `PatchTokensInEventWithCache()` 函数在修补 token 的同时写入推断的缓存值
  - **关键修复**：
    - `message_start` 的 `input_tokens` 不再累积到 `CollectedUsage.InputTokens`，确保差额计算正确
    - 使用 `originalUsageData` 传递给 `PatchMessageStartInputTokensIfNeeded`，避免误判
    - Token 修补逻辑增加隐式缓存信号检测，避免覆盖缓存命中场景下的正确低值
    - 隐式缓存推断在转发前执行，确保下游客户端能收到推断值
    - 仅当上游事件中不存在 `cache_read_input_tokens` 字段时才写入推断值，避免覆盖上游显式返回的 0 值
  - 涉及文件：
    - `backend-go/internal/handlers/common/stream.go` - 核心逻辑实现
    - `backend-go/internal/handlers/common/stream_test.go` - 单元测试（15 个边界场景）

---

## [v2.5.10] - 2026-01-26

### 新增

- **删除渠道时自动清理指标数据** - 修复删除渠道后内存和 SQLite 指标数据残留问题
  - 扩展 `PersistenceStore` 接口，新增按 `metrics_key` 和 `api_type` 批量删除记录的方法
  - 新增 `MetricsManager.DeleteChannelMetrics()` 方法，支持同时清理内存和持久化数据
  - 新增 `ChannelScheduler.DeleteChannelMetrics()` 统一删除入口
  - 修改 `DeleteUpstream` Handler（Messages/Responses/Gemini），删除后自动调用指标清理
  - SQLite 清理不依赖内存状态，确保即使内存中无数据也能正确清理持久化记录
  - 删除渠道时同时清理历史 Key 的指标数据
  - **按 `api_type` 过滤删除**：避免误删其他接口类型（messages/responses/gemini）的指标数据
  - **分批删除**：每批 500 条，避免触发 SQLite 变量上限（999）导致删除失败
  - **并发安全**：`flushMu` 互斥锁串行化 flush 与 delete；`asyncFlushWg` 确保 Close 前所有异步 flush 完成
  - 涉及文件：
    - `backend-go/internal/metrics/persistence.go` - 接口扩展（新增 apiType 参数）
    - `backend-go/internal/metrics/sqlite_store.go` - 实现 SQLite 删除逻辑（分批 + api_type 过滤）
    - `backend-go/internal/metrics/channel_metrics.go` - 新增删除方法，导出 `GenerateMetricsKey()`
    - `backend-go/internal/scheduler/channel_scheduler.go` - 新增统一删除入口
    - `backend-go/internal/handlers/*/channels.go` - 删除 Handler 改造
    - `backend-go/main.go` - 路由注册更新

- **换 Key 后历史数据累计统计** - 修复更换 API Key 后旧 Key 的历史统计数据丢失问题
  - 新增 `UpstreamConfig.HistoricalAPIKeys` 字段，存储历史 API Key 列表
  - 更新渠道时自动维护历史 Key 列表：被移除的 Key 进入历史列表，恢复的 Key 从历史列表移除
  - `Add*APIKey` / `Remove*APIKey` 接口同样维护历史 Key 列表
  - `ToResponseMultiURL()` 支持聚合历史 Key 指标（只计入总数，不影响实时失败率和熔断判断）
  - 前端查看渠道统计时，总数包含历史 Key 数据，Key 详情列表只显示当前活跃 Key
  - 涉及文件：
    - `backend-go/internal/config/config.go` - 新增 `HistoricalAPIKeys` 字段
    - `backend-go/internal/config/config_utils.go` - `Clone()` 方法深拷贝历史 Key
    - `backend-go/internal/config/config_*.go` - 更新渠道时维护历史 Key 列表
    - `backend-go/internal/metrics/channel_metrics.go` - 聚合逻辑支持历史 Key
    - `backend-go/internal/handlers/channel_metrics_handler.go` - 传入历史 Key 参数
    - `backend-go/internal/handlers/gemini/dashboard.go` - 传入历史 Key 参数

---

## [v2.5.9] - 2026-01-24

### 新增

- **前端模型映射智能选择功能** - 优化模型重定向配置体验，支持自动获取上游模型列表
  - 前端直连上游 `/v1/models` 接口，无需后端代理
  - 目标模型输入框改为 `v-combobox`，点击时自动获取模型列表
  - 为每个 API Key 并行检测 models 接口状态，提高效率
  - 在 API 密钥列表中实时显示状态标签：
    - 成功：绿色标签显示 `models 200 (N 个)`
    - 失败：红色标签显示 `models 错误码`，鼠标悬停显示详细错误消息
    - 加载中：蓝色标签显示 `检测中...`
  - 智能错误解析，支持上游标准错误格式 `{ "error": { "message": "...", "code": "..." } }`
  - 合并所有成功的模型列表并去重，提供完整的模型选项
  - 涉及文件：
    - `frontend/src/services/api.ts` - 新增 `fetchUpstreamModels` 函数和 `buildModelsURL` 工具函数
    - `frontend/src/components/AddChannelModal.vue` - 优化交互体验和状态管理

---

## [v2.5.8] - 2026-01-21

### 修复

- **客户端取消请求误计入失败** - 修复用户主动取消请求被错误计入渠道失败指标的问题
  - 新增 `isClientSideError` 函数，使用 `errors.Is` 正确识别被包装的 `context.Canceled` 错误
  - 仅识别明确的客户端取消（`context.Canceled`），连接故障（`broken pipe`、`connection reset`）继续 failover
  - 统一口径：`SendRequest` 和 `handleSuccess` 路径均应用客户端取消判断
  - 新增 `RecordRequestFinalizeClientCancel` 方法，客户端取消时仅计入总请求数，不计入失败数和失败率
  - 客户端取消不重置 `ConsecutiveFailures`，保留真实的连续失败计数
  - 涉及文件：
    - `backend-go/internal/handlers/common/upstream_failover.go` - 错误类型判断与分流
    - `backend-go/internal/metrics/channel_metrics.go` - 新增客户端取消记录方法
    - `backend-go/internal/handlers/common/client_error_test.go` - 单元测试

- **指标二次计数 Bug** - 修复 `RecordRequestFinalize*` fallback 路径导致的请求计数重复问题
  - 将 `RequestCount++` 从 `RecordRequestConnected` 移至 `RecordRequestFinalize*` 阶段
  - 采用延迟计数策略：连接时预写历史记录，完成时统一计数
  - 确保 fallback 路径（requestID 丢失/索引越界）不会触发二次计数
  - 涉及文件：`backend-go/internal/metrics/channel_metrics.go`

### 重构

- **指标记录架构优化** - 将指标记录职责从 handler 层下沉到 failover 层，实现"连接即计数"的实时统计
  - 新增 `RecordRequestConnected` / `RecordRequestFinalizeSuccess` / `RecordRequestFinalizeFailure` 三阶段记录机制
  - TCP 建连时即计入活跃请求数，响应完成后回写成功/失败与 token 数据
  - 移除 handler 层的 `RecordSuccessWithUsage` / `RecordFailure` 调用，统一由 `upstream_failover.go` 管理
  - 修改 `HandleSuccessFunc` 签名：返回 `(*types.Usage, error)` 而非 `*types.Usage`，支持流式响应错误处理
  - 修改 `ProcessStreamEvents` / `HandleStreamResponse` 返回 usage，避免在 stream 层直接记录指标
  - 新增 `pendingHistoryIdx` 映射表，支持请求 ID 到历史记录索引的快速查找
  - 新增 `cleanupHistoryLocked` 函数，清理过期历史记录时同步修正索引
  - 涉及文件：
    - `backend-go/internal/handlers/common/stream.go` - 移除指标记录，返回 usage
    - `backend-go/internal/handlers/common/upstream_failover.go` - 三阶段指标记录
    - `backend-go/internal/handlers/messages/handler.go` - 移除指标记录调用
    - `backend-go/internal/handlers/responses/handler.go` - 移除指标记录调用
    - `backend-go/internal/handlers/gemini/handler.go` - 移除指标记录调用
    - `backend-go/internal/metrics/channel_metrics.go` - 新增三阶段记录 API

## [v2.5.6] - 2026-01-20

### 修复

- **Gemini CLI 工具调用签名兼容** - 修复多轮工具调用中签名字段位置/命名不一致导致上游返回 400 的问题（启用 `injectDummyThoughtSignature` 时会为缺失签名的 `functionCall` 注入 dummy）。
- **Gemini CLI tools schema 兼容** - 支持 `parametersJsonSchema` 并在转发前清洗不兼容字段（`$schema` / `additionalProperties` / `const`），避免上游 400。
- **Gemini Dashboard stripThoughtSignature 字段缺失** - Dashboard API 补齐 `stripThoughtSignature` 字段，避免配置在刷新后丢失。

- **Gemini 渠道 stripThoughtSignature 字段无法保存** - 修复前端无法正确显示和保存"移除 Thought Signature"配置的问题
  - 修复 `GetUpstreams` 函数返回数据中缺失 `stripThoughtSignature` 字段
  - 修复前端图标显示问题（将 `mdi-signature-freehand` 改为 `mdi-close-circle`）
  - 统一图标和开关颜色为 `error` 红色，与"移除"操作语义一致
  - 涉及文件：
    - `backend-go/internal/handlers/gemini/channels.go` - 添加缺失字段
    - `frontend/src/components/AddChannelModal.vue` - 修复图标和颜色

### 新增

- **Gemini API thought_signature 兼容性方案** - 新增 `stripThoughtSignature` 配置项，支持兼容旧版 Gemini API
  - 新增 `StripThoughtSignature` 配置字段（布尔值），用于移除 `thought_signature` 字段
  - 实现 `stripThoughtSignatures()` 函数，移除所有 functionCall 的 thought_signature 字段
  - 配置优先级：`StripThoughtSignature` > `InjectDummyThoughtSignature`
  - 保持深拷贝机制，避免多渠道 failover 时污染后续请求
  - 前端添加"移除 Thought Signature"开关（仅 Gemini 渠道显示）
  - 涉及文件：
    - `backend-go/internal/config/config.go` - 配置结构定义
    - `backend-go/internal/config/config_gemini.go` - 配置更新逻辑
    - `backend-go/internal/handlers/gemini/handler.go` - 请求处理逻辑
    - `backend-go/internal/handlers/gemini/handler_test.go` - 单元测试
    - `frontend/src/components/AddChannelModal.vue` - 前端开关
    - `frontend/src/services/api.ts` - 类型定义

## [v2.5.5] - 2026-01-19

## [v2.5.4] - 2026-01-19

### 重构

- **Failover 逻辑模块化** - 将多渠道和单上游 failover 逻辑提取到公共模块，大幅减少代码重复
  - 新增 `backend-go/internal/handlers/common/multi_channel_failover.go` - 多渠道 failover 外壳逻辑
  - 新增 `backend-go/internal/handlers/common/upstream_failover.go` - 单上游 Key/BaseURL 轮转逻辑
  - 重构 Messages、Responses、Gemini 三个 handler，使用统一的 failover 函数
  - 代码行数减少：-1253 行，+475 行（净减少 778 行）
  - 涉及文件：
    - `backend-go/internal/handlers/messages/handler.go`
    - `backend-go/internal/handlers/responses/handler.go`
    - `backend-go/internal/handlers/gemini/handler.go`
    - `backend-go/internal/scheduler/channel_scheduler.go`

## [v2.5.3] - 2026-01-19

### 修复

- **Models API 日志标签修正** - 修正 Models API 相关日志标签，确保正确区分 Messages 和 Responses 渠道
  - 修正 `models.go` 中 `tryModelsRequest` 和 `fetchModelsFromChannel` 函数的日志标签
  - 使用动态 `channelType` 变量替代硬编码的 `"Messages"` 字符串
  - 日志标签格式统一为 `[Messages-Models]` 或 `[Responses-Models]`
  - 涉及文件：`backend-go/internal/handlers/messages/models.go`
- **多渠道 failover 客户端取消检测** - 在 failover 循环中添加客户端断开检测，避免客户端已取消请求后继续尝试其他渠道
  - 在每次渠道选择前检查 `c.Request.Context().Done()`
  - 客户端断开时立即返回，不再进行无效的渠道 failover
  - 涉及文件：
    - `backend-go/internal/handlers/gemini/handler.go` - Gemini API 处理器
    - `backend-go/internal/handlers/messages/handler.go` - Messages API 处理器
    - `backend-go/internal/handlers/responses/handler.go` - Responses API 处理器

### 新增

- **响应 model 字段改写可配置化** - 新增环境变量 `REWRITE_RESPONSE_MODEL` 控制是否改写响应中的 model 字段
  - 默认值：`false`（保持上游返回的原始 model）
  - 启用后：当上游返回的 model 与请求的 model 不一致时，自动改写为请求的 model
  - 适用范围：仅影响 Messages API 的流式响应，不影响 Responses API 和 Gemini API
  - 涉及文件：
    - `backend-go/.env.example` - 添加配置说明和默认值
    - `backend-go/internal/config/env.go` - 添加 `RewriteResponseModel` 配置字段
    - `backend-go/internal/handlers/common/stream.go` - 修改 `PatchMessageStartEvent` 函数，仅在配置启用时改写 model 字段

## [v2.5.2] - 2026-01-19

### 新增

- **Gemini thought_signature 可配置化** - 新增渠道级配置开关 `injectDummyThoughtSignature`
  - 新增 `ensureThoughtSignatures` 函数：为所有缺失 `thought_signature` 的 `functionCall` 注入 dummy 值
  - 使用官方推荐的 `skip_thought_signature_validator` 跳过验证
  - **默认关闭**：保持原样，符合官方 Gemini API 标准
  - **用户可开启**：为需要该字段的第三方 API 注入 dummy signature
  - 前端 UI：在 Gemini 渠道编辑界面添加"注入 Dummy Thought Signature"开关
  - 涉及文件：
    - `backend-go/internal/config/config.go` - 添加 `InjectDummyThoughtSignature` 配置字段
    - `backend-go/internal/config/config_gemini.go` - 更新方法支持新字段
    - `backend-go/internal/config/config_messages.go` - 更新方法支持新字段
    - `backend-go/internal/handlers/gemini/handler.go` - 根据配置决定是否调用 `ensureThoughtSignatures`
    - `backend-go/internal/types/gemini.go` - 新增共享常量 `DummyThoughtSignature`
    - `backend-go/internal/converters/gemini_converter.go` - 使用共享常量
    - `frontend/src/services/api.ts` - 添加类型定义
    - `frontend/src/components/AddChannelModal.vue` - 添加配置开关 UI
    - `frontend/src/plugins/vuetify.ts` - 添加 `mdi-signature` 图标映射
  - 配置优化：将 `.ccb_config/` 目录加入 `.gitignore`，避免泄露本机路径等敏感信息

- **codex-review 技能 v2.1.0** - 新增自动暂存新增文件功能，避免 codex 审核时报 P1 错误
  - 新增步骤 2：在审核前自动暂存所有新增文件
  - 使用安全的 `git ls-files -z | while read` 命令，正确处理特殊文件名（空格、换行、以 `-` 开头）
  - 修复空列表问题：当没有新增文件时安全跳过，不会报错
  - 优化元数据：添加 `user-invocable: true` 和 `context: fork` 字段
  - 优化描述：添加触发关键词，移除 `(user)` 后缀
  - 更新完整审核协议：增加 `[PREPARE] Stage Untracked Files` 步骤
  - 创建 Plugin Marketplace 配置：`.claude-plugin/marketplace.json`
  - 创建详细文档：`.claude/skills/codex-review/README.md`
  - 涉及文件：`.claude/skills/codex-review/SKILL.md`, `.claude-plugin/marketplace.json`, `.claude/skills/codex-review/README.md`

### 优化

- **渠道活跃度图表颜色优化** - 状态条柱状图颜色改为显示每个 6 秒段的独立成功率
  - 修改 SVG 渐变定义：为每个柱子单独定义渐变色（`gradient-${channelIndex}-${i}`）
  - 重构 `getActivityBars` 函数：为每个 6 秒时间段计算独立的成功率并分配颜色
  - 颜色规则（7 档分级）：
    - 深红色（0-5%）：极端故障
    - 红色（5-20%）：严重失败
    - 深橙色（20-40%）：高失败率
    - 橙色（40-60%）：中等失败率
    - 黄色（60-80%）：轻微失败
    - 黄绿色（80-95%）：良好
    - 绿色（95-100%）：优秀
  - 效果：用户可以更清晰地看到每个时间段的健康状况，颜色变化更细腻
  - 性能优化：新增 `activityBarsCache` 计算属性缓存柱状图数据，避免重复计算
  - 代码清理：删除未使用的 `activityColorCache` 和 `getActivityColor` 函数
  - 涉及文件：`frontend/src/components/ChannelOrchestration.vue`

- **修复 Dashboard 切换 Tab 时数据闪烁问题** - 将 Dashboard 数据改为按 API 类型独立缓存
  - 重构 `channelStore`：将单一全局 `dashboardMetrics`/`dashboardStats`/`dashboardRecentActivity` 改为按 Tab（messages/responses/gemini）独立缓存的 `dashboardCache` 结构
  - 新增 `currentDashboardMetrics`、`currentDashboardStats`、`currentDashboardRecentActivity` 计算属性，根据当前 Tab 返回对应缓存数据
  - 切换 Tab 时直接显示该 Tab 的缓存数据，避免显示其他 Tab 的旧数据导致闪烁
  - 涉及文件：`frontend/src/stores/channel.ts`、`frontend/src/views/ChannelsView.vue`

### 重构

- **前端系统状态管理重构** - 将 App.vue 中的系统级状态迁移到 SystemStore
  - 新增 `src/stores/system.ts` 系统状态 Store，统一管理系统运行状态、版本信息、Fuzzy 模式加载状态
  - 重构 `src/App.vue`，移除本地系统状态变量（systemStatus、versionInfo、isCheckingVersion、fuzzyModeLoading、fuzzyModeLoadError），改用 SystemStore 统一管理
  - 更新 `src/stores/index.ts`，导出 SystemStore
  - 新增 2 个计算属性：systemStatusText、systemStatusDesc
  - 新增 8 个状态管理方法：setSystemStatus、setVersionInfo、setCurrentVersion、setCheckingVersion、setFuzzyModeLoading、setFuzzyModeLoadError、resetSystemState
  - 优势：
    - 状态集中：所有系统级状态统一管理，避免分散在组件中
    - 代码简化：App.vue 系统状态逻辑更清晰，减少本地状态管理
    - 可复用性：其他组件可直接使用 SystemStore 的系统状态
    - 易维护：系统状态变更集中在 Store 中，便于调试和扩展
  - 涉及文件：`frontend/src/stores/system.ts`、`frontend/src/stores/index.ts`、`frontend/src/App.vue`

- **前端对话框状态管理重构** - 将 App.vue 中的对话框状态迁移到 DialogStore
  - 新增 `src/stores/dialog.ts` 对话框状态 Store，统一管理添加/编辑渠道对话框和添加 API 密钥对话框
  - 重构 `src/App.vue`，移除本地对话框状态变量（showAddChannelModal、showAddKeyModalRef、editingChannel、selectedChannelForKey、newApiKey），改用 DialogStore 统一管理
  - 更新 `src/stores/index.ts`，导出 DialogStore
  - 新增 6 个状态管理方法：openAddChannelModal、openEditChannelModal、closeAddChannelModal、openAddKeyModal、closeAddKeyModal、resetDialogState
  - 优势：
    - 状态集中：所有对话框相关状态统一管理，避免分散在组件中
    - 代码简化：App.vue 对话框逻辑更清晰，减少本地状态管理
    - 可复用性：其他组件可直接使用 DialogStore 的对话框状态
    - 易维护：对话框状态变更集中在 Store 中，便于调试和扩展
  - 涉及文件：`frontend/src/stores/dialog.ts`、`frontend/src/stores/index.ts`、`frontend/src/App.vue`

- **前端偏好设置管理重构** - 将 App.vue 中的用户偏好设置迁移到 PreferencesStore
  - 新增 `src/stores/preferences.ts` 偏好设置 Store，统一管理暗色模式、Fuzzy 模式、全局统计面板状态
  - 重构 `src/App.vue`，移除本地偏好设置变量（darkModePreference、fuzzyModeEnabled、showGlobalStats），改用 PreferencesStore 统一管理
  - 更新 `src/stores/index.ts`，导出 PreferencesStore
  - 支持自动持久化到 localStorage（使用 pinia-plugin-persistedstate）
  - 优势：
    - 状态集中：所有用户偏好设置统一管理，避免分散在组件中
    - 自动持久化：用户设置自动保存到本地存储，刷新页面后保持
    - 代码简化：App.vue 偏好设置逻辑更清晰，减少本地状态管理
    - 可复用性：其他组件可直接使用 PreferencesStore 的偏好设置
  - 涉及文件：`frontend/src/stores/preferences.ts`、`frontend/src/stores/index.ts`、`frontend/src/App.vue`

- **前端认证状态管理重构** - 将 App.vue 中的认证相关状态迁移到 AuthStore
  - 扩展 `src/stores/auth.ts`，新增认证 UI 状态管理（authError、authAttempts、authLockoutTime、isAutoAuthenticating、isInitialized、authLoading、authKeyInput）
  - 重构 `src/App.vue`，移除本地认证状态变量，改用 AuthStore 统一管理
  - 新增 `isAuthLocked` 计算属性，自动判断认证锁定状态
  - 新增 8 个状态管理方法：setAuthError、incrementAuthAttempts、resetAuthAttempts、setAuthLockout、setAutoAuthenticating、setInitialized、setAuthLoading、setAuthKeyInput
  - 优势：
    - 状态集中：所有认证相关状态统一管理，避免分散在组件中
    - 代码简化：App.vue 认证逻辑更清晰，减少本地状态管理
    - 可复用性：其他组件可直接使用 AuthStore 的认证状态
    - 安全性增强：认证失败次数和锁定时间集中管理，便于扩展
  - 涉及文件：`frontend/src/stores/auth.ts`、`frontend/src/App.vue`

- **前端渠道管理逻辑重构** - 将 App.vue 中的渠道管理逻辑提取到 Pinia Store
  - 新增 `src/stores/channel.ts` 渠道状态 Store，统一管理三种 API 类型（Messages/Responses/Gemini）的渠道数据
  - 重构 `src/App.vue`，移除 300+ 行本地状态和业务逻辑，改用 ChannelStore 统一管理
  - 更新 `src/stores/index.ts`，导出 ChannelStore
  - 优势：
    - 代码解耦：App.vue 从 1000+ 行减少到 700+ 行，职责更清晰
    - 状态集中：渠道数据、指标、自动刷新定时器统一管理
    - 可复用性：其他组件可直接使用 ChannelStore，无需通过 props 传递
    - 可测试性：业务逻辑独立于组件，便于单元测试
  - 涉及文件：`frontend/src/stores/channel.ts`、`frontend/src/stores/index.ts`、`frontend/src/App.vue`

- **前端状态管理架构升级** - 引入 Pinia 状态管理库，替代原有的本地状态管理
  - 新增 `pinia` 和 `pinia-plugin-persistedstate` 依赖，实现响应式状态管理和自动持久化
  - 新增 `src/stores/auth.ts` 认证状态 Store，统一管理 API Key 和认证状态
  - 重构 `src/services/api.ts`，从 AuthStore 获取 API Key，移除本地状态管理逻辑
  - 重构 `src/App.vue`，使用 AuthStore 替代 `isAuthenticated` 本地状态，简化认证流程
  - 更新 `src/main.ts`，初始化 Pinia 和持久化插件
  - 配置 `tsconfig.json` 路径别名 `@/*`，支持模块化导入
  - 优势：响应式状态管理、自动持久化、更好的类型推断、代码解耦
  - 涉及文件：`frontend/package.json`、`frontend/src/stores/auth.ts`、`frontend/src/services/api.ts`、`frontend/src/App.vue`、`frontend/src/main.ts`、`frontend/tsconfig.json`

---

## [v2.4.34] - 2026-01-17

### 新增

- **会话管理增强** - 支持 Gemini API 的 `X-Gemini-Api-Privileged-User-Id` 请求头
  - 在 `ExtractConversationID()` 函数中新增对该请求头的支持，用于会话亲和性管理
  - 优先级顺序：Conversation_id > Session_id > X-Gemini-Api-Privileged-User-Id > prompt_cache_key > metadata.user_id
  - 涉及文件：`backend-go/internal/handlers/common/request.go`

### 优化

- **Gemini Dashboard API 性能优化** - 将前端 3 个独立请求合并为 1 个后端统一接口
  - 新增 `/api/gemini/channels/dashboard` 端点，一次性返回 channels、metrics、stats、recentActivity 数据
  - 后端新增 `internal/handlers/gemini/dashboard.go` 处理器，减少网络往返次数
  - 涉及文件：`backend-go/main.go`、`backend-go/internal/handlers/gemini/dashboard.go`

### 重构

- **前端 UI 框架统一** - 移除 Tailwind CSS 和 DaisyUI，完全使用 Vuetify
  - 从 package.json 移除 tailwindcss、daisyui、autoprefixer、postcss 依赖
  - 删除 tailwind.config.js 和 postcss.config.js 配置文件
  - 更新 src/assets/style.css，移除 @tailwind 指令，保留自定义样式
  - 优势：消除多框架样式冲突、减少打包体积、统一设计语言（Material Design）
  - 涉及文件：`frontend/package.json`、`frontend/src/assets/style.css`、`frontend/src/main.ts`

---

## [v2.4.33] - 2026-01-17

### 新增

- **渠道实时活跃度可视化** - 在渠道列表中显示最近 15 分钟的活跃度数据
  - 后端新增 `GetRecentActivityMultiURL()` 方法，按 **6 秒粒度**分段统计请求量、成功/失败数、Token 消耗（共 150 段）
  - **支持多 URL 和多 Key 聚合**：自动聚合渠道所有故障转移 URL 和所有活跃 API Key 的数据，提供完整的渠道活跃度视图
  - Dashboard API 返回 `recentActivity` 字段，包含每个渠道的 150 段活跃度数据
  - 前端渠道行显示 RPM/TPM 指标，**背景波形柱状图**实时反映活跃度变化（整体颜色根据全局失败率着色：绿色=成功率≥80%，橙色=成功率≥50%，红色=成功率<50%）
  - 柱状图每 2 秒自动更新，用户调用 API 后立即看到柱子"跳动"，提供直观的脉冲式活跃度展示
  - 涉及文件：`backend-go/internal/metrics/channel_metrics.go`、`backend-go/internal/handlers/channel_metrics_handler.go`、`frontend/src/components/ChannelOrchestration.vue`、`frontend/src/services/api.ts`、`frontend/src/App.vue`

---

## [v2.4.32] - 2026-01-14

### ✨ 新增

- **Gemini 渠道支持 thinking 模式函数调用签名传递** - `GeminiFunctionCall` 结构体新增 `ThoughtSignature` 字段
  - 用于 thinking 模式下的签名，需原样传回上游
  - 涉及文件：`backend-go/internal/types/gemini.go`

### 🔧 优化

- **Gemini 渠道添加模态框增强** - 扩展服务类型和模型选项
  - 服务类型新增 OpenAI 和 Claude 选项，支持更多上游协议
  - 更新 Gemini 模型列表：新增 gemini-2、gemini-2.5-flash-lite、gemini-2.5-flash-image、TTS 预览模型、gemini-3 系列预览模型
  - 涉及文件：`frontend/src/components/AddChannelModal.vue`

### 🐛 修复

- **修复快速输入解析器冒号分隔导致 URL 被截断的问题** - 增强 `extractTokens()` 函数支持冒号作为分隔符，同时保护 URL 完整性
  - 新增 URL 占位符机制：先提取完整 URL 并替换为占位符，分割后再恢复
  - 支持中文标点分隔符：逗号（，）、分号（；）、冒号（：）
  - 涉及文件：`frontend/src/utils/quickInputParser.ts`

---

## [v2.4.31] - 2026-01-12

### 🐛 修复

- **修复流式工具调用输出稳定性和合并逻辑** - 增强 `stream_synthesizer.go` 的工具调用处理
  - 工具调用输出按 index 排序，避免 map 遍历顺序不稳定导致日志顺序随机
  - 修复 ID 生成错误：`string(rune(index))` 改为 `strconv.Itoa(index)`，避免非 ASCII 字符
  - 合并逻辑增强：仅合并连续 index 的工具调用，防止误合并不相关调用
  - 新增 ID 匹配检查：合并时验证两个 block 的 ID 一致（或其中一个为空）
  - 支持 ID 补全：合并时若 curr 无 ID 但 next 有，自动补全
  - 涉及文件：`backend-go/internal/utils/stream_synthesizer.go`

---

## [v2.4.30] - 2026-01-10

### 🐛 修复

- **修复流式响应工具调用分裂问题** - 当上游返回的工具调用被意外分成两个 content_block 时自动合并
  - 问题场景：第一个 block 有 name 和 id 但参数为空 "{}"，第二个 block 没有 name 但有完整参数
  - 新增 `mergeSplitToolCalls()` 方法检测并合并分裂的工具调用
  - 在 `GetSynthesizedContent()` 中调用，确保日志输出正确的工具调用信息
  - 涉及文件：`backend-go/internal/utils/stream_synthesizer.go`

---

## [v2.4.29] - 2026-01-10

### 🐛 修复

- **修复空 signature 字段导致 Claude API 400 错误** - 客户端可能发送带空 `signature` 字段（空字符串或 null）的请求，Claude API 会拒绝并返回 400 错误
  - 新增 `RemoveEmptySignatures()` 函数，定向移除 `messages[*].content[*].signature` 路径下的空值
  - 使用 `json.Decoder` 保留数字精度，`SetEscapeHTML(false)` 保持原始格式
  - **注意**：当请求体被修改时，JSON 字段顺序可能发生变化（不影响 API 语义）
  - 在 Messages Handler 入口处调用预处理，确保请求发送前清理无效字段
  - 涉及文件：`backend-go/internal/handlers/common/request.go`、`backend-go/internal/handlers/messages/handler.go`

### ✨ 改进

- **增强 Trace 亲和性日志记录** - 在关键操作点添加详细日志，方便排查亲和性相关问题
  - `[Affinity-Set]` 记录新建/变更用户亲和
  - `[Affinity-Remove]` 记录手动移除用户亲和
  - `[Affinity-RemoveByChannel]` 记录渠道移除时批量清理
  - `[Affinity-Cleanup]` 记录定时清理过期记录
  - 日志在锁外执行，避免高负载下的尾延迟
  - 用户 ID 分级脱敏：短 ID 也保留部分字符便于关联
  - 涉及文件：`backend-go/internal/session/trace_affinity.go`

## [v2.4.28] - 2026-01-07

### 🐛 修复

- **修复内容审核错误导致无限重试问题** - 当上游返回 `sensitive_words_detected` 等内容审核错误时，单渠道场景下会无限重试
  - 根因：`classifyByStatusCode(500)` 触发 failover，但未检查 `error.code` 字段中的不可重试错误码
  - 新增 `isNonRetryableErrorCode()` 函数，检测内容审核和无效请求错误码
  - 新增 `isNonRetryableError()` 函数，从响应体提取并检测不可重试错误
  - 在 `shouldRetryWithNextKeyNormal()` 和 `shouldRetryWithNextKeyFuzzy()` 入口处优先检测
  - 不可重试错误码：`sensitive_words_detected`、`content_policy_violation`、`content_filter`、`content_blocked`、`moderation_blocked`、`invalid_request`、`invalid_request_error`、`bad_request`
  - 涉及文件：`backend-go/internal/handlers/common/failover.go`

### 🧪 测试

- **新增不可重试错误码测试** - 覆盖 `sensitive_words_detected` 等错误码在 Normal/Fuzzy 模式下的行为
  - 涉及文件：`backend-go/internal/handlers/common/failover_test.go`

## [v2.4.27] - 2026-01-05

### 🐛 修复

- **修复多端点 failover 渠道统计丢失问题** - 当渠道配置多个 `baseUrls` 时，请求路由到非主 URL 后指标无法正确聚合到渠道统计
  - 根因：指标存储使用 `hash(baseURL + apiKey)` 作为键，但查询方法只使用主 BaseURL
  - 新增 4 个多 URL 聚合方法：`GetHistoricalStatsMultiURL`、`GetChannelKeyUsageInfoMultiURL`、`GetKeyHistoricalStatsMultiURL`、`calculateAggregatedTimeWindowsMultiURL`
  - `ToResponseMultiURL` 按 API Key 去重聚合，避免同一 Key 在多 URL 场景下产生重复条目
  - Handler 层全部改用 `upstream.GetAllBaseURLs()` 获取所有 URL 进行聚合
  - 涉及文件：`backend-go/internal/metrics/channel_metrics.go`、`backend-go/internal/handlers/channel_metrics_handler.go`

## [v2.4.26] - 2026-01-05

### 🐛 修复

- **修复 Key 趋势图切换时间范围后不刷新问题** - 持久化 view/duration 选择到 localStorage，使用 requestId 防止自动刷新旧响应覆盖新选择
  - 涉及文件：`frontend/src/components/KeyTrendChart.vue`

- **修复 KeyTrendChart SSR 兼容性和健壮性问题**
  - 添加 `isLocalStorageAvailable()` 检查，防止 SSR 环境下访问 localStorage 崩溃
  - 为 localStorage 读写操作添加 try/catch 异常捕获（配额超限、隐私模式等场景）
  - 添加 `channelType` prop 变化监听，切换渠道类型时自动重载偏好设置并刷新数据
  - 优化 channelType watcher 逻辑，避免与 duration watcher 重复触发刷新
  - 涉及文件：`frontend/src/components/KeyTrendChart.vue`

- **修复缓存创建统计缺失问题** - 当上游仅返回 TTL 细分字段（5m/1h）时，兜底汇总为 cacheCreationTokens
  - 涉及文件：`backend-go/internal/metrics/channel_metrics.go`

- **透传缓存 TTL 细分字段到指标层** - Responses 非流式/流式 usage 现在包含 CacheCreation5m/1h + CacheTTL
  - 涉及文件：`backend-go/internal/handlers/responses/handler.go`

### 🧪 测试

- **新增 TTL 细分字段兜底测试** - 覆盖 cache_creation_input_tokens 为 0 时的汇总场景
  - 涉及文件：`backend-go/internal/metrics/channel_metrics_cache_stats_test.go`

## [v2.4.25] - 2026-01-04

### 🧪 测试

- **新增 baseUrl/baseUrls 一致性测试套件** - 覆盖 URL 配置的完整场景，防止编辑渠道时数据不一致问题回归
  - `TestUpdateUpstream_BaseURLConsistency`: 验证 Messages 渠道更新时 baseUrl/baseUrls 的一致性（4 场景）
  - `TestUpdateResponsesUpstream_BaseURLConsistency`: 验证 Responses 渠道更新一致性
  - `TestUpdateGeminiUpstream_BaseURLConsistency`: 验证 Gemini 渠道更新一致性
  - `TestGetAllBaseURLs_Priority`: 验证 URL 获取优先级逻辑（4 场景）
  - `TestGetEffectiveBaseURL_Priority`: 验证有效 URL 选择逻辑（3 场景）
  - `TestDeduplicateBaseURLs`: 验证 URL 去重逻辑（7 场景，含末尾斜杠/井号差异）
  - `TestAddUpstream_BaseURLDeduplication`: 验证添加渠道时的 URL 去重
  - 涉及文件：`internal/config/config_baseurl_test.go`（新增 414 行）

### 🐛 修复

- **修复历史分桶边界导致边界点漏算** - 历史统计 API 的时间过滤条件从开区间 `(startTime, endTime)` 改为半开区间 `[startTime, endTime)`，避免恰好落在 startTime 的记录被遗漏
  - 涉及文件：`internal/metrics/channel_metrics.go`

- **修复历史图表时间戳错位** - 将返回的 Timestamp 从"桶结束时间"改为"桶起始时间"，前端图表不再出现一格偏差
  - 涉及文件：`internal/metrics/channel_metrics.go`

- **修复成功计数可能重复记录** - 移除多渠道/单渠道成功路径上多余的 `RecordSuccess()` 调用，统一使用 `RecordSuccessWithUsage()` 作为唯一成功计数入口
  - Messages 路径：移除重复调用，保留流式/非流式末尾的 `RecordSuccessWithUsage`
  - Responses compact 路径：改用 `RecordSuccessWithUsage(nil)` 替代原 `RecordSuccess`，保持指标一致性
  - 涉及文件：`internal/handlers/messages/handler.go`、`internal/handlers/responses/compact.go`

- **修复多 BaseURL 故障转移时成功指标归属错误** - 当请求通过 fallback BaseURL 成功时，成功指标错误地记录到主 BaseURL 而非实际成功的 URL
  - 根本原因：`handleNormalResponse` 和 `HandleStreamResponse` 接收的是原始 `upstream` 而非设置了 `currentBaseURL` 的 `upstreamCopy`
  - 修复方式：将两处调用点的参数从 `upstream` 改为 `upstreamCopy`
  - 影响范围：多渠道/单渠道的流式与非流式响应处理
  - 涉及文件：`internal/handlers/messages/handler.go`

---

## [v2.4.24] - 2026-01-04

### ✨ 新功能

- **缓存命中率统计** - 按 Token 口径展示各渠道缓存读/写与命中率：
  - 后端：在 `timeWindows` 聚合统计中新增 `inputTokens`/`outputTokens`/`cacheCreationTokens`/`cacheReadTokens`/`cacheHitRate` 字段
  - 命中率定义：`cacheReadTokens / (cacheReadTokens + inputTokens) * 100`
  - 前端：渠道编排列表在 15 分钟有请求时额外显示缓存命中率，tooltip 中按 15m/1h/6h/24h 展示缓存统计
  - 新字段均为 `omitempty`，向后兼容

### 🎨 优化

- **调整渠道指标显示间距** - 优化缓存命中率 chip 与请求数之间的间距，避免布局拥挤

---

## [v2.4.23] - 2026-01-03

### ✨ 新功能

- **lowQuality 模式输出完整的 token 验证过程日志** - 启用低质量渠道时，日志会显示完整的验证过程：
  - 偏差 > 5% 时显示修补详情
  - 偏差 ≤ 5% 时显示保留上游值
  - 上游返回无效值时显示本地估算值

### 🐛 修复

- **修复渠道列表 API 未返回 `lowQuality` 字段** - 在 `GetUpstreams` 和 `GetChannelDashboard` 函数返回的 JSON 中补充 `lowQuality` 字段：
  - 之前前端编辑渠道时无法正确显示已保存的"低质量渠道"开关状态
  - 涉及文件：`handlers/messages/channels.go`、`handlers/responses/channels.go`、`handlers/gemini/channels.go`、`handlers/channel_metrics_handler.go`

---

## [v2.4.22] - 2026-01-02

### ✨ 新功能

- **低质量渠道处理机制** - 新增 `lowQuality` 渠道配置选项，用于处理返回不完整数据的上游渠道：
  - Token 偏差检测：启用后对比上游返回值与本地估算值，偏差 > 5% 时使用本地估算值
  - Model 一致性检查：验证响应中的 model 是否与请求一致，不一致则改写为请求的 model
  - 空 ID 补全：自动补全上游返回的空 `message.id`（生成 `msg_<uuid>` 格式）
  - 前端支持：渠道编辑 modal 新增"低质量渠道"开关

### 🐛 修复

- **暂停渠道时自动清除促销期** - 当用户暂停一个正在抢优先级的渠道时，自动清除其 `promotionUntil` 字段：
  - 避免暂停后仍显示促销期标识
  - 涉及三个渠道类型：Messages、Responses、Gemini
  - 涉及文件：`config_messages.go`、`config_responses.go`、`config_gemini.go`

- **修复 `lowQuality` 字段更新不持久化的问题** - 在 `UpdateUpstream` 系列函数中补充 `LowQuality` 字段处理：
  - 之前前端切换"低质量渠道"开关后变更不会被保存
  - 涉及文件：`config_messages.go`、`config_responses.go`、`config_gemini.go`

- **修复渠道列表 API 未返回 `lowQuality` 字段** - 在 `GetUpstreams` 和 `GetChannelDashboard` 函数返回的 JSON 中补充 `lowQuality` 字段：
  - 之前前端编辑渠道时无法正确显示已保存的"低质量渠道"开关状态
  - 涉及文件：`handlers/messages/channels.go`、`handlers/responses/channels.go`、`handlers/gemini/channels.go`、`handlers/channel_metrics_handler.go`

---

## [v2.4.21] - 2026-01-02

### 🐛 修复

- **修复流式响应 input_tokens 为 nil 时丢失的问题** - 当上游返回的顶层 usage 中 `input_tokens` 为 `nil` 时，之前从 `message.usage` 收集到的有效值无法被修补：
  - 原因：`patchUsageFieldsWithLog` 和 `checkUsageFieldsWithPatch` 函数中类型断言 `.(float64)` 失败时跳过了修补逻辑
  - 表现：日志显示 `InputTokens=<nil>` 而非之前收集到的有效值（如 10920）
  - 修复：在两处函数中新增 `input_tokens == nil` 检测，无论是否有缓存 token 都用收集到的值修补
  - 涉及文件：`backend-go/internal/handlers/common/stream.go`

---

## [v2.4.18] - 2025-12-31

### 🐛 修复

- **Gemini 日志和 Header 透传改进** - 修复 Gemini 接口的日志显示和请求头处理：
  - 修复 `contents`/`parts` 字段在日志中不显示的问题
  - 修复原生 Gemini handler 未透传客户端 Header 的问题
  - 新增 `compactGeminiContentsArray` 和 `compactGeminiPart` 函数
  - 涉及文件：`backend-go/internal/utils/json.go`、`backend-go/internal/handlers/gemini/handler.go`

### 🔧 重构

- **Gemini tools 日志简化支持** - 新增 `extractToolNames` 函数支持 Gemini 格式的工具提取：
  - 支持 Gemini `functionDeclarations` 数组格式
  - 兼容 Claude 和 OpenAI 格式
  - 日志中 tools 字段现在统一显示为 `["tool1", "tool2", ...]` 格式
  - 涉及文件：`backend-go/internal/utils/json.go`

- **移除非标准 Gemini API 路由** - 简化 API 端点，仅保留官方格式：
  - 移除：`POST /v1/models/{model}:generateContent`（非标准简化格式）
  - 保留：`POST /v1beta/models/{model}:generateContent`（Gemini 官方格式）
  - 更新前端预览 URL 显示完整路径格式 `/models/{model}:generateContent`
  - 涉及文件：`backend-go/main.go`、`frontend/src/components/AddChannelModal.vue`

---

## [v2.4.17] - 2025-12-30

### 🐛 修复

- **修复 ModelMapping 导致请求字段丢失** - 解决使用模型重定向时 Claude API 返回 403 的问题：
  - 原因：`ClaudeRequest` 结构体缺少 `metadata` 字段，JSON 反序列化时该字段被丢弃
  - 表现：配置 `modelMapping` 后请求被上游拒绝（如 `opus` → `claude-opus-4-5-20251101`）
  - 修复：在 `ClaudeRequest` 中添加 `Metadata map[string]interface{}` 字段
  - 涉及文件：`backend-go/internal/types/types.go`

---

## [v2.4.16] - 2025-12-30

### 🐛 修复

- **修复 Gemini 渠道预期请求 URL 预览** - 创建渠道时预览显示正确的 `/v1beta` 路径：
  - 原问题：Gemini 渠道预览错误显示 `/v1` 而后端实际使用 `/v1beta`
  - 修复：当 serviceType 为 gemini 时使用 `/v1beta` 作为版本前缀
  - 涉及文件：`frontend/src/components/AddChannelModal.vue`

---

## [v2.4.15] - 2025-12-30

### 🐛 修复

- **修复 Gemini API 路由注册失败** - 解决 Gin 框架路由 panic 问题：
  - 原因：Gin 不支持 `:param\:literal` 格式，即使转义冒号也会被解析为两个通配符
  - 方案：使用 `*modelAction` 通配符捕获 `model:action` 整体，在 handler 内解析
  - 涉及文件：`main.go`、`internal/handlers/gemini/handler.go`

### ✨ 新功能

- **Gemini 历史指标 API 完整实现** - 补全 Gemini 模块的历史数据端点：
  - `GET /api/gemini/channels/metrics/history` - 渠道级别指标历史
  - `GET /api/gemini/channels/:id/keys/metrics/history` - Key 级别指标历史
  - `GET /api/gemini/global/stats/history` - 全局统计历史
  - 涉及文件：`internal/handlers/channel_metrics_handler.go`、`main.go`

- **Gemini 前端管理界面完整实现** - 与 Messages/Responses 功能完全对齐：
  - 新增 Gemini Tab 切换，支持完整渠道 CRUD、Key 管理、状态/促销设置
  - KeyTrendChart 和 GlobalStatsChart 组件支持 Gemini 数据展示（移除降级显示）
  - 涉及文件：`frontend/src/App.vue`、`frontend/src/components/`、`frontend/src/services/api.ts`

---

## [v2.4.14] - 2025-12-29

### ✨ 新功能

- **新增 Gemini API 模块** - 与 `/v1/messages`、`/v1/responses` 同级的完整 Gemini 代理支持：
  - **代理端点**：`POST /v1/models/{model}:generateContent`（非流式）、`:streamGenerateContent`（流式）
  - **协议转换**：支持 Gemini 请求转发到 Claude/OpenAI/Gemini 上游，双向转换器自动处理格式差异
  - **渠道管理 API**：完整 CRUD、API Key 管理、状态/促销设置、指标监控（`/api/gemini/channels/*`）
  - **多渠道调度**：集成 ChannelScheduler，支持优先级、熔断、Trace 亲和性
  - **认证方式**：兼容 Gemini 原生格式（`x-goog-api-key` 头、`?key=` 参数）
  - 涉及文件：`internal/handlers/gemini/`、`internal/converters/gemini_converter.go`、`internal/types/gemini.go`

### 🔧 重构

- **config 包模块化拆分** - 将 1973 行的单文件拆分为 6 个职责清晰的模块：
  - `config.go`（297 行）：核心类型定义 + 共享方法
  - `config_loader.go`（384 行）：配置加载、迁移、验证、文件监听
  - `config_messages.go`（429 行）：Messages 渠道 CRUD
  - `config_responses.go`（380 行）：Responses 渠道 CRUD
  - `config_gemini.go`（361 行）：Gemini 渠道 CRUD
  - `config_utils.go`（183 行）：工具函数（去重、模型重定向、状态辅助）
  - 遵循单一职责原则，提升代码可维护性

---

## [v2.4.12] - 2025-12-29

### 🐛 修复

- **修复 Responses API 错误消息提取失败的问题** - 解决 upstream_error 字段无法被正确解析：
  - 扩展 `classifyByErrorMessage` 函数：支持多个消息字段（`message`, `upstream_error`, `detail`）
  - 支持嵌套对象格式：当 `upstream_error` 为对象时，提取其中的 `message` 字段
  - 之前仅检查 `error.message` 字段，导致 `{type, upstream_error}` 格式的错误无法被识别
  - 新增 4 个测试用例覆盖 upstream_error 字符串、嵌套对象、detail 字段等场景
  - 涉及文件：`internal/handlers/common/failover.go`, `internal/handlers/common/failover_test.go`

---

## [v2.4.11] - 2025-12-29

### 🐛 修复

- **修复 Fuzzy 模式下 403 + 预扣费消息未触发 Key 降级的问题** - 补充 v2.4.10 修复的遗漏场景：
  - 修改 `shouldRetryWithNextKeyFuzzy` 函数：新增 `bodyBytes` 参数，对非 402/429 状态码检查消息体中的配额关键词
  - 之前 Fuzzy 模式仅检查状态码（402/429 = quota），不解析消息体，导致 403 + "预扣费额度失败" 返回 `isQuotaRelated=false`
  - 新增 `TestShouldRetryWithNextKey_FuzzyMode_403WithQuotaMessage` 测试用例
  - 涉及文件：`internal/handlers/common/failover.go`, `internal/handlers/common/failover_test.go`

### 🔧 调试

- **添加 Key 降级调试日志** - 用于追踪 `isQuotaRelated` 值和密钥降级流程：
  - 在 `ShouldRetryWithNextKey` 调用后记录返回值（statusCode, shouldFailover, isQuotaRelated）
  - 在密钥标记为配额相关失败时记录日志
  - 涉及文件：`internal/handlers/messages/handler.go`
- **改进 .env.example 文档** - 添加日志配置默认值说明（默认启用，需显式设置 false 禁用）

---

## [v2.4.10] - 2025-12-29

### 🐛 修复

- **修复 403 预扣费额度不足的 Key 未被自动降级的问题** - 解决配额不足的密钥始终被优先尝试：
  - 修改 `shouldRetryWithNextKeyNormal` 逻辑：即使 HTTP 状态码已触发 failover，仍检查消息体确定是否为配额相关错误
  - 之前 403 状态码直接返回 `isQuotaRelated=false`，跳过消息体解析，导致 `DeprioritizeAPIKey` 未被调用
  - 新增 "预扣费" 关键词到 `quotaKeywords` 列表，确保匹配中文预扣费错误消息
  - 涉及文件：`internal/handlers/common/failover.go`

---

## [v2.4.9] - 2025-12-27

### 🔧 改进

- **重构 URL 预热机制为非阻塞动态排序** - 解决首次请求延迟 500ms+ 的问题：
  - 移除阻塞式 ping 预热（`URLWarmupManager`），改用非阻塞的 `URLManager`
  - 新排序策略：基于实际请求结果动态调整 URL 顺序
    - 请求成功：重置失败计数，URL 保持/提升位置
    - 请求失败：增加失败计数，URL 移到末尾
    - 冷却期机制：失败的 URL 在 30 秒后自动恢复可用
  - 排序规则：无失败记录优先 > 冷却期已过 > 仍在冷却期
  - 涉及文件：`warmup/url_manager.go`（新建）、`warmup/url_warmup.go`（删除）、`scheduler/channel_scheduler.go`、`messages/handler.go`、`responses/handler.go`、`main.go`

---

## [v2.4.8] - 2025-12-27

### 🐛 修复

- **修复多端点渠道密钥轮换时的并发竞争问题** - 解决高并发下 BaseURL 被错误修改导致密钥跨渠道混用：
  - 新增 `UpstreamConfig.Clone()` 深拷贝方法，避免并发修改共享对象
  - Messages/Responses Handler 改用深拷贝替代临时修改模式
  - 新增 `MarkWarmupURLFailed()` 方法，请求失败时触发预热缓存失效
  - HTTP 5xx 和网络超时均会触发预热缓存失效，确保失败端点被重新排序
  - 涉及文件：`config/config.go`、`messages/handler.go`、`responses/handler.go`、`scheduler/channel_scheduler.go`、`warmup/url_warmup.go`

---

## [v2.4.6] - 2025-12-27

### ✨ 新功能

- **多端点预热排序** - 渠道首次访问前自动 ping 所有端点，按延迟排序：
  - 新增 `internal/warmup/url_warmup.go` 预热管理器模块
  - 渠道首次访问时自动并发 ping 所有 BaseURL
  - 排序策略：成功的端点优先，同类型按延迟从低到高排序
  - ping 结果缓存 5 分钟，避免频繁测试
  - 支持并发安全的预热请求去重（多个请求同时触发时只执行一次预热）
  - Messages 和 Responses API 均支持预热排序

---

## [v2.4.5] - 2025-12-27

### 🔧 改进

- **统一日志前缀规范** - Messages 和 Responses 接口日志标签标准化：
  - Messages 流式处理日志统一使用 `[Messages-Stream]`、`[Messages-Stream-Token]` 前缀
  - Responses 流式处理日志保持 `[Responses-Stream]`、`[Responses-Stream-Token]` 前缀
  - 修复 3 处遗漏前缀的错误日志（`messages/handler.go`、`responses/handler.go`）
  - 更新 `backend-go/CLAUDE.md` 日志规范文档

---

## [v2.4.4] - 2025-12-27

### ✨ 新功能

- **全局流量和 Token 统计图表** - 新增全局统计可视化功能：
  - 后端新增 `/api/messages/global/stats/history` 和 `/api/responses/global/stats/history` API
  - 支持请求数量（成功/失败/总量）和 Token 总量（输入/输出）统计
  - 前端新增 `GlobalStatsChart.vue` 组件，支持流量/Token 双视图切换
  - 时间范围支持 1h / 6h / 24h / 今日 多档位切换
  - 用户偏好（时间范围、视图模式）按 Messages/Responses 分别保存到 localStorage
  - 以顶部可折叠卡片形式展示，随当前 Tab 自动切换对应 API 类型的统计

- **渠道 Key 趋势图表支持"今日"** - KeyTrendChart 新增今日时间范围选项：
  - 后端 `GetChannelKeyMetricsHistory` 支持 `duration=today` 参数
  - 前端添加"今日"按钮，动态计算从今日 0 点到当前的时长

---

## [v2.4.3] - 2025-12-27

### 🐛 修复

- **Responses API Token 统计修复** - 解决上游无 usage 时本地统计无数据的问题：
  - 修复 SSE 事件解析格式兼容性：支持 `data:` 和 `data: ` 两种格式（某些上游不带空格）
  - 修复 `handleSuccess` / `handleStreamSuccess` 不返回 usage 数据的问题
  - 修复调用点使用 `RecordSuccess` 而非 `RecordSuccessWithUsage` 导致 token 统计未入库
  - 涉及函数：`checkResponsesEventUsage`、`injectResponsesUsageToCompletedEvent`、`patchResponsesCompletedEventUsage`、`tryChannelWithAllKeys`

---

## [v2.4.2] - 2025-12-26

### 🐛 修复

- **原始请求日志修复** - 修复多渠道模式下原始请求头/请求体日志不显示的问题：
  - 将 `LogOriginalRequest` 调用移至 Handler 入口处，确保无论单/多渠道模式都只记录一次
  - 移除单渠道处理函数中重复的日志调用和未使用变量
  - 同时修复 Messages 和 Responses 两个处理器

### 🧹 清理

- **移除废弃环境变量 `LOAD_BALANCE_STRATEGY`** - 负载均衡策略已迁移至 config.json 热重载配置：
  - 删除 `env.go` 中 `LoadBalanceStrategy` 字段
  - 更新 `.env.example`、`docker-compose.yml`、`README.md` 移除相关配置
  - 更新 `CLAUDE.md` 添加配置方式说明

---

## [v2.4.0] - 2025-12-26

### ✨ 改进

- **渠道编辑表单优化** - 改进 AddChannelModal 用户体验：
  - 预期请求支持显示所有 BaseURL 端点，而非仅显示首个
  - 修复 Gemini 类型渠道预期请求显示错误端点的问题（应为 `/generateContent`）
  - 修复从快速模式切换到详细模式时 BaseURL 输入框为空的问题
  - 表单字段重排：TLS 验证开关和描述字段移至表单末尾
  - BaseURL 输入框不再自动修改用户输入，仅在提交时进行去重处理
  - 调整预期请求区域下方间距，改善视觉效果

- **API Key/BaseURL 策略简化** - 移除过度设计，采用纯 failover 模式：
  - 删除 `ResourceAffinityManager` 及相关代码（资源亲和性）
  - 移除 API Key 策略选择（round-robin/random/failover），始终使用优先级顺序
  - 移除 BaseURL 策略选择，始终使用优先级顺序并在失败时切换
  - 前端删除策略选择器，简化渠道配置界面
  - 保留渠道级 Trace 亲和性（TraceAffinityManager）用于会话一致性
  - 清理遗留无用代码：`requestCount`/`responsesRequestCount` 字段、`EnableStreamEventDedup` 环境变量

### 🐛 修复

- **多 BaseURL failover 失效** - 修复当所有 API Key 在首个 BaseURL 失败后不会切换到下一个 BaseURL 的问题：
  - 重构 `tryChannelWithAllKeys` 函数，采用嵌套循环遍历所有 BaseURL
  - 重构 `handleSingleChannel` 函数，单渠道模式也支持多 BaseURL failover
  - 每个 BaseURL 尝试所有 Key 后，若全部失败则自动切换下一个
  - 每次切换 BaseURL 时重置失败 Key 列表
  - 同时修复 Messages 和 Responses 两个处理器
  - 修复 `GetEffectiveBaseURL()` 优先级：临时设置的 `BaseURL` 字段优先于 `BaseURLs` 数组
  - 移除废弃代码：`MarkBaseURLFailed()`、`baseURLIndex` 字段

- **SSE 流式事件完整性** - 修复 Claude Provider 流式响应可能在事件边界处截断的问题：
  - 改用事件缓冲机制，按空行分隔完整 SSE 事件后再转发
  - 确保 `event:`/`data:`/`id:`/`retry:` 等字段作为整体发送
  - 处理上游未以空行结尾的边界情况

- **前端延迟测试结果被覆盖** - 修复 ping 延迟值显示几秒后消失的问题：
  - 新增 `mergeChannelsWithLocalData()` 函数保留本地延迟测试结果
  - 应用于自动刷新、Tab 切换、手动刷新三处数据更新点
  - 添加 5 分钟有效期检查，确保过期数据自动清除

---

## [v2.3.11] - 2025-12-26

### 🐛 修复

- **Responses API usage 字段缺失** - 修复当上游服务（OpenAI/Gemini）不返回 usage 信息时，`response.completed` 事件完全不包含 `usage` 字段的问题：
  - 转换器现在始终生成基础 `usage` 字段（`input_tokens`、`output_tokens`、`total_tokens`），即使值为 0
  - Handler 检测到 usage 存在后，会用本地 token 估算值替换 0 值
  - 确保下游客户端始终能获得合理的 token 使用估算

### ✨ 新功能

- **API Key/Base URL 去重** - 前后端全链路自动去重：
  - 前端详细表单模式输入时自动过滤重复 URL（忽略末尾 `/` 和 `#` 差异）
  - 后端 AddUpstream/UpdateUpstream 接口添加去重逻辑
  - 同时覆盖 Messages 和 Responses 渠道

### 🔧 改进

- **API Key 策略推荐调整** - 将默认推荐策略从"轮询"改为"故障转移"，更符合实际使用场景
- **延迟测试结果持久显示** - 优化渠道延迟测试体验：
  - 测试结果直接显示在故障转移序列列表中，不再使用短暂 Toast 通知
  - 延迟结果保持显示 5 分钟后自动清除
  - 支持单个渠道测试和批量测试统一行为

---

## [v2.3.10] - 2025-12-25

### ✨ 新功能

- **快速添加支持等号分割** - 输入 `KEY=value` 格式时自动按等号分割，识别 `value` 为 API Key
- **快速添加支持多 Base URL** - 自动识别输入中所有 HTTP 链接作为 Base URL（最多 10 个）
- **多 URL 预期请求展示** - 快速添加模式下逐一展示每个 URL 的预期请求地址

---

## [v2.3.9] - 2025-12-25

### ✨ 新功能

- **渠道级 API Key 策略** - 每个渠道可独立配置 API Key 分配策略：
  - `round-robin`（默认）：轮询分发请求到不同 Key
  - `random`：随机选择 Key
  - `failover`：故障转移，优先使用第一个 Key
  - 单 Key 时自动强制使用 `failover`，UI 显示禁用状态
- **多 BaseURL 支持** - 单个渠道可配置多个 BaseURL，支持三种策略：
  - `round-robin`（默认）：轮询分发请求，自动分散负载
  - `random`：随机选择 URL
  - `failover`：手动故障转移（需配合外部监控切换）
- **促销期状态展示** - 渠道列表显示正在"抢优先级"的渠道，带火箭图标和剩余时间
- **延迟测试优化** - 批量测试时直接在列表显示每个渠道的延迟值，颜色根据延迟等级变化（绿/黄/红）
- **多 URL 延迟测试** - 当渠道配置多个 BaseURL 时，并发测试所有 URL 并显示最快的延迟
- **资源亲和性** - 记录用户成功使用的 BaseURL 和 API Key 索引，后续请求优先使用相同资源组合，减少不必要的资源切换

---

## [v2.3.8] - 2025-12-24

### 🔨 重构

- **日志输出规范化** - 移除所有 emoji 符号，统一使用 `[Component-Action]` 标签格式，确保跨平台兼容性

---

## [v2.3.7] - 2025-12-24

### 🐛 修复

- **滑动窗口重建逻辑优化** - 服务重启时只从最近 15 分钟的历史记录重建滑动窗口，避免历史失败记录导致渠道长期处于不健康状态

---

## [v2.3.6] - 2025-12-24

### ✨ 新功能

- **快速添加渠道 - API Key 识别增强** - 大幅改进 `quickInputParser` 的密钥识别能力
  - 新增各平台特定格式支持：OpenAI (sk-/sk-proj-)、Anthropic (sk-ant-api03-)、Google Gemini (AIza)、OpenRouter (sk-or-v1-)、Hugging Face (hf_)、Groq (gsk_)、Perplexity (pplx-)、Replicate (r8_)、智谱 AI (id.secret)、火山引擎 (UUID/AK)
  - 新增宽松兜底规则：常见前缀 (sk/api/key/ut/hf/gsk/cr/ms/r8/pplx) + 任意后缀，支持识别短密钥如 `sk-111`
  - 新增配置键名排除：全大写下划线分隔格式 (如 `API_TIMEOUT_MS`) 不再被误识别为密钥

### 🐛 修复

- **Claude Code settings.json 解析修复** - 粘贴 Claude Code 配置时，不再将键名 (`ANTHROPIC_AUTH_TOKEN` 等) 误识别为 API 密钥

---

## [v2.3.5] - 2025-12-24

### ✨ 新功能

- **Responses API Token 统计补全** - 为 Responses 接口添加完整的输入输出 Token 统计功能
  - 非流式响应：自动检测上游是否返回 usage，无 usage 时本地估算，修补虚假值（`input_tokens/output_tokens <= 1`）
  - 流式响应：累积收集流事件中的文本内容，在 `response.completed` 事件中检测并修补 Token 统计
  - 新增 `EstimateResponsesRequestTokens`、`EstimateResponsesOutputTokens` 专用估算函数
  - 支持缓存 Token 细分统计（5m/1h TTL）
  - 与 Messages API 保持一致的处理逻辑

### 🐛 修复

- **缓存 Token 5m/1h 字段检测完善** - 修复缓存 Token 检测逻辑，同时检测 `cache_creation_5m_input_tokens` 和 `cache_creation_1h_input_tokens` 字段
- **类型化 ResponsesItem 处理** - `EstimateResponsesOutputTokens` 现支持直接处理 `[]types.ResponsesItem` 类型
- **total_tokens 零值补全** - 修复当上游返回有效 `input_tokens/output_tokens` 但 `total_tokens` 为 0 时未自动补全的问题（非流式和流式均已修复）
- **特殊类型 Token 估算回退** - 当 `ResponsesItem` 的 `Type` 为 `function_call`、`reasoning` 等特殊类型时，自动序列化整个结构进行估算
- **流式 delta 类型扩展** - `extractResponsesTextFromEvent` 现支持更多 delta 事件类型：`output_json.delta`、`content_part.delta`、`audio.delta`、`audio_transcript.delta`
- **流式缓冲区内存保护** - `outputTextBuffer` 添加 1MB 大小上限，防止长流式响应导致内存溢出
- **Claude/OpenAI 缓存格式区分** - 新增 `HasClaudeCache` 标志，正确区分 Claude 原生缓存字段（`cache_creation/read_input_tokens`）和 OpenAI 格式（`input_tokens_details.cached_tokens`），避免 OpenAI 格式错误阻止 `input_tokens` 补全
- **流式缓存标志传播** - 修复 `updateResponsesStreamUsage` 未传播 `HasClaudeCache` 标志的问题，确保流式响应正确识别 Claude 缓存

---

## [v2.3.4] - 2025-12-23

### ✨ 新功能

- **Models API 增强** - `/v1/models` 端点重大改进
  - 使用调度器按故障转移顺序选择渠道（与 Messages/Responses API 一致）
  - 同时从 Messages 和 Responses 两种渠道获取模型列表并合并去重
  - 添加详细日志：渠道名称、脱敏 Key、选择原因
  - 移除对 Claude 原生渠道的跳过限制（第三方 Claude 代理通常支持 /models）
  - 移除不常用的 `DELETE /v1/models/:model` 端点

---

## [v2.3.3] - 2025-12-23

### ✨ 新功能

- **Models API 端点支持** - 新增 `/v1/models` 系列端点，转发到上游 OpenAI 兼容服务
  - `GET /v1/models` - 获取模型列表
  - `GET /v1/models/:model` - 获取单个模型详情
  - `DELETE /v1/models/:model` - 删除微调模型
  - 自动跳过不支持的 Claude 原生渠道，遍历所有上游直到成功或返回 404

---

## [v2.3.2] - 2025-12-23

### ✨ 新功能

- **快速添加渠道自动检测协议类型** - 根据 URL 路径自动选择正确的服务类型
  - `/messages` → Claude 协议
  - `/chat/completions` → OpenAI 协议
  - `/responses` → Responses 协议
  - `/generateContent` → Gemini 协议
- **快速添加支持 `%20` 分隔符** - 解析输入时自动将 URL 编码的空格转换为实际空格

---

## [v2.3.1] - 2025-12-22

### ✨ 新功能

- **HTTP 响应头超时可配置** - 新增 `RESPONSE_HEADER_TIMEOUT` 环境变量（默认 60 秒，范围 30-120 秒），解决上游响应慢导致的 `http2: timeout awaiting response headers` 错误

---

## [v2.3.0] - 2025-12-22

### ✨ 新功能

- **快速添加渠道支持引号内容提取** - 支持从双引号/单引号中提取 URL 和 API Key，可直接粘贴 Claude Code 环境变量 JSON 配置格式
- **SQLite 指标持久化存储** - 服务重启后不再丢失历史指标数据，启动时自动加载最近 24 小时数据
  - 新增 `METRICS_PERSISTENCE_ENABLED`（默认 true）和 `METRICS_RETENTION_DAYS`（默认 30，范围 3-90）配置
  - 异步批量写入（100 条/批或每 30 秒），WAL 模式高并发，自动清理过期数据
- **完整的 Responses API Token Usage 统计** - 支持多格式自动检测（Claude/Gemini/OpenAI）、缓存 TTL 细分统计（5m/1h）
- **Messages API 缓存 TTL 细分统计** - 区分 5 分钟和 1 小时 TTL 的缓存创建统计

### 🔨 重构

- **SQLite 驱动切换为纯 Go 实现** - 从 `go-sqlite3`（CGO）切换为 `modernc.org/sqlite`，简化交叉编译

### 🐛 修复

- **Usage 解析数值类型健壮性** - 支持 `float64`/`int`/`int64`/`int32` 四种数值类型
- **CachedTokens 重复计算** - `CachedTokens` 仅包含 `cache_read`，不再包含 `cache_creation`
- **流式响应纯缓存场景 Usage 丢失** - 有任何 usage 字段时都记录

---

## [v2.2.0] - 2025-12-21

### 🔨 重构

- **Handlers 模块重构为同级子包结构** - 将 Messages/Responses API 处理器重构为同级模块，新增 `handlers/common/` 公共包，代码量减少约 180 行

### 🐛 修复

- **Stream 错误处理完善** - 流式传输错误时发送 SSE 错误事件并记录失败指标
- **CountTokens 端点安全加固** - 应用请求体大小限制
- **非 failover 错误指标记录** - 400/401/403 等错误正确记录失败指标

---

## [v2.1.35] - 2025-12-21

- **流量图表失败率可视化** - 失败率超过 10% 显示红色背景，Tooltip 显示详情

---

## [v2.1.34] - 2025-12-20

- **Key 级别使用趋势图表** - 支持流量/Token I/O/缓存三种视图，智能 Key 筛选
- **合并 Dashboard API** - 3 个并行请求优化为 1 个

---

## [v2.1.33] - 2025-12-20

- **Fuzzy Mode 错误处理开关** - 所有非 2xx 错误自动触发 failover
- **渠道指标历史数据 API** - 支持时间序列图表

---

## [v2.1.25] - 2025-12-18

### ✨ 新功能

- **TransformerMetadata 和 CacheControl 支持** - 转换器元数据保留原始格式信息，实现特性透传
- **FinishReason 统一映射函数** - OpenAI/Anthropic/Responses 三种协议间双向映射
- **原始日志输出开关** - `RAW_LOG_OUTPUT` 环境变量，开启后不进行格式化或截断

---

## [v2.1.23] - 2025-12-13

- 修复编辑渠道弹窗中基础 URL 布局和验证问题

---

## [v2.1.31] - 2025-12-19

- **前端显示版本号和更新检查** - 自动检查 GitHub 最新版本

---

## [v2.1.30] - 2025-12-19

- **强制探测模式** - 所有 Key 熔断时自动启用强制探测

---

## [v2.1.28] - 2025-12-19

- **BaseURL 支持 `#` 结尾跳过自动添加 `/v1`**

---

## [v2.1.27] - 2025-12-19

- 移除 Claude Provider 畸形 tool_call 修复逻辑

---

## [v2.1.26] - 2025-12-19

- Responses 渠道新增 `gpt-5.2-codex` 模型选项

---

## [v2.1.24] - 2025-12-17

- Responses 渠道新增 `gpt-5.2`、`gpt-5` 模型选项
- 移除 openaiold 服务类型支持

---

## [v2.1.23] - 2025-12-13

- 修复 402 状态码未触发 failover 的问题
- 重构 HTTP 状态码 failover 判断逻辑（两层分类策略）

---

## [v2.1.22] - 2025-12-13

### 🐛 修复

- **流式日志合成器类型修复** - 所有 Provider 的 HandleStreamResponse 都将响应转换为 Claude SSE 格式，日志合成器使用 "claude" 类型解析
- **insecureSkipVerify 字段提交修复** - 修复前端 insecureSkipVerify 为 false 时不提交的问题

---

## [v2.1.21] - 2025-12-13

### 🐛 修复

- **促销渠道绕过健康检查** - 促销渠道现在绕过健康检查直接尝试使用，只有本次请求实际失败后才跳过

---

## [v2.1.20] - 2025-12-12

- 渠道名称支持点击打开编辑弹窗

---

## [v2.1.19] - 2025-12-12

- 修复添加渠道弹窗密钥重复错误状态残留
- 新增 `/v1/responses/compact` 端点

---

## [v2.1.15] - 2025-12-12

### 🔒 安全加固

- **请求体大小限制** - 新增 `MAX_REQUEST_BODY_SIZE_MB` 环境变量（默认 50MB），超限返回 413
- **Goroutine 泄漏修复** - ConfigManager 添加 `stopChan` 和 `Close()` 方法释放资源
- **数据竞争修复** - 负载均衡计数器改用 `sync/atomic` 原子操作
- **优雅关闭** - 监听 SIGINT/SIGTERM，10 秒超时优雅关闭

---

## [v2.1.14] - 2025-12-12

- 修复流式响应 Token 计数中间更新被覆盖

---

## [v2.1.12] - 2025-12-11

- 支持 Claude 缓存 Token 计数

---

## [v2.1.10] - 2025-12-11

- 修复流式响应 Token 计数补全逻辑

---

## [v2.1.8] - 2025-12-11

- 重构过长方法，提升代码可读性

---

## [v2.1.7] - 2025-12-11

### 🐛 修复

- 修复前端 MDI 图标无法显示
- **Token 计数补全虚假值处理** - 当 `input_tokens <= 1` 或 `output_tokens == 0` 时用本地估算值覆盖

---

## [v2.1.6] - 2025-12-11

### ✨ 新功能

- **Messages API Token 计数补全** - 当上游不返回 usage 时，本地估算 token 数量并附加到响应中

---

## [v2.1.4] - 2025-12-11

- 修复前端渠道健康度统计不显示数据

---

## [v2.1.1] - 2025-12-11

- 新增 `QUIET_POLLING_LOGS` 环境变量（默认 true），过滤前端轮询日志噪音

---

## [v2.1.0] - 2025-12-11

### 🔨 重构

- **指标系统重构：Key 级别绑定** - 指标键改为 `hash(baseURL + apiKey)`，每个 Key 独立追踪
- **熔断器生效修复** - 在 `tryChannelWithAllKeys` 中调用 `ShouldSuspendKey()` 跳过熔断的 Key
- **单渠道路径指标记录** - 转换失败、发送失败、failover、成功时正确记录指标

---

## [v2.0.20-go] - 2025-12-08

- 修复单渠道模式渠道选择逻辑

---

## [v2.0.11-go] - 2025-12-06

### 🚀 多渠道智能调度器

- **ChannelScheduler** - 基于优先级的渠道选择、Trace 亲和性、失败率检测和自动熔断
- **MetricsManager** - 滑动窗口算法计算实时成功率
- **TraceAffinityManager** - 用户会话与渠道绑定

### 🎨 渠道编排面板

- 拖拽排序、实时指标、状态切换、备用池管理

---

## [v2.0.10-go] - 2025-12-06

### 🎨 复古像素主题

- Neo-Brutalism 设计语言：无圆角、等宽字体、粗实体边框、硬阴影

---

## [v2.0.5-go] - 2025-11-15

### 🚀 Responses API 转换器架构重构

- 策略模式 + 工厂模式实现多上游转换器
- 完整支持 Responses API 标准格式

---

## [v2.0.4-go] - 2025-11-14

### ✨ Responses API 透明转发

- Codex Responses API 端点 (`/v1/responses`)
- 会话管理系统（多轮对话跟踪）
- Messages API 多上游协议支持（Claude/OpenAI/Gemini）

---

## [v2.0.0-go] - 2025-10-15

### 🎉 Go 语言重写版本

- **性能提升**: 启动速度 20x，内存占用 -70%
- **单文件部署**: 前端资源嵌入二进制
- **完整功能移植**: 所有上游适配器、协议转换、流式响应、配置热重载

---

## 历史版本

<details>
<summary>v1.x TypeScript 版本</summary>

### v1.2.0 - 2025-09-19
- Web 管理界面、模型映射、渠道置顶、API 密钥故障转移

### v1.1.0 - 2025-09-17
- SSE 数据解析优化、Bearer Token 处理简化、代码重构

### v1.0.0 - 2025-09-13
- 初始版本：多上游支持、负载均衡、配置管理

</details>
