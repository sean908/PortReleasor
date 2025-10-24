//go:build linux

package platform

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"portreleasor/internal/types"
)

// LinuxManager Linux平台实现
type LinuxManager struct {
	processNameCache map[int]string
	processPathCache map[int]string
	isRoot          bool
}

// initProcessCache 初始化进程缓存
func (lm *LinuxManager) initProcessCache() {
	if lm.processNameCache == nil {
		lm.processNameCache = make(map[int]string)
	}
	if lm.processPathCache == nil {
		lm.processPathCache = make(map[int]string)
	}
	// 检测是否为root用户
	if currentUser, err := user.Current(); err == nil {
		lm.isRoot = currentUser.Uid == "0"
	}
}

// getAllProcessInfo 批量获取所有进程信息（名称和路径）
func (lm *LinuxManager) getAllProcessInfo() error {
	lm.initProcessCache()

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

			// 处理WSL中可能被截断的进程名
			if len(name) > 15 && strings.Contains(name, "(") {
				// 可能是截断的进程名，尝试获取完整名称
				if pid, err := strconv.Atoi(pidStr); err == nil {
					fullName := lm.getProcessName(pid)
					if fullName != "" && fullName != "Unknown" {
						lm.processNameCache[pid] = fullName
					} else {
						lm.processNameCache[pid] = name
					}
					// 尝试获取进程路径
					lm.cacheProcessPath(pid)
				}
			} else {
				if pid, err := strconv.Atoi(pidStr); err == nil {
					lm.processNameCache[pid] = name
					// 尝试获取进程路径
					lm.cacheProcessPath(pid)
				}
			}
		}
	}

	return nil
}

// GetPortConnections 获取Linux系统端口连接信息
func (lm *LinuxManager) GetPortConnections() ([]types.PortInfo, error) {
	// 首先批量获取所有进程信息
	if err := lm.getAllProcessInfo(); err != nil {
		return nil, fmt.Errorf("failed to get process info: %v", err)
	}

	cmd := exec.Command("ss", "-tunlp")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return lm.getPortConnectionsWithNetstat()
	}

	var connections []types.PortInfo
	lines := strings.Split(out.String(), "\n")

	re := regexp.MustCompile(`\s+`)
	portMap := make(map[string]types.PortInfo) // key: "port:protocol"

	for i, line := range lines {
		if i == 0 {
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := re.Split(line, -1)

		// Handle WSL/ss output format variations
		if len(fields) < 4 {
			continue
		}

		protocol := strings.ToUpper(fields[0])
		state := fields[1]
		var localAddr string
		if len(fields) > 4 {
			localAddr = fields[4]
		} else {
			localAddr = fields[3]
		}

		// Extract port from local address
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

		// Check if we have process info (WSL/ss might not show Process column)
		if len(fields) >= 6 {
			for j := 5; j < len(fields); j++ {
				processInfo := fields[j]
				if strings.Contains(processInfo, "pid=") {
					pidMatch := regexp.MustCompile(`pid=(\d+)`).FindStringSubmatch(processInfo)
					if len(pidMatch) > 1 {
						pid, _ = strconv.Atoi(pidMatch[1])
						processName = lm.getProcessNameFromCache(pid)
						break
					}
				}
			}
		}

		// If no PID found, try to infer from state or use alternative method
		if pid == 0 && strings.Contains(state, "LISTEN") {
			// For listening ports without PID info, try alternative detection
			pid, processName = lm.findProcessForPort(port, protocol)
		}

		key := fmt.Sprintf("%d:%s", port, protocol)

		// 获取进程路径
		processPath := lm.getProcessPathWithPermissionCheck(pid)

		// 创建新的端口信息
		newInfo := types.PortInfo{
			Port:        port,
			Protocol:    protocol,
			PID:         pid,
			ProcessName: processName,
			ProcessPath: processPath,
			LocalAddr:   localAddr,
			State:       "LISTENING",
		}

		// 检查是否已存在该端口的记录
		if _, exists := portMap[key]; exists {
			// 保持现有的记录（ss 通常不会重复显示监听端口）
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

// findProcessForPort 通过其他方法查找使用指定端口的进程
func (lm *LinuxManager) findProcessForPort(port int, protocol string) (int, string) {
	// 尝试使用lsof命令
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port))
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err == nil {
		lines := strings.Split(out.String(), "\n")
		if len(lines) > 1 {
			// 跳过标题行
			for _, line := range lines[1:] {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				fields := strings.Fields(line)
				if len(fields) >= 2 {
					// lsof输出格式: COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME
					pidStr := fields[1]
					processName := fields[0]
					if pid, err := strconv.Atoi(pidStr); err == nil {
						return pid, processName
					}
				}
			}
		}
	}

	// 如果lsof不可用，尝试解析/proc/net/tcp和udp
	return lm.findProcessFromProcNet(port, protocol)
}

// findProcessFromProcNet 从/proc/net中查找进程信息
func (lm *LinuxManager) findProcessFromProcNet(port int, protocol string) (int, string) {
	var filename string
	if strings.ToUpper(protocol) == "TCP" {
		filename = "/proc/net/tcp"
	} else if strings.ToUpper(protocol) == "UDP" {
		filename = "/proc/net/udp"
	} else {
		return 0, ""
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return 0, ""
	}

	lines := strings.Split(string(data), "\n")
	portHex := fmt.Sprintf("%04X", port)

	for i, line := range lines {
		if i == 0 || line == "" {
			continue // Skip header
		}

		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		// 检查本地地址端口是否匹配
		localAddr := fields[1]
		if strings.HasSuffix(localAddr, ":"+portHex) {
			// 从inode字段查找进程
			inode := fields[9]
			return lm.findProcessByInode(inode)
		}
	}

	return 0, ""
}

// findProcessByInode 通过inode查找进程
func (lm *LinuxManager) findProcessByInode(inode string) (int, string) {
	if inode == "0" {
		return 0, ""
	}

	// 遍历/proc/[pid]/fd目录查找socket链接
	procDir, err := os.Open("/proc")
	if err != nil {
		return 0, ""
	}
	defer procDir.Close()

	entries, err := procDir.Readdirnames(-1)
	if err != nil {
		return 0, ""
	}

	for _, entry := range entries {
		pid, err := strconv.Atoi(entry)
		if err != nil {
			continue // 不是数字目录，跳过
		}

		fdDir := fmt.Sprintf("/proc/%d/fd", pid)
		fdEntries, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}

		for _, fdEntry := range fdEntries {
			linkPath := fmt.Sprintf("%s/%s", fdDir, fdEntry.Name())
			linkTarget, err := os.Readlink(linkPath)
			if err != nil {
				continue
			}

			// 检查是否是socket且inode匹配
			if strings.HasPrefix(linkTarget, "socket:[") && strings.Contains(linkTarget, inode) {
				processName := lm.getProcessNameFromCache(pid)
				if processName == "" {
					processName = lm.getProcessName(pid)
				}
				return pid, processName
			}
		}
	}

	return 0, ""
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
			if processInfo != "-" {
				pidMatch := regexp.MustCompile(`(\d+)/(.+)`).FindStringSubmatch(processInfo)
				if len(pidMatch) > 2 {
					pid, _ = strconv.Atoi(pidMatch[1])
					processName = pidMatch[2]
				}
			} else {
				// netstat显示"-"表示无法获取PID信息，尝试其他方法
				pid, processName = lm.findProcessForPort(port, protocol)
			}
		}

		connections = append(connections, types.PortInfo{
			Port:        port,
			Protocol:    protocol,
			PID:         pid,
			ProcessName: processName,
			ProcessPath: lm.getProcessPathWithPermissionCheck(pid),
			LocalAddr:   localAddr,
			State:       "LISTENING",
		})
	}

	return connections, nil
}

