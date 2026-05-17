# MiniMax Setup

## Get API Key

1. Visit [MiniMax Platform](https://platform.minimax.io/)
2. Sign up and log in
3. Go to "API Keys" page
4. Create a new API Key

## Add Channel in CCX

### Option 1: Chat Endpoint (OpenAI Compatible)

| Field | Value |
|-------|-------|
| Name | `MiniMax` |
| Service Type | `openai` |
| Base URL | `https://api.minimax.io/v1` |
| API Keys | Your MiniMax API Key |

### Option 2: Messages Endpoint (Anthropic Compatible)

For Claude Code CLI and other tools using Claude Messages protocol.

| Field | Value |
|-------|-------|
| Name | `MiniMax Claude` |
| Service Type | `claude` |
| Base URL | `https://api.minimax.io/anthropic` |
| API Keys | Your MiniMax API Key |

#### Model Mapping (Recommended for Messages)

| Request Model | Redirects To |
|---------------|--------------|
| `opus` | `MiniMax-M2.7` |
| `sonnet` | `MiniMax-M2.7` |
| `haiku` | `MiniMax-M2.7-highspeed` |

## Available Models

| Model | Description |
|-------|-------------|
| `MiniMax-M2.7` | Latest flagship, 230B MoE with self-evolution |
| `MiniMax-M2.7-highspeed` | High-speed version |

## Notes

- MiniMax supports both OpenAI Chat and Anthropic Messages protocols
- Official domain is now `api.minimax.io` (old domain `api.minimaxi.com` may still work)
- OpenAI-compatible Base URL includes `/v1`
- Anthropic-compatible Base URL: `https://api.minimax.io/anthropic`
