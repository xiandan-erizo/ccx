# Kimi (Moonshot) Setup

## Get API Key

1. Visit [Kimi Platform](https://platform.kimi.ai/)
2. Sign up and log in
3. Go to "API Key Management" page
4. Create a new API Key

## Add Channel in CCX

### Option 1: Chat Endpoint (OpenAI Compatible)

| Field | Value |
|-------|-------|
| Name | `Kimi` |
| Service Type | `openai` |
| Base URL | `https://api.moonshot.ai/v1` |
| API Keys | Your Moonshot API Key |

### Option 2: Messages Endpoint (Coding Endpoint)

For Claude Code CLI and other tools using Claude Messages protocol.

| Field | Value |
|-------|-------|
| Name | `Kimi Coding` |
| Service Type | `claude` |
| Base URL | `https://api.kimi.com/coding/` |
| API Keys | Your Kimi API Key |

#### Model Mapping (Recommended for Messages)

| Request Model | Redirects To |
|---------------|--------------|
| `opus` | `kimi-k2.6` |
| `sonnet` | `kimi-k2.6` |
| `haiku` | `kimi-k2.5` |

## Available Models

| Model | Description |
|-------|-------------|
| `kimi-k2.6` | Latest, native multimodal Agentic model, 1T total / 32B active |
| `kimi-k2.5` | Multimodal Agentic model |
| `moonshot-v1-auto` | Auto-selects context length (legacy) |
| `moonshot-v1-128k` | 128K context (legacy) |

::: warning Deprecation Notice
`kimi-k2` will be retired on **2026/05/25**. Please migrate to `kimi-k2.6`.
:::

## Notes

- Kimi OpenAI-compatible Base URL is `https://api.moonshot.ai/v1` (note: `moonshot.ai` not `moonshot.cn`)
- Coding endpoint is `https://api.kimi.com/coding/`, suitable for Claude Code CLI
- `kimi-k2.6` is the latest recommended model with long-context coding support
