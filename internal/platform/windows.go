//go:build windows

package platform

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"portreleasor/internal/types"
)

// WindowsManager Windows平台实现
type WindowsManager struct {
	processNameCache map[int]string
}

// Windows API 相关常量和结构体
const (
	TH32CS_SNAPPROCESS = 0x00000002
	PROCESS_QUERY_INFORMATION = 0x0400
	PROCESS_VM_READ = 0x0010
)

type PROCESSENTRY32 struct {
	dwSize              uint32
	cntUsage            uint32
	th32ProcessID       uint32
	th32DefaultHeapID   uintptr
	th32ModuleID        uint32
	cntThreads          uint32
	th32ParentProcessID uint32
	pcPriClassBase      int32
	dwFlags             uint32
	szExeFile           [260]uint16
}

// initProcessCache 初始化进程缓存
func (wm *WindowsManager) initProcessCache() {
	if wm.processNameCache == nil {
		wm.processNameCache = make(map[int]string)
	}
}

// getAllProcessNames 批量获取所有进程名称
func (wm *WindowsManager) getAllProcessNames() error {
	wm.initProcessCache()

	cmd := exec.Command("tasklist", "/FO", "CSV", "/NH")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute tasklist: %v", err)
	}

	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) >= 2 {
			name := strings.Trim(fields[0], "\"")
			pidStr := strings.Trim(fields[1], "\"")
			if pid, err := strconv.Atoi(pidStr); err == nil {
				wm.processNameCache[pid] = name
			}
		}
	}

	return nil
}
// GetPortConnections 获取Windows系统端口连接信息
func (wm *WindowsManager) GetPortConnections() ([]types.PortInfo, error) {
	// 首先批量获取所有进程信息
	if err := wm.getAllProcessNames(); err != nil {
		return nil, fmt.Errorf("failed to get process names: %v", err)
	}

	cmd := exec.Command("netstat", "-ano")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute netstat: %v", err)
	}

	var connections []types.PortInfo
	lines := strings.Split(out.String(), "\n")

	re := regexp.MustCompile(`\s+`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "TCP") && !strings.HasPrefix(line, "UDP") {
			continue
		}

		fields := re.Split(line, -1)
		if len(fields) < 4 {
			continue
		}

		protocol := fields[0]
		localAddr := fields[1]
		var pid int
		var state string

		if protocol == "TCP" {
			if len(fields) >= 5 {
				state = fields[3]
				pidStr := fields[4]
				var err error
				pid, err = strconv.Atoi(pidStr)
				if err != nil {
					continue
				}
			}
		} else if protocol == "UDP" {
			if len(fields) >= 4 {
				pidStr := fields[3]
				var err error
				pid, err = strconv.Atoi(pidStr)
				if err != nil {
					continue
				}
				state = "LISTENING"
			}
		}

		parts := strings.Split(localAddr, ":")
		if len(parts) < 2 {
			continue
		}

		portStr := parts[len(parts)-1]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}

		processName := wm.getProcessNameFromCache(pid)

		connections = append(connections, types.PortInfo{
			Port:        port,
			Protocol:    protocol,
			PID:         pid,
			ProcessName: processName,
			LocalAddr:   localAddr,
			State:       state,
		})
	}

	return connections, nil
}

// getProcessNameFromCache 从缓存获取进程名称
func (wm *WindowsManager) getProcessNameFromCache(pid int) string {
	wm.initProcessCache()
	if name, exists := wm.processNameCache[pid]; exists {
		return name
	}
	return "Unknown"
}
func (wm *WindowsManager) getProcessName(pid int) string {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "Unknown"
	}

	output := strings.TrimSpace(out.String())
	if output == "" {
		return "Unknown"
	}

	fields := strings.Split(output, ",")
	if len(fields) > 0 {
		name := strings.Trim(fields[0], "\"")
		return name
	}

	return "Unknown"
}

// KillProcessByPID 在Windows上杀死指定PID的进程
func (wm *WindowsManager) KillProcessByPID(pid int) error {
	// 使用syscall.kill
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("无法找到进程 %d: %v", pid, err)
	}

	err = proc.Kill()
	if err != nil {
		return fmt.Errorf("无法杀死进程 %d: %v", pid, err)
	}

	return nil
}

// GetProcessPath 获取Windows进程路径
func (wm *WindowsManager) GetProcessPath(pid int) (string, error) {
	cmd := exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", pid), "get", "ExecutablePath", "/format:list")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get process path: %v", err)
	}

	output := strings.TrimSpace(out.String())
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ExecutablePath=") {
			path := strings.TrimPrefix(line, "ExecutablePath=")
			path = strings.TrimSpace(path)
			if path != "" {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("process path not found")
}

func init() {
	// 注册Windows管理器
	GetPlatformManager = func() PlatformManager {
		return &WindowsManager{}
	}
}