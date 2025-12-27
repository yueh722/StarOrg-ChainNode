package lib

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

type NodeRuntime interface {
	GetAppSession() *Session
	GetNodeSession() *Session
	Error() error
	KillNode()
	IsRunning() bool
}

type NodeRuntimeImpl struct {
	NodeInst *Instance
	AppInst  *Instance
}

func (r NodeRuntimeImpl) GetAppSession() *Session {
	return r.NodeInst.Session
}

func (r NodeRuntimeImpl) GetNodeSession() *Session {
	return r.AppInst.Session
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
	_ = r.AppInst.Stop()
	time.Sleep(2 * time.Second)
	_ = r.NodeInst.Stop()
}

func (r NodeRuntimeImpl) IsRunning() bool {
	return r.AppInst.IsRunning() && r.NodeInst.IsRunning()
}
