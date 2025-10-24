//go:build darwin

package platform

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"portreleasor/internal/types"
)

// DarwinManager macOS平台实现
type DarwinManager struct {
	processNameCache map[int]string
}

// initProcessCache 初始化进程缓存
func (dm *DarwinManager) initProcessCache() {
	if dm.processNameCache == nil {
		dm.processNameCache = make(map[int]string)
	}
}

// getAllProcessNames 批量获取所有进程名称
func (dm *DarwinManager) getAllProcessNames() error {
	dm.initProcessCache()

	cmd := exec.Command("ps", "-axo", "pid,comm")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute ps: %v", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(out.String()))
	// Skip header line
	if scanner.Scan() {
		// Skip header
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pidStr := fields[0]
			name := fields[1]
			if pid, err := strconv.Atoi(pidStr); err == nil {
				dm.processNameCache[pid] = name
			}
		}
	}

	return nil
}

// GetPortConnections 获取macOS系统端口连接信息
func (dm *DarwinManager) GetPortConnections() ([]types.PortInfo, error) {
	// 首先批量获取所有进程信息
	if err := dm.getAllProcessNames(); err != nil {
		return nil, fmt.Errorf("failed to get process names: %v", err)
	}

	cmd := exec.Command("lsof", "-i", "-P", "-n")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute lsof: %v", err)
	}

	var connections []types.PortInfo
	lines := strings.Split(out.String(), "\n")

	portMap := make(map[string]types.PortInfo) // key: "port:protocol"

	re := regexp.MustCompile(`\s+`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "COMMAND") {
			continue
		}

		fields := re.Split(line, -1)
		if len(fields) < 9 {
			continue
		}

		processName := fields[0]
		pidStr := fields[1]
		protocol := strings.ToUpper(fields[7])
		address := fields[8]

		// 提取端口号
		var port int
		if strings.Contains(address, "->") {
			// 跳过 outgoing connections
			continue
		}

		parts := strings.Split(address, ":")
		if len(parts) < 2 {
			continue
		}

		portStr := parts[len(parts)-1]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		// 从缓存获取进程名称
		cachedName := dm.getProcessNameFromCache(pid)
		if cachedName != "" {
			processName = cachedName
		}

		// 提取进程名称和完整路径
		displayName, fullPath := dm.extractProcessNameAndPath(processName)

		key := fmt.Sprintf("%d:%s", port, protocol)

		// 创建新的端口信息
		newInfo := types.PortInfo{
			Port:        port,
			Protocol:    protocol,
			PID:         pid,
			ProcessName: displayName,
			ProcessPath: fullPath,
			LocalAddr:   address,
			State:       "LISTENING",
		}

		// 检查是否已存在该端口的记录
		if _, exists := portMap[key]; exists {
			// 保持现有的记录（macOS lsof 通常不会重复显示监听端口）
			continue
		} else {
			// 如果不存在，直接添加
			portMap[key] = newInfo
		}
	}

	// 将 map 转换为 slice
	for _, info := range portMap {
		connections = append(connections, info)
	}

	return connections, nil
}

// extractProcessNameAndPath 从完整路径中提取进程名称和路径
func (dm *DarwinManager) extractProcessNameAndPath(fullPath string) (string, string) {
	if fullPath == "" {
		return "", ""
	}

	// 如果路径中包含斜杠，提取最后一个部分作为进程名
	if strings.Contains(fullPath, "/") {
		parts := strings.Split(fullPath, "/")
		name := parts[len(parts)-1]
		return name, fullPath
	}

	// 如果没有斜杠，说明本身就不是路径，直接返回
	return fullPath, ""
}

// getProcessNameFromCache 从缓存获取进程名称
func (dm *DarwinManager) getProcessNameFromCache(pid int) string {
	dm.initProcessCache()
	if name, exists := dm.processNameCache[pid]; exists {
		return name
	}
	return ""
}

// KillProcessByPID 在macOS上杀死指定PID的进程
func (dm *DarwinManager) KillProcessByPID(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %v", pid, err)
	}

	err = proc.Signal(syscall.SIGTERM)
	if err != nil {
		// 如果 SIGTERM 失败，尝试 SIGKILL
		err = proc.Signal(syscall.SIGKILL)
		if err != nil {
			return fmt.Errorf("failed to kill process %d: %v", pid, err)
		}
	}

	return nil
}

// GetProcessPath 获取macOS进程路径
func (dm *DarwinManager) GetProcessPath(pid int) (string, error) {
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get process path: %v", err)
	}

	lines := strings.Split(out.String(), "\n")
	if len(lines) >= 2 {
		path := strings.TrimSpace(lines[1])
		if path != "" {
			return path, nil
		}
	}

	return "", fmt.Errorf("process path not found")
}

func init() {
	// 注册macOS管理器
	GetPlatformManager = func() PlatformManager {
		return &DarwinManager{}
	}
}