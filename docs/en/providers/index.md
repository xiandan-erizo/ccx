# Provider Setup

This section guides you through configuring various LLM providers in CCX.

## General Workflow

1. Log in to the CCX admin console (default `http://localhost:3000`)
2. Select the appropriate proxy endpoint (Chat, Messages, etc.)
3. Click "Add Channel"
4. Fill in the channel configuration
5. Save and test

## Key Configuration Fields

| Field | Description |
|-------|-------------|
| **Name** | Display name for the channel |
| **Service Type** | Upstream API protocol: `openai`, `claude`, `gemini`, `responses` |
| **Base URL** | Upstream API endpoint |
| **API Keys** | Authentication keys (supports multi-key rotation) |
| **Supported Models** | Model allowlist for this channel |
| **Model Mapping** | Map request model names to upstream model names |
| **Priority** | Lower number = higher priority |

## Service Type Guide

| Provider | Chat Endpoint | Messages Endpoint | Notes |
|----------|---------------|-------------------|-------|
| DeepSeek | `openai` / `https://api.deepseek.com` | `claude` / `https://api.deepseek.com/anthropic` | Both protocols |
| GLM (Zhipu) | `openai` / `https://open.bigmodel.cn/api/paas/v4` | `claude` / `https://open.bigmodel.cn/api/anthropic` | Both protocols |
| MiniMax | `openai` / `https://api.minimax.io/v1` | `claude` / `https://api.minimax.io/anthropic` | Both protocols |
| Kimi | `openai` / `https://api.moonshot.ai/v1` | `claude` / `https://api.kimi.com/coding/` | Coding endpoint |
| OpenAI GPT | `openai` / `https://api.openai.com/v1` | — | OpenAI only |
| Xiaomi MiMo | `openai` / `https://api.siliconflow.cn/v1` | — | Via SiliconFlow |
| Claude | `claude` (translation) | `claude` / `https://api.anthropic.com` | Native Messages |
| Gemini | `openai` or `gemini` | — | OpenAI-compat and native |

::: tip
Most Chinese LLM providers now support both OpenAI Chat and Anthropic Messages protocols. If you use Claude Code CLI, you can configure the Anthropic-compatible endpoint directly under the Messages endpoint.
:::

## Provider Guides

- [DeepSeek](./deepseek)
- [GLM (Zhipu AI)](./glm)
- [MiniMax](./minimax)
- [Kimi (Moonshot)](./kimi)
- [OpenAI GPT](./openai)
- [Xiaomi MiMo](./mimo)
- [Claude](./claude)
- [Gemini](./gemini)
