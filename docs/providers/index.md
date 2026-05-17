# 配置教程

本节将指导你如何在 CCX 中配置各个 LLM 提供商的渠道。

## 通用配置流程

无论使用哪个提供商，添加渠道的基本步骤都是一致的：

1. 登录 CCX 管理界面（默认 `http://localhost:3000`）
2. 选择对应的代理入口（如 Chat、Messages 等）
3. 点击「添加渠道」
4. 填写渠道配置信息
5. 保存并测试

## 关键配置字段说明

| 字段 | 说明 |
|------|------|
| **名称** | 渠道的显示名称，便于识别 |
| **服务类型** | 上游 API 的协议类型：`openai`、`claude`、`gemini`、`responses` |
| **Base URL** | 上游 API 的地址 |
| **API Keys** | 上游服务的认证密钥，支持多 Key 轮转 |
| **模型白名单** | 限制该渠道可用的模型列表 |
| **模型映射** | 将请求中的模型名映射为上游实际模型名 |
| **优先级** | 数字越小优先级越高 |

## 服务类型选择指南

| 提供商 | Chat 入口 | Messages 入口 | 说明 |
|--------|-----------|---------------|------|
| DeepSeek | `openai` / `https://api.deepseek.com` | `claude` / `https://api.deepseek.com/anthropic` | 同时支持两种协议 |
| 智谱 GLM | `openai` / `https://open.bigmodel.cn/api/paas/v4` | `claude` / `https://open.bigmodel.cn/api/anthropic` | 同时支持两种协议 |
| MiniMax | `openai` / `https://api.minimax.io/v1` | `claude` / `https://api.minimax.io/anthropic` | 同时支持两种协议 |
| Kimi | `openai` / `https://api.moonshot.ai/v1` | `claude` / `https://api.kimi.com/coding/` | 编码专用端点 |
| OpenAI GPT | `openai` / `https://api.openai.com/v1` | — | 仅 OpenAI 协议 |
| 小米 MiMo | `openai` / `https://api.siliconflow.cn/v1` | — | 通过硅基流动访问 |
| Claude | `claude`（协议转换） | `claude` / `https://api.anthropic.com` | 原生 Messages 协议 |
| Gemini | `openai` 或 `gemini` | — | 支持 OpenAI 兼容和原生协议 |

::: tip
大多数国产 LLM 提供商现在同时兼容 OpenAI Chat 和 Anthropic Messages 协议。如果你使用 Claude Code CLI，可以直接在 Messages 入口配置 Anthropic 兼容端点。
:::

## 提供商配置指南

- [DeepSeek](./deepseek)
- [智谱 GLM](./glm)
- [MiniMax](./minimax)
- [Kimi (月之暗面)](./kimi)
- [OpenAI GPT](./openai)
- [小米 MiMo](./mimo)
- [Claude](./claude)
- [Gemini](./gemini)
