# CHANGELOG

## [Unreleased]

### Added

- 新增内置 OTA 更新能力：后端提供 `/api/system/update/check` 与 `/api/system/update/apply` 管理接口，支持 GitHub Release 版本检查、SHA256 校验、二进制替换备份与 Docker 环境禁用升级提示。
- 新增前端系统更新对话框，版本徽标优先通过后端检查更新，失败时保留 GitHub 直连降级路径，并支持升级后健康检查轮询。
- 发布工作流为 Linux、macOS、Windows 各平台资产生成并上传独立 `.sha256` 校验文件。

### Fixed

- 修复 Responses 转 Chat 时孤儿 reasoning 生成 `content:null` 的 assistant 消息，避免 Codex 停止生成后继续输入触发 DeepSeek `Invalid assistant message: content or tool_calls must be set` 错误。
- 修复 Responses 转 Chat 时缺少 `type` 但包含 `role/content` 的输入消息被丢弃的问题，避免 Codex 简化 input 触发上游 `messages` 异常。
- 修复公共 `/v1/models` 与 `/v1/models/:model` 未纳入 Chat 渠道的问题，统一按 `messages → responses → chat` 聚合与回退模型查询，并保留 routePrefix 与已拉黑 key fallback 语义。
- 补充 `/v1/models` Chat 聚合与模型详情回退回归测试，覆盖去重优先级、routePrefix 与已拉黑 key fallback 行为。

- 修复 capability-test 在取消后恢复旧任务时返回过期的 `cancelled` job 快照，避免前端误判任务已结束而停止轮询。
- 为 capability-test 增加取消后恢复场景的 HTTP 回归测试，覆盖恢复响应状态正确性。
- 将 capability-test 的限速、共享结果与运行复用收敛到 upstream identity 维度，并新增 shared snapshot API 与单协议测试交互提示。
- 修复 capability-test 的 `chat` 与 `responses(codex)` 协议默认探测模型顺序不一致问题，统一将 `gpt-5.5` 提升为首位，并同步前端占位模型列表与后端探测配置。
