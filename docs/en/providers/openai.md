# OpenAI GPT Setup

## Get API Key

1. Visit [OpenAI Platform](https://platform.openai.com/)
2. Log in
3. Go to "API Keys" page
4. Click "Create new secret key"

## Add Channel in CCX

| Field | Value |
|-------|-------|
| Name | `OpenAI` |
| Service Type | `openai` |
| Base URL | `https://api.openai.com/v1` |
| API Keys | Your OpenAI API Key (`sk-...`) |

### Steps

1. Go to CCX admin console, select **Chat** endpoint
2. Click "Add Channel"
3. Set service type to `OpenAI Chat`
4. Set Base URL to `https://api.openai.com/v1`
5. Paste your API Key
6. Save

## Available Models

| Model | Description |
|-------|-------------|
| `gpt-4o` | GPT-4o multimodal flagship |
| `gpt-4o-mini` | Lightweight cost-effective version |
| `o1` | Reasoning model |
| `o3` | Latest reasoning model |
| `o3-mini` | Lightweight reasoning model |
| `o4-mini` | Latest lightweight reasoning model |

## Images Endpoint

For image generation:

1. Select **Images** endpoint
2. Add channel with same Base URL and API Key
3. Service type: `openai`

Supported models: `dall-e-3`, `dall-e-2`, `gpt-image-1`

## Responses Endpoint

For Codex/Responses protocol:

1. Select **Responses** endpoint
2. Add channel with same Base URL and API Key
3. Service type: `responses`

## Notes

- OpenAI API Keys start with `sk-`
- If using a proxy, fill in the "Proxy URL" field
- Reasoning models (o1/o3 series) don't support system messages — CCX handles compatibility automatically
