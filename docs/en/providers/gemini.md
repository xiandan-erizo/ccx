# Gemini Setup

## Get API Key

1. Visit [Google AI Studio](https://aistudio.google.com/)
2. Log in with your Google account
3. Click "Get API Key"
4. Create a new API Key

## Add Channel in CCX

### Gemini Endpoint (Native Protocol)

| Field | Value |
|-------|-------|
| Name | `Gemini` |
| Service Type | `gemini` |
| Base URL | `https://generativelanguage.googleapis.com` |
| API Keys | Your Google AI API Key |

### Steps

1. Go to CCX admin console, select **Gemini** endpoint
2. Click "Add Channel"
3. Set service type to `Gemini`
4. Set Base URL to `https://generativelanguage.googleapis.com`
5. Paste your API Key
6. Save

### Chat Endpoint (OpenAI Compatible)

Gemini can also be accessed via OpenAI-compatible interface:

| Field | Value |
|-------|-------|
| Name | `Gemini (OpenAI)` |
| Service Type | `openai` |
| Base URL | `https://generativelanguage.googleapis.com/v1beta/openai` |
| API Keys | Your Google AI API Key |

## Available Models

| Model | Description |
|-------|-------------|
| `gemini-2.5-pro` | Latest flagship with thinking |
| `gemini-2.5-flash` | Fast version with thinking |
| `gemini-2.0-flash` | High-speed multimodal |
| `gemini-1.5-pro` | Ultra-long context (2M tokens) |

## Advanced Configuration

### thought_signature Handling

Some third-party Gemini proxies require `thought_signature` in requests:

- **Inject dummy thought_signature**: Enable `injectDummyThoughtSignature`
- **Strip thought_signature**: Enable `stripThoughtSignature` (for older APIs)

## Notes

- Gemini native endpoint uses Google's generateContent protocol
- For OpenAI Chat protocol access, use Chat endpoint + `openai` service type + OpenAI-compatible Base URL
- Servers in mainland China need a proxy to access Google APIs
- Vision capabilities are automatically supported via native protocol
