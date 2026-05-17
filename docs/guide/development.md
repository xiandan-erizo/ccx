# 开发指南

本文档说明 CCX 的本地开发方式、常用命令和验证流程。

> 相关文档：
> - 架构设计：`ARCHITECTURE.md`
> - 环境变量：`ENVIRONMENT.md`
> - 发布流程：`RELEASE.md`

## 推荐开发方式

| 方式 | 适用场景 | 说明 |
| --- | --- | --- |
| 根目录 Make | 日常联调 | 同时启动前端开发服务器和后端热重载 |
| backend-go Make | 后端专项开发 | 只运行 Go 后端命令 |
| frontend Bun | 前端专项开发 | 只运行 Vue/Vite 前端命令 |
| Docker | 接近生产环境验证 | 不适合作为主要热重载开发方式 |

## 方式一：根目录开发（推荐）

根目录 `Makefile` 是联调开发的事实来源。

```bash
make help
make dev
make run
make frontend-dev
make build
make clean
```

说明：
- `make dev`：启动前端开发服务器，并在 `backend-go/` 下以热重载模式运行 Go 后端
- `make run`：构建前端并运行后端
- `make build`：构建前端并编译 Go 后端
- `make frontend-dev`：仅启动前端开发服务器

## 方式二：backend-go 目录开发

`backend-go/Makefile` 是后端命令的事实来源。

```bash
cd "backend-go"
make help
make dev
make run
make build
make test
make test-cover
make fmt
make lint
make deps
```

说明：
- `make dev`：使用 Air 热重载
- `make run`：复制前端构建产物后直接运行
- `make build`：构建生产二进制到 `dist/`
- `make test`：运行所有 Go 测试
- `make test-cover`：生成覆盖率报告

## 方式三：frontend 目录开发

前端脚本以 `frontend/package.json` 为准，优先使用 Bun。

```bash
cd "frontend"
bun install
bun run dev
bun run build
bun run preview
bun run type-check
bun run lint
```

如果本地没有 Bun，可改用 `npm install` / `npm run <script>` 作为兼容方案。

## Windows 环境建议

如果没有 `make`，可分别使用 Go 和 Bun 命令：

```powershell
cd backend-go
air
go test ./...
go fmt ./...

cd ../frontend
bun install
bun run dev
```

推荐安装：
- Go
- Bun
- Make
- Git

## 文件变更与重载规则

### 自动热重载

- Go 源码：`make dev` / `cd "backend-go" && make dev` 下自动重载
- 配置文件：`backend-go/.config/config.json` 修改后自动生效

### 需要重启

- 环境变量文件：`backend-go/.env`
- 依赖或构建配置变更

## 常用验证命令

在提交改动前，至少执行以下检查：

```bash
make build
cd "backend-go" && make test
cd "frontend" && bun run build
```

如果只改后端代码，建议额外执行：

```bash
cd "backend-go" && make lint
```

## 本地访问入口

- Web 管理界面：`http://localhost:3000`
- 代理 API：`http://localhost:3000/v1`
- 健康检查：`http://localhost:3000/health`
- 前端开发服务器：默认 `http://localhost:5173`

### Windows / WSL / Docker 访问建议

Windows 下 cmd、PowerShell、WSL 与 Docker 可能处在不同网络环境中，`localhost` / `127.0.0.1` 不一定指向运行 CCX 的进程。跨环境调用时建议使用 Windows 主机的局域网 IPv4 地址，例如：

```text
http://192.168.1.23:3000/v1
```

获取地址：

```powershell
ipconfig
```

验证连通性：

```powershell
curl.exe -i http://192.168.1.23:3000/health
```

CCX 后端默认监听 `:PORT`，等价于监听所有网卡地址；通常不需要额外修改为 `0.0.0.0`。如果局域网 IP 无法访问，优先检查 Windows 防火墙是否允许对应端口入站。

## 常见开发任务

### 只调试后端

```bash
cd "backend-go"
make dev
```

### 只调试前端

```bash
cd "frontend"
bun install
bun run dev
```

### 前后端联调

```bash
make dev
```

### 验证生产构建

```bash
make build
```
