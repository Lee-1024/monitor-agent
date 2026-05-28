# Monitor Agent

监控系统数据采集代理，部署在被监控主机上，负责采集系统指标、日志、进程和服务状态，并通过gRPC上报到后端服务。

## 📋 目录

- [概述](#概述)
- [功能特性](#功能特性)
- [技术栈](#技术栈)
- [快速开始](#快速开始)
- [项目结构](#项目结构)
- [配置说明](#配置说明)
- [使用指南](#使用指南)
- [开发指南](#开发指南)
- [部署](#部署)

## 🎯 概述

Monitor Agent 是监控系统的数据采集组件，部署在被监控的主机上，负责：

- 采集系统指标（CPU、内存、磁盘、网络）
- 采集 GPU 指标（如主机存在可用显卡）
- 收集系统日志和应用日志
- 监控进程资源使用
- 监控系统服务状态
- 执行远程脚本
- 通过gRPC将数据上报到Backend

## ✨ 功能特性

### 核心功能

- ✅ **指标采集**: CPU、内存、磁盘、网络等系统指标实时采集
- ✅ **GPU采集**: 采集 GPU 设备、使用率、显存、温度、功耗等指标
- ✅ **日志收集**: 支持多文件日志收集，自动识别日志级别
- ✅ **进程监控**: 进程资源使用监控和Top进程识别
- ✅ **服务监控**: 系统服务状态监控（支持Linux systemd和Windows服务）
- ✅ **脚本执行**: 远程脚本执行和结果记录
- ✅ **数据上报**: 通过gRPC实时上报数据到Backend

### 详细功能

1. **CPU采集**: 使用率、负载、核心数
2. **内存采集**: 使用率、总量、可用量
3. **磁盘采集**: 使用率、分区信息、IO统计
4. **网络采集**: 流量统计、接口信息
5. **GPU采集**: 设备列表、厂商、型号、显存、使用率、温度、功耗
6. **日志收集**: 支持多文件、自动级别识别
7. **进程监控**: 进程列表、资源使用、Top进程
8. **服务监控**: 服务状态、自启动配置、端口可访问性
9. **脚本执行**: Shell/Python/系统命令执行

## 当前采集与告警关系

### 采集周期

默认采集周期为 `10s`。Agent 每次成功上报指标、心跳、进程、日志、服务、Docker 等数据时，Backend 会更新该主机的 `last_seen`。

Backend 当前默认以 `30s` 未上报作为主机离线展示口径。宕机告警不再等待固定 2 分钟，而是按告警规则的持续时间检查 `last_seen`。

### CPU 口径

Agent 上报 CPU 原始使用率和核心数。前端对进程、Docker 容器等可能超过 100% 的 CPU 数据，会结合主机核心数展示为“占主机总 CPU 容量百分比”，同时保留原始百分比和折算核心数。

### GPU 不可用告警

GPU 不可用告警依赖 Agent 上报的 GPU 设备列表：

- 如果主机有 GPU 且采集正常，`gpu.devices` 应包含至少一个设备。
- 如果最新指标中 `gpu.devices` 缺失或为空，Backend 的 `gpu_unavailable` 规则会认为 GPU 不可用。
- 该规则仍会按持续时间进入 firing，避免一次短暂采集失败立即告警。

### 服务端口告警

服务端口告警依赖服务监控采集结果。Agent 会上报服务状态和端口可访问性，Backend 的 `service_port` 规则按指定端口判断是否不可访问，并按规则持续时间触发告警。

## 🛠️ 技术栈

- **语言**: Go 1.21+
- **采集库**: gopsutil (系统指标采集)
- **通信**: gRPC
- **配置**: YAML
- **其他**: 
  - 支持Linux和Windows
  - 跨平台文件系统操作
  - 日志级别自动识别

## 🚀 快速开始

### 环境要求

- Go >= 1.21
- 操作系统: Linux / Windows / macOS

### 1. 安装依赖

```bash
go mod download
```

### 2. 生成Protobuf代码

```bash
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/collector.proto
```

### 3. 配置文件

创建 `agent-config.yaml`:

```yaml
server_addr: "localhost:50051"  # Backend gRPC地址
host_id: "server-001"            # 主机唯一标识
collect_interval: 10             # 采集间隔（秒）
manual_ip: "192.168.1.100"      # 手动指定IP（可选）
debug: false                     # 调试模式
```

### 4. 运行

```bash
# 使用配置文件
go run .

# 或使用命令行参数
go run . -server localhost:50051 -host-id server-001 -interval 10

# 调试模式
go run . -debug
```

### 5. 编译

```bash
go build -o monitor-agent
```

## 📁 项目结构

```
monitor-agent/
├── main.go                    # 入口文件
├── config_agent.go            # 配置管理
├── reporter.go                # 数据上报
├── types.go                   # 数据结构定义
│
├── collector_cpu.go           # CPU采集器
├── collector_memory.go        # 内存采集器
├── collector_disk.go          # 磁盘采集器
├── collector_network.go       # 网络采集器
├── collector_gpu.go           # GPU采集器
├── collector_log.go           # 日志采集器
├── collector_process.go       # 进程采集器
├── collector_service.go       # 服务采集器
├── collector_script.go        # 脚本执行器
│
├── proto/                     # Protobuf定义
│   ├── collector.proto        # 数据采集协议
│   ├── collector.pb.go        # 生成的代码
│   └── collector_grpc.pb.go   # 生成的gRPC代码
│
├── agent-config.yaml          # 配置文件
├── agent-config.example.yaml  # 配置示例
├── CONFIG_GUIDE.md            # 配置指南
├── go.mod                     # Go模块定义
└── go.sum                     # 依赖校验和
```

## ⚙️ 配置说明

### 配置文件: agent-config.yaml

详细配置说明请参考 [CONFIG_GUIDE.md](./CONFIG_GUIDE.md)

#### 基础配置

```yaml
server_addr: "localhost:50051"    # gRPC服务器地址
host_id: "server-001"             # 主机唯一标识
collect_interval: 10              # 采集间隔（秒）
manual_ip: "192.168.1.100"       # 手动指定IP（可选）
debug: false                      # 调试模式
```

#### 日志收集配置

```yaml
log_paths:
  - "/var/log/syslog"
  - "/var/log/nginx/access.log"
  - "/opt/myapp/logs/app.log"
```

#### 脚本执行配置

```yaml
scripts:
  - id: "script-1"
    name: "磁盘检查"
    command: "df"
    args: ["-h"]
    timeout: 30
    interval: 300  # 执行间隔（秒）
```

#### 服务监控配置

```yaml
services:
  - "sshd"
  - "nginx"
  - "mysql"
```

#### GPU采集说明

GPU 采集器会尽量将不同厂商命令输出归一化为统一结构。Backend 接收的字段包括：

- `index`
- `name`
- `vendor`
- `model`
- `uuid`
- `driver_version`
- `utilization_percent`
- `memory_total`
- `memory_used`
- `memory_used_percent`
- `temperature`
- `power_watts`
- `fan_speed_percent`

新增 GPU 厂商适配时，应优先在 Agent 侧新增 parser，并保持输出字段兼容该结构。

### 命令行参数

```bash
-server string      # gRPC服务器地址
-host-id string     # 主机唯一标识
-interval int       # 采集间隔（秒）
-ip string          # 手动指定IP
-debug              # 调试模式
-config string      # 配置文件路径（默认: agent-config.yaml）
```

## 📖 使用指南

### 基本使用

1. **配置Agent**: 编辑 `agent-config.yaml`，设置Backend地址和主机ID
2. **启动Agent**: 运行 `./monitor-agent` 或 `go run .`
3. **验证连接**: 检查Agent日志，确认已连接到Backend
4. **查看数据**: 在Backend前端界面查看监控数据

### 日志收集

配置日志文件路径：

```yaml
log_paths:
  - "/var/log/syslog"
  - "/var/log/nginx/access.log"
```

Agent会自动：
- 读取日志文件的最后N行（默认100行）
- 自动识别日志级别（ERROR, WARN, INFO, DEBUG）
- 定期采集并上报

### 脚本执行

配置脚本：

```yaml
scripts:
  - id: "disk-check"
    name: "磁盘空间检查"
    command: "df"
    args: ["-h"]
    timeout: 30
    interval: 300
```

支持的脚本类型：
- Shell脚本
- Python脚本
- 系统命令
- Windows PowerShell脚本

### 服务监控

配置要监控的服务：

```yaml
services:
  - "sshd"
  - "nginx"
  - "mysql"
```

Agent会监控：
- 服务运行状态
- 是否开机自启
- 服务描述和运行时长
- 端口可访问性（用于 Backend 服务端口告警）

### Docker 和进程监控

Agent 会采集进程和 Docker 容器资源使用情况，并上报历史趋势所需字段。多核主机上进程或容器 CPU 原始值可能超过 100%，这是因为原始值按单核百分比累计。前端会结合主机核心数展示容量占比，便于判断实际负载。

### 常见日志路径

**Linux系统日志**:
- `/var/log/syslog` - 系统日志
- `/var/log/messages` - 系统消息
- `/var/log/kern.log` - 内核日志

**Web服务器日志**:
- `/var/log/nginx/access.log` - Nginx访问日志
- `/var/log/nginx/error.log` - Nginx错误日志
- `/var/log/apache2/access.log` - Apache访问日志

**应用日志**:
- `/opt/myapp/logs/app.log` - 应用日志

## 💻 开发指南

### 安装依赖

```bash
go mod download
```

### 生成Protobuf代码

```bash
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/collector.proto
```

### 添加新的采集器

1. 创建新的采集器文件，如 `collector_custom.go`
2. 实现采集逻辑
3. 在 `main.go` 中注册采集器
4. 在 `reporter.go` 中添加数据上报逻辑

### 添加新的 GPU 厂商适配

1. 在 GPU 采集器中新增厂商探测逻辑。
2. 调用厂商命令或读取驱动接口。
3. 将输出解析为统一的 GPU 设备结构。
4. 确保多卡场景稳定生成多条设备记录。
5. 在无 GPU 或命令不可用时返回空设备列表，不要伪造设备数据。

### 代码结构说明

- **main.go**: 程序入口，初始化配置和采集器
- **config_agent.go**: 配置加载和管理
- **reporter.go**: 数据上报到Backend
- **collector_*.go**: 各种采集器实现
- **types.go**: 数据结构定义

## 🚢 部署

### Docker 构建与运行

项目提供 `Dockerfile`，可用于构建 Agent 镜像并在容器中运行。

**构建镜像：**

```bash
cd monitor-agent
docker build -t monitor-agent:latest .
```

**运行容器：**

需挂载配置文件（或通过环境变量指定配置路径），并确保能访问 Backend 的 gRPC 端口（默认 50051）。

```bash
# 使用宿主机上的 agent-config.yaml
docker run -d --name monitor-agent \
  -v /path/on/host/agent-config.yaml:/app/agent-config.yaml \
  -e CONFIG_PATH=/app/agent-config.yaml \
  monitor-agent:latest
```

若需采集宿主机指标，建议挂载必要路径并赋予相应权限（按需使用 `--pid=host` 或挂载 `/proc`、`/sys` 等），例如：

```bash
docker run -d --name monitor-agent \
  -v /path/on/host/agent-config.yaml:/app/agent-config.yaml \
  -v /proc:/host/proc:ro \
  -v /sys:/host/sys:ro \
  -e CONFIG_PATH=/app/agent-config.yaml \
  monitor-agent:latest
```

**环境变量：**

- `CONFIG_PATH`：配置文件路径，默认 `/app/agent-config.yaml`。挂载配置时请与此路径一致。

### 编译

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o monitor-agent

# Windows
GOOS=windows GOARCH=amd64 go build -o monitor-agent.exe

# macOS
GOOS=darwin GOARCH=amd64 go build -o monitor-agent
```

### 部署到生产环境

1. **上传文件**:
```bash
scp monitor-agent user@server:/opt/monitor-agent/
scp agent-config.yaml user@server:/opt/monitor-agent/
```

2. **设置权限**:
```bash
chmod +x /opt/monitor-agent/monitor-agent
```

3. **配置systemd服务** (Linux):

创建 `/etc/systemd/system/monitor-agent.service`:

```ini
[Unit]
Description=Monitor Agent Service
After=network.target

[Service]
Type=simple
User=monitor
WorkingDirectory=/opt/monitor-agent
ExecStart=/opt/monitor-agent/monitor-agent
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl enable monitor-agent
sudo systemctl start monitor-agent
sudo systemctl status monitor-agent
```

### Windows服务部署

使用NSSM将Agent安装为Windows服务：

```bash
nssm install MonitorAgent "C:\monitor-agent\monitor-agent.exe"
nssm start MonitorAgent
```

## 🔧 故障排查

### Agent无法连接到Backend

- 检查Backend的gRPC服务是否启动
- 检查配置文件中的`server_addr`是否正确
- 检查防火墙设置
- 检查网络连通性: `telnet backend-ip 50051`

### 日志收集失败

- 检查日志文件路径是否正确
- 检查Agent是否有读取权限
- 检查日志文件是否存在

### 脚本执行失败

- 检查脚本路径是否正确
- 检查脚本是否有执行权限
- 检查脚本执行环境（如Python版本）

### 服务监控失败

- 检查服务名称是否正确
- 检查是否有权限查询服务状态
- Linux: 使用 `systemctl list-units` 查看服务名
- Windows: 使用 `sc query` 查看服务名

## 📝 依赖说明

主要依赖：

```go
require (
    github.com/shirou/gopsutil/v3 v3.23.12
    google.golang.org/grpc v1.60.0
    google.golang.org/protobuf v1.31.0
    gopkg.in/yaml.v3 v3.0.1
)
```

## 📄 许可证

MIT license

## 📞 联系方式

WX:Li1024_REBOOT

## 🔗 相关文档

- [配置指南](./CONFIG_GUIDE.md) - 详细的配置说明和示例
- [Backend README](../monitor-backend/README.md) - Backend服务文档
