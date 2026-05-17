# Claude Setup

## Get API Key

1. Visit [Anthropic Console](https://console.anthropic.com/)
2. Log in
3. Go to "API Keys" page
4. Create a new API Key

## Add Channel in CCX

### Messages Endpoint (Recommended)

| Field | Value |
|-------|-------|
| Name | `Claude` |
| Service Type | `claude` |
| Base URL | `https://api.anthropic.com` |
| API Keys | Your Anthropic API Key (`sk-ant-...`) |

### Steps

1. Go to CCX admin console, select **Messages** endpoint
2. Click "Add Channel"
3. Set service type to `Claude`
4. Set Base URL to `https://api.anthropic.com`
5. Paste your API Key
6. Save

### Chat Endpoint (Protocol Translation)

If your client uses OpenAI Chat protocol, you can add a Claude channel under the Chat endpoint — CCX handles protocol translation automatically:

1. Select **Chat** endpoint
2. Set service type to `claude`
3. Same Base URL and API Key

## Available Models

| Model | Description |
|-------|-------------|
| `claude-opus-4-7` | Opus 4.7, most capable |
| `claude-sonnet-4-6` | Sonnet 4.6, balanced |
| `claude-haiku-4-5-20251001` | Haiku 4.5, fast |
| `claude-3-5-sonnet-20241022` | Claude 3.5 Sonnet |

## Notes

- Claude API Keys start with `sk-ant-`
- Messages endpoint uses Claude native protocol with full feature support (thinking, tool_use, etc.)
- Chat endpoint supports Claude via protocol translation — some advanced features may be limited
- Use the "Proxy URL" field if you need a proxy to access Anthropic API
