package utils

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/shirou/gopsutil/process"
)

func KillProcessByPath(path string) error {
	normTarget, err := normalizePath(path)
	if err != nil {
		return err
	}

	ps, err := process.Processes()
	if err != nil {
		return fmt.Errorf("error fetching processes: %v", err)
	}

	killed := 0
	var lastErr error

	for _, p := range ps {
		exe, err := p.Exe()
		if err != nil || exe == "" {
			continue
		}

		normExe, err := normalizePath(exe)
		if err != nil {
			continue
		}

		if normExe == normTarget {
			if err := p.Terminate(); err != nil {
				fmt.Printf("error killing process with PID %d, Proc %s: %v\n", p.Pid, exe, err)
				lastErr = err
			}
			killed++
		}
	}

	if killed == 0 {
		if lastErr != nil {
			return lastErr
		}
		return fmt.Errorf("no process matched the given path %v", path)
	}

	return nil
}

func normalizePath(p string) (string, error) {
	if p == "" {
		return "", errors.New("empty path")
	}

	p = strings.ReplaceAll(p, "/", `\`)
	p = filepath.Clean(p)
	p = strings.ToLower(p)

	return p, nil
}

func KillProcessByPid(pid int32) error {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return fmt.Errorf("error fetching process: %v", err)
	}

	return p.Terminate()
}

func IsProcessRunning(pid int) (bool, error) {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return false, fmt.Errorf("error attach process with pid: %d, err: %v", pid, err)
	}

	return p.IsRunning()
}

type ProcessInfo struct {
	ImagePath       string
	CommandLine     string
	CurrentDir      string
	ParentImagePath string
	StartedTime     time.Time
	PID             int32
}

func GetProcessInfoByPID(pid int32) (*ProcessInfo, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid pid: %d", pid)
	}

	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("NewProcess(%d) failed: %w", pid, err)
	}

	// 这些字段在 Windows 上通常可用，但依旧可能因权限/进程保护失败
	imagePath, _ := p.Exe()
	cmdline, _ := p.Cmdline()

	// Cwd 在 Windows 上不一定能取到（依赖实现/权限），失败就返回空字符串
	cwd, _ := p.Cwd()

	ppid, _ := p.Ppid()
	parentImage := ""
	if ppid > 0 {
		if pp, err := process.NewProcess(ppid); err == nil {
			parentImage, _ = pp.Exe()
		}
	}

	// CreateTime: milliseconds since Unix epoch
	startedAt := time.Time{}
	if ms, err := p.CreateTime(); err == nil && ms > 0 {
		startedAt = time.Unix(0, ms*int64(time.Millisecond)).Local()
	}

	return &ProcessInfo{
		ImagePath:       imagePath,
		CommandLine:     cmdline,
		CurrentDir:      cwd,
		ParentImagePath: parentImage,
		StartedTime:     startedAt,
		PID:             pid,
	}, nil
}

func GetProcessCreatedTime(node, app string) string {

	timeStr := ""
	procNames := []string{node, app}
	for _, name := range procNames {
		pi, err := GetProcessInfoByProcName(name)
		if err != nil {
			continue
		}

		timeStr = fmt.Sprintf("%s %s", timeStr, pi.StartedTime.String())
	}

	return strings.TrimLeft(timeStr, " ")
}

// 匹配规则：大小写不敏感，支持传入 "notepad" 或 "notepad.exe"
// 多个匹配时：返回启动时间最早的那个
func GetProcessInfoByProcName(name string) (*ProcessInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("process name is empty")
	}

	target := strings.ToLower(name)
	targetNoExe := strings.TrimSuffix(target, ".exe")

	ps, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("list processes failed: %w", err)
	}

	type candidate struct {
		p  *process.Process
		ct int64 // create time in ms, 0 if unknown
	}
	cands := make([]candidate, 0, 8)

	for _, p := range ps {
		n, err := p.Name()
		if err != nil || n == "" {
			continue
		}
		nl := strings.ToLower(n)
		nlNoExe := strings.TrimSuffix(nl, ".exe")

		if nl == target || nlNoExe == targetNoExe {
			ct, _ := p.CreateTime() // ms since epoch; may fail -> 0
			cands = append(cands, candidate{p: p, ct: ct})
		}
	}

	if len(cands) == 0 {
		return nil, fmt.Errorf("process %q not found", name)
	}

	// 多个候选：按启动时间排序（ct=0 的放最后），取最早启动的
	slice.SortBy(cands, func(a, b candidate) bool {
		return a.ct < b.ct
	})

	return buildProcessInfo(cands[0].p)
}

func buildProcessInfo(p *process.Process) (*ProcessInfo, error) {
	pid := p.Pid

	imagePath, _ := p.Exe()
	cmdline, _ := p.Cmdline()
	cwd, _ := p.Cwd()

	ppid, _ := p.Ppid()
	parentImage := ""
	if ppid > 0 {
		if pp, err := process.NewProcess(ppid); err == nil {
			parentImage, _ = pp.Exe()
		}
	}

	startedAt := time.Time{}
	if ms, err := p.CreateTime(); err == nil && ms > 0 {
		startedAt = time.Unix(0, ms*int64(time.Millisecond)).UTC()
	}

	return &ProcessInfo{
		ImagePath:       imagePath,
		CommandLine:     cmdline,
		CurrentDir:      cwd,
		ParentImagePath: parentImage,
		StartedTime:     startedAt,
		PID:             pid,
	}, nil
}
