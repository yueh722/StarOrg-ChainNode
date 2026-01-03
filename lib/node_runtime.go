package lib

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

type ProcInfo struct {
	Pid       int
	StartTime time.Time
}

type NodeRuntime interface {
	GetAppInfo() ProcInfo
	GetNodeInfo() ProcInfo
	Error() error
	KillNode()
	IsRunning() bool
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
	var err error

	//get node error
	err = fmt.Errorf("node error: %v", r.NodeInst.GetErr())

	//get app error
	if err != nil {
		err = errors.Wrapf(err, "app error: %v", r.AppInst.GetErr().Error())
	}

	return err
}

func (r NodeRuntimeImpl) KillNode() {
	if !r.IsRunning() {
		return
	}
	_ = r.AppInst.Stop()
	time.Sleep(2 * time.Second)
	_ = r.NodeInst.Stop()
}

func (r NodeRuntimeImpl) IsRunning() bool {
	return r.AppInst.IsRunning() && r.NodeInst.IsRunning()
}
