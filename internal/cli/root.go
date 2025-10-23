package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"portreleasor/internal/core"
)

var (
	releasePorts    []string
	forceRelease    bool
	checkPorts      []string
	verboseCheck    bool
	wildcardCheck   bool
)

var rootCmd = &cobra.Command{
	Use:   "portreleasor",
	Short: "跨平台端口释放工具",
	Long: `PortReleasor 是一个跨平台的端口管理工具，
可以检查端口占用情况并释放指定端口`,
}

var releaseCmd = &cobra.Command{
	Use:   "release [ports...]",
	Short: "释放指定端口",
	Long: `释放被占用的端口，支持：
- 单个端口: 8080
- 多个端口: 8080 8081 8082
- 端口范围: 8080-8090`,
	Run: runRelease,
}

var checkCmd = &cobra.Command{
	Use:   "check [pattern]",
	Short: "检查端口占用情况",
	Long: `检查端口占用情况，显示端口、进程ID、协议和程序信息
-v 显示程序的绝对路径
-w 通配符模式匹配`,
	Run: runCheck,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(checkCmd)

	// Release command flags
	releaseCmd.Flags().BoolVarP(&forceRelease, "force", "f", false, "强制释放，无需确认")
	releaseCmd.Args = cobra.MinimumNArgs(1)

	// Check command flags
	checkCmd.Flags().BoolVarP(&verboseCheck, "verbose", "v", false, "显示程序的绝对路径")
	checkCmd.Flags().BoolVarP(&wildcardCheck, "wildcard", "w", false, "通配符模式匹配")
}

func runRelease(cmd *cobra.Command, args []string) {
	releasePorts = args

	if err := core.ReleasePorts(releasePorts, forceRelease); err != nil {
		fmt.Fprintf(os.Stderr, "释放端口失败: %v\n", err)
		os.Exit(1)
	}
}

func runCheck(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		checkPorts = args
	}

	if err := core.CheckPorts(checkPorts, verboseCheck, wildcardCheck); err != nil {
		fmt.Fprintf(os.Stderr, "检查端口失败: %v\n", err)
		os.Exit(1)
	}
}