# GLM (Zhipu AI / Z.AI) Setup

## Get API Key

1. Visit [Zhipu AI Platform](https://open.bigmodel.cn/) (rebranded as [Z.AI](https://z.ai))
2. Sign up and log in
3. Go to "API Keys" page
4. Create a new API Key

## Add Channel in CCX

### Option 1: Chat Endpoint (OpenAI Compatible)

| Field | Value |
|-------|-------|
| Name | `GLM` |
| Service Type | `openai` |
| Base URL | `https://open.bigmodel.cn/api/paas/v4` |
| API Keys | Your Zhipu API Key |

### Option 2: Messages Endpoint (Anthropic Compatible)

For Claude Code CLI and other tools using Claude Messages protocol.

| Field | Value |
|-------|-------|
| Name | `GLM Claude` |
| Service Type | `claude` |
| Base URL | `https://open.bigmodel.cn/api/anthropic` |
| API Keys | Your Zhipu API Key |

#### Model Mapping (Recommended for Messages)

| Request Model | Redirects To |
|---------------|--------------|
| `opus` | `glm-5.1` |
| `sonnet` | `glm-5` |
| `haiku` | `glm-5-turbo` |

## Available Models

| Model | Description |
|-------|-------------|
| `glm-5.1` | Latest flagship, Agentic engineering, 744B params |
| `glm-5` | Previous flagship |
| `glm-5-turbo` | Fast version |
| `glm-4.7` | MoE architecture |
| `glm-4.6` | Multimodal (vision support) |
| `glm-4.5` | MoE, 106B total / 12B active |

## Notes

- Zhipu API supports both OpenAI Chat and Anthropic Messages protocols
- OpenAI-compatible Base URL uses `/v4` path — do not append `/chat/completions`
- Anthropic-compatible Base URL: `https://open.bigmodel.cn/api/anthropic`
- API Key format is `xxxxxxxx.yyyyyyyy` (the entire string is the key)
- Zhipu has rebranded to Z.AI, but `open.bigmodel.cn` endpoints remain active
