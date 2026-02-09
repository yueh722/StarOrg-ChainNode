package lib

import (
	"fmt"
	"strings"
	"time"
)

type ProcInfo struct {
	Pid       int
	StartTime time.Time
}

type NodeRuntime interface {
	GetAppInfo() ProcInfo
	GetNodeInfo() ProcInfo
	Error() error
	StopNode() error
	IsRunning() bool
	CreatedTime() string
}

type NodeRuntimeImpl struct {
	NodeInst *Instance
	AppInst  *Instance
}

func (r NodeRuntimeImpl) GetNodeInfo() ProcInfo {
	pi := ProcInfo{}
	if r.NodeInst == nil || r.NodeInst.Session == nil {
		return pi
	}

	pi = ProcInfo{
		Pid:       r.NodeInst.Pid,
		StartTime: r.NodeInst.StartTime,
	}

	return pi
}

func (r NodeRuntimeImpl) GetAppInfo() ProcInfo {
	pi := ProcInfo{}
	if r.AppInst == nil || r.AppInst.Session == nil {
		return pi
	}

	pi = ProcInfo{
		Pid:       r.AppInst.Pid,
		StartTime: r.AppInst.StartTime,
	}

	return pi
}

func (r NodeRuntimeImpl) Error() error {
	var errs = NewMultiError()

	insts := []*Instance{r.NodeInst, r.AppInst}

	for _, inst := range insts {
		if inst == nil {
			continue
		}

		if err := inst.GetErr(); err != nil {
			errs.AddError(err)
		}

	}

	return errs.ErrorOrNil()
}

func (r NodeRuntimeImpl) StopNode() error {
	if !r.IsRunning() {
		return nil
	}
	err1 := r.AppInst.Stop()
	time.Sleep(2 * time.Second)
	err2 := r.NodeInst.Stop()

	if err1 != nil || err2 != nil {
		return fmt.Errorf("%v,%v", err1, err2)
	}

	return nil
}

func (r NodeRuntimeImpl) IsRunning() bool {
	return r.AppInst.IsRunning() && r.NodeInst.IsRunning()
}

func (r NodeRuntimeImpl) CreatedTime() string {

	timeStr := ""
	insts := []*Instance{r.NodeInst, r.AppInst}
	for _, inst := range insts {
		if inst == nil || !inst.IsRunning() {
			continue
		}
		timeStr = fmt.Sprintf("%s,%s", timeStr, inst.StartTime.Format("2006-01-02 15:04:05"))
	}

	return strings.Trim(timeStr, " ")
}

func GetNodeCreatedTime(nr NodeRuntime) string {
	if nr == nil {
		return ""
	}
	return nr.CreatedTime()
}
