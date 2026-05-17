# Kimi (月之暗面) 配置指南

## 获取 API Key

1. 访问 [Kimi 开放平台](https://platform.kimi.ai/)
2. 注册并登录账号
3. 进入「API Key 管理」页面
4. 创建新的 API Key 并复制

## 在 CCX 中添加渠道

### 方式一：Chat 入口（OpenAI 兼容协议）

| 字段 | 值 |
|------|-----|
| 名称 | `Kimi`（自定义） |
| 服务类型 | `openai` |
| Base URL | `https://api.moonshot.ai/v1` |
| API Keys | 你的 Moonshot API Key |

#### 配置步骤

1. 进入 CCX 管理界面，选择 **Chat** 入口
2. 点击「添加渠道」
3. 填写以下信息：
   - **名称**：`Kimi`
   - **服务类型**：选择 `OpenAI Chat`
   - **Base URL**：`https://api.moonshot.ai/v1`
   - **API Keys**：粘贴你的 API Key
4. 点击保存

### 方式二：Messages 入口（编码专用端点）

适用于 Claude Code CLI 等使用 Claude Messages 协议的工具。

| 字段 | 值 |
|------|-----|
| 名称 | `Kimi Coding`（自定义） |
| 服务类型 | `claude` |
| Base URL | `https://api.kimi.com/coding/` |
| API Keys | 你的 Kimi API Key |

#### 模型映射（Messages 入口推荐）

| 请求模型 | 重定向到 |
|----------|----------|
| `opus` | `kimi-k2.6` |
| `sonnet` | `kimi-k2.6` |
| `haiku` | `kimi-k2.5` |

### 模型白名单（可选）

```
kimi-k2.6
kimi-k2.5
moonshot-v1-auto
moonshot-v1-128k
```

## 可用模型

| 模型 | 说明 |
|------|------|
| `kimi-k2.6` | 最新推荐，原生多模态 Agentic 模型，1T 总参 / 32B 激活 |
| `kimi-k2.5` | 多模态 Agentic 模型 |
| `moonshot-v1-auto` | 自动选择上下文长度（旧一代） |
| `moonshot-v1-128k` | 128K 上下文（旧一代） |

::: warning 模型停服通知
`kimi-k2` 将于 **2026/05/25** 停服，请迁移到 `kimi-k2.6`。
:::

## 注意事项

- Kimi OpenAI 兼容 API 的 Base URL 为 `https://api.moonshot.ai/v1`（注意是 `moonshot.ai` 不是 `moonshot.cn`）
- 编码专用端点为 `https://api.kimi.com/coding/`，适合 Claude Code CLI 场景
- `kimi-k2.6` 是当前最新推荐模型，支持长上下文编码
- Kimi 支持联网搜索等扩展功能
