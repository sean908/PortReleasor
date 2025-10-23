# PortReleasor

A cross-platform port management utility for checking port usage and releasing specified ports.

## Features

- 🔍 **Port Check** - List port usage with process IDs and program names
- 🔄 **Port Release** - Safely release specified ports (with confirmation)
- 🌍 **Cross-Platform** - Support for Windows, Linux, and macOS
- ⚡ **Performance Optimized** - Process caching and deduplication for fast response
- 🎯 **Flexible Matching** - Support for single ports, multiple ports, port ranges, and wildcard patterns

## System Requirements

- **Go 1.21+** - Runtime environment requirement
- **Administrator privileges** - May be required for port release operations

## Installation and Usage

### Method 1: Run from Source

```bash
# Clone repository
git clone https://github.com/sean908/PortReleasor.git
cd PortReleasor

# Run directly from source
go run . -h
go run . check -w 8080
go run . release 8080
```

### Method 2: Compile and Run

```bash
# Compile
go build -o portreleasor

# Run
./portreleasor -h  # Linux/macOS
portreleasor.exe -h  # Windows
```

## Feature Details

### Port Check (`check`)

Check current port usage:

```bash
# Check all ports
go run . check

# Check specific ports
go run . check 8080 8081 8082

# Wildcard matching (ports containing the specified number)
go run . check -w 80  # Matches 80, 8080, 18080, etc.

# Verbose mode (show program paths)
go run . check -v
```

**Output Example:**
```
PORT/PROTOCOL    PID     PROCESS
----------------------------------------
8080/TCP         12345    node.exe
8081/TCP         12346    python.exe

Showing 2 unique port(s)
```

### Port Release (`release`)

Release specified ports by terminating occupying processes:

```bash
# Release single port (requires confirmation)
go run . release 8080

# Force release (no confirmation)
go run . release 8080 -f

# Release multiple ports
go run . release 8080 8081 8082

# Release port range
go run . release 8080-8090

# Wildcard release
go run . release -w 80
```

**Safety Mechanisms:**
- User confirmation required by default (y/N)
- `-f` flag bypasses confirmation
- Shows process information before termination

### Help Information

```bash
go run . -h
go run . check -h
go run . release -h
```

## Cross-Platform Support

| Platform | Port Detection Tools | Process Management |
|----------|---------------------|-------------------|
| Windows | `netstat -ano` + `tasklist` | `os.FindProcess().Kill()` |
| Linux | `ss -tunlp` + `ps` | `syscall.SIGKILL` |
| macOS | `lsof -i -P -n` + `ps` | `SIGTERM -> SIGKILL` |

## ⚠️ Important Notes

- Releasing ports forcefully terminates occupying processes - please confirm operations
- System critical processes may be protected and cannot be terminated
- Use `check` command first to review port usage before releasing

## Author

Se@n

## License

This project is licensed under the MIT License.