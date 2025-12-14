/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package lib

import (
	"encoding/json"
	"os/exec"
	"syscall"
	"time"

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
	Launcher *Launcher

	Session     *Session
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
		Launcher: launcher,
	}
}

func (i *Instance) Start() error {
	sess, err := i.Launcher.RunApp()
	if err != nil {
		return errors.WithMessage(err, "could not execute run")
	}
	i.Session = sess
	return nil
}

// Stop signals the process to terminate with SIGTERM. If the process doesn't
// terminate within TermTimeout, the process is killed with SIGKILL.
func (i *Instance) Stop() error {
	if i.Session == nil {
		return errors.Errorf("instance has not been started")
	}

	done := make(chan struct{})
	go func() { i.Wait(); close(done) }()

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
