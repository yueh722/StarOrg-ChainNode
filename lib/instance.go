/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package lib

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"github.com/duke-git/lancet/v2/mathutil"
	"github.com/pkg/errors"
)

const (
	DialTimeout = 3 * time.Second
	CometbftDir = ""
)

type AppInfo struct {
}

// An Instance presents a running process.
type Instance struct {
	Launcher    *Launcher
	Session     *Session
	Pid         int
	StartTime   time.Time
	Err         error
	TermTimeout time.Duration
}

// Duration used for the DialTimeout property
type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
	case string:
		dur, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(dur)
	default:
		return errors.New("invalid duration")
	}

	return nil
}

func NewInstance(launcher *Launcher) *Instance {
	return &Instance{
		Launcher:    launcher,
		TermTimeout: 5 * time.Second,
	}
}

func (i *Instance) Start() error {
	var sess *Session
	sess, i.Err = i.Launcher.RunApp()
	if i.Err != nil {
		return errors.WithMessage(i.Err, "could not execute run")
	}
	i.Session = sess

	if sess != nil {
		i.Pid = sess.Pid()
		i.StartTime = time.Now()
	}

	return nil
}

// Stop signals the process to terminate with SIGTERM. If the process doesn't
// terminate within TermTimeout, the process is killed with SIGKILL.
func (i *Instance) Stop() error {
	if i.Session == nil {
		return errors.Errorf("instance has not been started")
	}

	if mathutil.Abs(time.Now().Sub(i.StartTime)) < (2 * time.Second) {
		fmt.Println("delay to stop instance", i.Launcher.Args)
		<-time.After(2 * time.Second)
		fmt.Println("delay time exceed", i.Launcher.Args)
	}

	done := make(chan struct{})
	go func() {
		i.Wait()
		close(done)
	}()

	i.Session.Signal(syscall.SIGTERM)
	select {
	case <-time.After(i.TermTimeout):
		i.Session.Signal(syscall.SIGKILL)
	case <-done:
		return nil
	}

	select {
	case <-time.After(5 * time.Second):
		return errors.Errorf("failed to stop instance '%v'", i.Launcher.Name)
	case <-done:
		return nil
	}
}

func (i *Instance) Wait() (int, error) {
	if i.Session == nil {
		return -1, errors.Errorf("instance was not successfully started")
	}

	err := i.Session.Wait()
	err = errors.Wrapf(err, "launcher '%s' run failed", i.Launcher.Name)
	if exitErr, ok := errors.Cause(err).(*exec.ExitError); ok {
		return exitErr.ExitCode(), err
	}
	return 0, err
}

func (i *Instance) IsRunning() bool {
	return i.Session != nil && i.Session.exitTime == 0

	//return i.Err == nil && i.Session != nil && i.Session.exitErr == nil

	//running, err := utils.IsProcessRunning(i.Pid)
	//return running && err == nil
}

func (i *Instance) GetErr() error {
	if i.Err != nil {
		return i.Err
	}

	if i.Session != nil {
		return i.Session.exitErr
	}

	return errors.New("unknow error")
}
