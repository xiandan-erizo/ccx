# Deployment

## Docker Compose

```yaml
version: '3.8'
services:
  ccx:
    image: ghcr.io/benedictking/ccx:latest
    ports:
      - "3000:3000"
    volumes:
      - ./config:/app/.config
    environment:
      - PROXY_ACCESS_KEY=your-proxy-key
      - ADMIN_ACCESS_KEY=your-admin-key
      - ENV=production
    restart: unless-stopped
```

## System Service

### Linux (systemd)

```ini
[Unit]
Description=CCX AI API Gateway
After=network.target

[Service]
Type=simple
ExecStart=/opt/ccx/ccx
WorkingDirectory=/opt/ccx
Environment=PROXY_ACCESS_KEY=your-proxy-key
Environment=ADMIN_ACCESS_KEY=your-admin-key
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### macOS (launchd)

See `docs/service/com.ccx.gateway.plist` for reference.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 3000 | Server port |
| `ENV` | production | Runtime environment |
| `PROXY_ACCESS_KEY` | - | Proxy access key (required) |
| `ADMIN_ACCESS_KEY` | - | Admin console key (optional) |
| `QUIET_POLLING_LOGS` | true | Suppress polling logs |
| `MAX_REQUEST_BODY_SIZE_MB` | 50 | Max request body size |

See `ENVIRONMENT.md` in the project root for the full list.
