# PortReleasor

一个跨平台的端口管理工具，用于检查端口占用情况并释放指定端口。

## 功能特性

- 🔍 **端口检查** - 列出端口占用情况、进程ID和程序名称
- 🔄 **端口释放** - 安全释放指定端口（支持确认机制）
- 🌍 **跨平台支持** - 支持 Windows、Linux 和 macOS
- ⚡ **性能优化** - 进程缓存和去重机制，响应快速
- 🎯 **灵活匹配** - 支持单端口、多端口、端口范围和通配符模式

## 系统要求

- **Go 1.21+** - 运行环境要求
- **管理员权限** - 释放端口时可能需要

## 安装与使用

### 方法一：源码直接运行

```bash
# 克隆仓库
git clone https://github.com/sean908/PortReleasor.git
cd PortReleasor

# 直接运行源码
go run . -h
go run . check -w 8080
go run . release 8080
```

### 方法二：编译运行

```bash
# 编译
go build -o portreleasor

# 运行
./portreleasor -h  # Linux/macOS
portreleasor.exe -h  # Windows
```

## 功能详解

### 端口检查 (`check`)

检查当前端口占用情况：

```bash
# 检查所有端口
go run . check

# 检查指定端口
go run . check 8080 8081 8082

# 通配符匹配（包含指定数字的端口）
go run . check -w 80  # 匹配 80, 8080, 18080 等

# 详细模式（显示程序路径）
go run . check -v
```

**输出示例：**
```
PORT/PROTOCOL    PID     PROCESS
----------------------------------------
8080/TCP         12345    node.exe
8081/TCP         12346    python.exe

Showing 2 unique port(s)
```

### 端口释放 (`release`)

释放指定端口，终止占用进程：

```bash
# 释放单个端口（需要确认）
go run . release 8080

# 强制释放（无需确认）
go run . release 8080 -f

# 释放多个端口
go run . release 8080 8081 8082

# 释放端口范围
go run . release 8080-8090

# 通配符释放
go run . release -w 80
```

**安全机制：**
- 默认需要用户确认（y/N）
- `-f` 参数可跳过确认直接执行
- 显示将要终止的进程信息

### 帮助信息

```bash
go run . -h
go run . check -h
go run . release -h
```

## 跨平台支持

| 平台 | 端口检测工具 | 进程管理 |
|------|------------|----------|
| Windows | `netstat -ano` + `tasklist` | `os.FindProcess().Kill()` |
| Linux | `ss -tunlp` + `ps` | `syscall.SIGKILL` |
| macOS | `lsof -i -P -n` + `ps` | `SIGTERM -> SIGKILL` |

## ⚠️ 注意事项

 **安全警告：**
- 释放端口会强制终止占用进程，请确认操作
- 系统关键进程可能无法终止（权限保护）
- 建议先使用 `check` 命令查看端口占用情况

## 作者

Se@n

## 许可证

本项目采用 MIT 许可证。