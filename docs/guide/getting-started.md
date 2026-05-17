# 快速开始

## 安装

### Docker 部署（推荐）

```bash
docker run -d \
  --name ccx \
  -p 3000:3000 \
  -v ./config:/app/.config \
  -e PROXY_ACCESS_KEY=your-proxy-key \
  -e ADMIN_ACCESS_KEY=your-admin-key \
  ghcr.io/benedictking/ccx:latest
```

### 二进制部署

从 [GitHub Releases](https://github.com/BenedictKing/ccx/releases) 下载对应平台的二进制文件，然后运行：

```bash
export PROXY_ACCESS_KEY=your-proxy-key
export ADMIN_ACCESS_KEY=your-admin-key
./ccx
```

服务默认监听 `http://localhost:3000`。

## 基本概念

### 渠道 (Channel)

渠道是 CCX 的核心概念，每个渠道对应一个上游 API 的配置，包括：

- **API Key**：上游服务的认证密钥
- **Base URL**：上游 API 的地址
- **模型列表**：该渠道支持的模型
- **优先级**：调度时的优先级权重

### 五类代理入口

| 入口 | 路径 | 说明 |
|------|------|------|
| Claude Messages | `/v1/messages` | Claude 原生协议 |
| OpenAI Chat | `/v1/chat/completions` | OpenAI Chat 协议 |
| Codex Responses | `/v1/responses` | OpenAI Responses 协议 |
| Gemini | `/v1beta/models/*` | Gemini 原生协议 |
| Images | `/v1/images/*` | OpenAI Images 协议 |

## 访问管理界面

启动后访问 `http://localhost:3000`，使用 `ADMIN_ACCESS_KEY` 登录管理界面。

![渠道管理界面](/images/guide/channel-list.png)

在管理界面中，你可以：
- 添加和管理渠道
- 查看请求日志和流量统计
- 测试渠道连通性
- 调整渠道优先级

## 下一步

前往 [配置教程](/providers/) 了解如何配置各个 LLM 提供商。
