package utils

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/process"
)

func KillProcessByPath(path string) error {

	_, procName := filepath.Split(path)

	procs, err := process.Processes()
	if err != nil {
		return fmt.Errorf("error fetching processes: %v", err)
	}

	for _, p := range procs {
		cmdline, err := p.Cmdline()
		if err != nil {
			continue
		}

		// 判断命令行是否包含指定的路径
		if strings.Contains(cmdline, path) {
			// 获取进程的名称
			procNameFromCmd, err := p.Name()
			if err != nil {
				continue
			}

			if strings.EqualFold(procNameFromCmd, procName) {
				// 获取进程的 PID
				pid := p.Pid
				// 发送 SIGKILL 信号结束进程
				err := p.Kill()
				if err != nil {
					return fmt.Errorf("error killing process with PID %d, Proc %s: %v", pid, procName, err)
				}
				log.Printf("Successfully killed process with PID: %d, Proc: %v\n", pid, procName)
			}
		}
	}
	return nil
}

func IsProcessRunning(pid int) (bool, error) {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return false, fmt.Errorf("error attach process with pid: %d, err: %v", pid, err)
	}

	return p.IsRunning()
}
