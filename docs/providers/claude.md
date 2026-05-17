# Claude 配置指南

## 获取 API Key

1. 访问 [Anthropic Console](https://console.anthropic.com/)
2. 登录账号
3. 进入「API Keys」页面
4. 创建新的 API Key 并复制

## 在 CCX 中添加渠道

### Messages 入口（推荐）

Claude 原生协议使用 Messages 入口：

| 字段 | 值 |
|------|-----|
| 名称 | `Claude`（自定义） |
| 服务类型 | `claude` |
| Base URL | `https://api.anthropic.com` |
| API Keys | 你的 Anthropic API Key（`sk-ant-...`） |

### 配置步骤

1. 进入 CCX 管理界面，选择 **Messages** 入口
2. 点击「添加渠道」
3. 填写以下信息：
   - **名称**：`Claude`
   - **服务类型**：选择 `Claude`
   - **Base URL**：`https://api.anthropic.com`
   - **API Keys**：粘贴你的 API Key
4. 点击保存

### Chat 入口（协议转换）

如果你的客户端使用 OpenAI Chat 协议，也可以在 Chat 入口添加 Claude 渠道，CCX 会自动进行协议转换：

1. 选择 **Chat** 入口
2. 服务类型选择 `claude`
3. 其余配置相同

### 模型白名单（可选）

```
claude-sonnet-4-6
claude-opus-4-7
claude-haiku-4-5-20251001
claude-3-5-sonnet-20241022
```

## 可用模型

| 模型 | 说明 |
|------|------|
| `claude-opus-4-7` | Opus 4.7，最强能力 |
| `claude-sonnet-4-6` | Sonnet 4.6，均衡性能 |
| `claude-haiku-4-5-20251001` | Haiku 4.5，快速响应 |
| `claude-3-5-sonnet-20241022` | Claude 3.5 Sonnet |

## 注意事项

- Claude API Key 以 `sk-ant-` 开头
- Messages 入口使用 Claude 原生协议，支持所有 Claude 特性（thinking、tool_use 等）
- Chat 入口通过协议转换支持 Claude，部分高级特性可能受限
- 如需使用代理访问，在「代理地址」字段填写代理 URL