// cacheProcessPath 缓存进程路径
func (lm *LinuxManager) cacheProcessPath(pid int) {
	if path := lm.getProcessPathFromCache(pid); path == "" {
		// 尝试从/proc/pid/exe获取路径
		exePath := fmt.Sprintf("/proc/%d/exe", pid)
		if path, err := os.Readlink(exePath); err == nil {
			lm.processPathCache[pid] = path
		}
	}
}

// getProcessPathFromCache 从缓存获取进程路径
func (lm *LinuxManager) getProcessPathFromCache(pid int) string {
	lm.initProcessCache()
	if path, exists := lm.processPathCache[pid]; exists {
		return path
	}
	return ""
}

// getProcessPathWithPermissionCheck 根据权限检查获取进程路径显示文本
func (lm *LinuxManager) getProcessPathWithPermissionCheck(pid int) string {
	// PID为0的情况
	if pid == 0 {
		if lm.isRoot {
			return "N/A"
		} else {
			return "NO PERMISSION"
		}
	}

	// 尝试从缓存获取路径
	if path := lm.getProcessPathFromCache(pid); path != "" {
		return path
	}

	// 尝试实时获取路径
	exePath := fmt.Sprintf("/proc/%d/exe", pid)
	if path, err := os.Readlink(exePath); err == nil {
		return path
	}

	// 无法获取路径，检查权限
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); os.IsNotExist(err) {
		return "PROCESS NOT FOUND"
	}

	return "NO PERMISSION"
}

// getProcessNameFromCache 从缓存获取进程名称
func (lm *LinuxManager) getProcessNameFromCache(pid int) string {
	lm.initProcessCache()
	if name, exists := lm.processNameCache[pid]; exists {
		return name
	}
	// 备用方案：从 /proc 读取
	return lm.getProcessName(pid)
}

// getProcessName 获取进程名称（备用方案）
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
