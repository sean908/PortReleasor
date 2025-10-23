//go:build linux

package platform

import (
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

// LinuxManager Linux平台实现
type LinuxManager struct{}

// GetPortConnections 获取Linux系统端口连接信息
func (lm *LinuxManager) GetPortConnections() ([]types.PortInfo, error) {
	cmd := exec.Command("ss", "-tunlp")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return lm.getPortConnectionsWithNetstat()
	}

	var connections []types.PortInfo
	lines := strings.Split(out.String(), "\n")

	re := regexp.MustCompile(`\s+`)

	for i, line := range lines {
		if i == 0 {
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := re.Split(line, -1)
		if len(fields) < 5 {
			continue
		}

		protocol := strings.ToUpper(fields[0])
		localAddr := fields[4]

		parts := strings.Split(localAddr, ":")
		if len(parts) < 2 {
			continue
		}

		portStr := parts[len(parts)-1]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}

		var pid int
		var processName string

		if len(fields) >= 6 {
			processInfo := fields[5]
			pidMatch := regexp.MustCompile(`pid=(\d+)`).FindStringSubmatch(processInfo)
			if len(pidMatch) > 1 {
				pid, _ = strconv.Atoi(pidMatch[1])
				processName = lm.getProcessName(pid)
			}
		}

		connections = append(connections, types.PortInfo{
			Port:        port,
			Protocol:    protocol,
			PID:         pid,
			ProcessName: processName,
			LocalAddr:   localAddr,
			State:       "LISTENING",
		})
	}

	return connections, nil
}

// getPortConnectionsWithNetstat 使用netstat作为备用方案
func (lm *LinuxManager) getPortConnectionsWithNetstat() ([]types.PortInfo, error) {
	cmd := exec.Command("netstat", "-tunlp")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute netstat: %v", err)
	}

	var connections []types.PortInfo
	lines := strings.Split(out.String(), "\n")

	re := regexp.MustCompile(`\s+`)

	for i, line := range lines {
		if i < 2 {
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := re.Split(line, -1)
		if len(fields) < 6 {
			continue
		}

		protocol := strings.ToUpper(fields[0])
		localAddr := fields[3]

		parts := strings.Split(localAddr, ":")
		if len(parts) < 2 {
			continue
		}

		portStr := parts[len(parts)-1]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}

		var pid int
		var processName string

		if len(fields) >= 7 {
			processInfo := fields[6]
			pidMatch := regexp.MustCompile(`(\d+)/(.+)`).FindStringSubmatch(processInfo)
			if len(pidMatch) > 2 {
				pid, _ = strconv.Atoi(pidMatch[1])
				processName = pidMatch[2]
			}
		}

		connections = append(connections, types.PortInfo{
			Port:        port,
			Protocol:    protocol,
			PID:         pid,
			ProcessName: processName,
			LocalAddr:   localAddr,
			State:       "LISTENING",
		})
	}

	return connections, nil
}

// getProcessName 获取进程名称
func (lm *LinuxManager) getProcessName(pid int) string {
	cmdPath := fmt.Sprintf("/proc/%d/comm", pid)
	data, err := os.ReadFile(cmdPath)
	if err != nil {
		return "Unknown"
	}

	return strings.TrimSpace(string(data))
}

// KillProcessByPID 在Linux上杀死指定PID的进程
func (lm *LinuxManager) KillProcessByPID(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %v", pid, err)
	}

	err = proc.Signal(syscall.SIGKILL)
	if err != nil {
		return fmt.Errorf("failed to kill process %d: %v", pid, err)
	}

	return nil
}

// GetProcessPath 获取Linux进程路径
func (lm *LinuxManager) GetProcessPath(pid int) (string, error) {
	exePath := fmt.Sprintf("/proc/%d/exe", pid)
	path, err := os.Readlink(exePath)
	if err != nil {
		return "", fmt.Errorf("failed to get process path: %v", err)
	}

	return path, nil
}

func init() {
	GetPlatformManager = func() PlatformManager {
		return &LinuxManager{}
	}
}
