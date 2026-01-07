# Agent配置指南

## 配置文件位置

Agent配置文件：`agent-config.yaml`

## 配置项说明

### 基础配置

```yaml
server_addr: "localhost:50051"    # gRPC服务器地址
host_id: "server-001"             # 主机唯一标识
collect_interval: 10              # 采集间隔（秒）
manual_ip: "192.168.21.14"       # 手动指定IP（可选）
debug: false                      # 调试模式
```

### 日志收集配置

#### 配置格式

```yaml
log_paths:
  - "/var/log/syslog"
  - "/var/log/nginx/access.log"
  - "/opt/myapp/logs/app.log"
```

#### 功能说明

- Agent会定期读取指定日志文件的最后N行（默认100行）
- 自动识别日志级别（ERROR, WARN, INFO, DEBUG）
- 支持多个日志文件
- 日志文件不存在时会自动跳过

#### 常见日志路径

**Linux系统日志：**
- `/var/log/syslog` - 系统日志
- `/var/log/messages` - 系统消息
- `/var/log/kern.log` - 内核日志
- `/var/log/auth.log` - 认证日志

**Web服务器日志：**
- `/var/log/nginx/access.log` - Nginx访问日志
- `/var/log/nginx/error.log` - Nginx错误日志
- `/var/log/apache2/access.log` - Apache访问日志
- `/var/log/apache2/error.log` - Apache错误日志

**数据库日志：**
- `/var/log/mysql/error.log` - MySQL错误日志
- `/var/log/postgresql/postgresql.log` - PostgreSQL日志

**应用日志：**
- `/opt/myapp/logs/app.log` - 应用日志
- `/opt/myapp/logs/error.log` - 错误日志

**Windows日志（Windows系统）：**
- `C:\Windows\System32\LogFiles\W3SVC1\*.log` - IIS日志
- `C:\Program Files\MyApp\logs\*.log` - 应用日志

### 脚本执行配置

#### 配置格式

```yaml
scripts:
  - id: "script-id"           # 脚本唯一ID
    name: "脚本名称"          # 脚本显示名称
    command: "命令或脚本路径"  # 要执行的命令
    args:                     # 命令参数（可选）
      - "arg1"
      - "arg2"
    timeout: 30               # 超时时间（秒）
    interval: 300             # 执行间隔（秒）
```

#### 功能说明

- 支持执行Shell脚本、Python脚本、系统命令等
- 记录执行结果（输出、错误、退出码、耗时）
- 支持设置执行间隔，避免频繁执行
- 超时自动终止脚本执行

#### 配置示例

**示例1：执行系统命令**
```yaml
scripts:
  - id: "disk-check"
    name: "磁盘空间检查"
    command: "df"
    args:
      - "-h"
    timeout: 10
    interval: 300  # 5分钟执行一次
```

**示例2：执行Shell脚本**
```yaml
scripts:
  - id: "backup-check"
    name: "备份检查"
    command: "/opt/scripts/check_backup.sh"
    args: []
    timeout: 60
    interval: 3600  # 1小时执行一次
```

**示例3：执行Python脚本**
```yaml
scripts:
  - id: "health-check"
    name: "健康检查"
    command: "python3"
    args:
      - "/opt/scripts/health_check.py"
      - "--verbose"
    timeout: 120
    interval: 120  # 2分钟执行一次
```

**示例4：执行带参数的命令**
```yaml
scripts:
  - id: "network-test"
    name: "网络测试"
    command: "ping"
    args:
      - "-c"
      - "5"
      - "8.8.8.8"
    timeout: 15
    interval: 300
```

**示例5：Windows PowerShell脚本**
```yaml
scripts:
  - id: "windows-check"
    name: "Windows检查"
    command: "powershell"
    args:
      - "-File"
      - "C:\\Scripts\\check.ps1"
    timeout: 30
    interval: 300
```

#### 脚本执行结果

脚本执行结果会包含：
- 执行时间
- 是否成功
- 标准输出
- 错误输出
- 退出码
- 执行耗时

### 服务状态监控配置

#### 配置格式

```yaml
services:
  - "sshd"
  - "nginx"
  - "mysql"
```

#### 功能说明

- 监控系统服务的运行状态
- 支持Linux systemd服务和Windows服务
- 显示服务状态（running, stopped, failed）
- 显示是否开机自启
- 显示服务描述和运行时长

#### Linux服务示例

```yaml
services:
  - "sshd"          # SSH服务
  - "docker"        # Docker服务
  - "nginx"         # Nginx
  - "mysql"         # MySQL
  - "postgresql"    # PostgreSQL
  - "redis"         # Redis
  - "mongodb"       # MongoDB
```

#### Windows服务示例

```yaml
services:
  - "Spooler"       # 打印服务
  - "Themes"        # 主题服务
  - "WSearch"       # Windows搜索
  - "MySQL80"       # MySQL服务
  - "MSSQLSERVER"   # SQL Server
```

## 完整配置示例

### Linux系统完整配置

```yaml
server_addr: "192.168.1.100:50051"
host_id: "web-server-01"
collect_interval: 10
manual_ip: "192.168.1.10"
debug: false

log_paths:
  - "/var/log/syslog"
  - "/var/log/nginx/access.log"
  - "/var/log/nginx/error.log"
  - "/var/log/mysql/error.log"
  - "/opt/myapp/logs/app.log"

scripts:
  - id: "disk-check"
    name: "磁盘空间检查"
    command: "df"
    args: ["-h"]
    timeout: 10
    interval: 300

  - id: "backup-status"
    name: "备份状态检查"
    command: "/opt/scripts/check_backup.sh"
    args: []
    timeout: 60
    interval: 3600

services:
  - "sshd"
  - "nginx"
  - "mysql"
  - "docker"
```

### Windows系统完整配置

```yaml
server_addr: "192.168.1.100:50051"
host_id: "windows-server-01"
collect_interval: 10
manual_ip: "192.168.1.20"
debug: false

log_paths:
  - "C:\\Windows\\System32\\LogFiles\\W3SVC1\\*.log"
  - "C:\\Program Files\\MyApp\\logs\\*.log"

scripts:
  - id: "disk-check"
    name: "磁盘空间检查"
    command: "powershell"
    args:
      - "-Command"
      - "Get-PSDrive C | Select-Object Used,Free"
    timeout: 10
    interval: 300

services:
  - "Spooler"
  - "Themes"
  - "MySQL80"
```

## 配置注意事项

1. **日志文件权限**：确保Agent有权限读取日志文件
2. **脚本执行权限**：确保脚本有执行权限
3. **路径格式**：
   - Linux使用 `/` 作为路径分隔符
   - Windows使用 `\\` 或 `/` 作为路径分隔符
4. **超时设置**：根据脚本执行时间合理设置超时值
5. **执行间隔**：避免设置过小的间隔，防止系统负载过高
6. **服务名称**：使用systemctl或sc命令查询准确的服务名称

## 验证配置

配置完成后，重启Agent，查看日志确认：
- 日志收集器是否成功加载日志路径
- 脚本执行器是否成功加载脚本配置
- 服务监控器是否成功加载服务列表

## 故障排查

1. **日志收集失败**：检查日志文件路径和权限
2. **脚本执行失败**：检查脚本路径、权限和执行环境
3. **服务监控失败**：检查服务名称是否正确，是否有权限查询服务状态

