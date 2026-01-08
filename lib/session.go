/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package lib

import (
	"bufio"
	"chain/lib/zlog"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

type ExitFunc func(error)
type StdoutFunc func(string)

type Session struct {
	mutex      sync.Mutex
	command    *exec.Cmd
	exited     chan struct{}
	exitErr    error
	waitStatus syscall.WaitStatus
	exitFuncs  []ExitFunc
}

type LogWriter struct {
	logger   *zlog.Logger
	errLevel bool
}

func (lw *LogWriter) Write(p []byte) (n int, err error) {
	if lw.errLevel {
		lw.logger.Error(string(p))
		fmt.Println("err", string(p))
	} else {
		lw.logger.Info(string(p))
		//fmt.Println("info", string(p))
	}

	return len(p), nil
}

// Start will start the provided command and return a Session that can be used
// to await completion or signal the process.
//
// The provided zlog is used zlog stderr from the running process.
func Start(logger *zlog.Logger, cmd *exec.Cmd, exitFuncs ...ExitFunc) (*Session, error) {
	//zlog = zlog.With("command", filepath.Base(cmd.Path))

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	/*
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
		stdout.Close()
	*/

	cmd.Stdout = &LogWriter{
		logger:   logger,
		errLevel: false,
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	sess := &Session{
		command:   cmd,
		exitFuncs: exitFuncs,
		exited:    make(chan struct{}),
	}
	go sess.waitForExit(logger, nil, stderr)

	return sess, nil
}

func (s *Session) waitForExit(logger *zlog.Logger, stdout, stderr io.Reader) {
	// copy stderr to the zlog until stderr is closed
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		logger.Error(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		logger.Errorf("command output scanning failed: %s", err)
	}

	// wait for the command to exit and to complete
	err := s.command.Wait()

	// update state and close the exited channel
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.exitErr = err
	s.waitStatus = s.command.ProcessState.Sys().(syscall.WaitStatus)
	for _, exit := range s.exitFuncs {
		exit(s.exitErr)
	}
	close(s.exited)
}

// Wait waits for the running command to terminate and returns the exit error
// from the command. If the command has already exited, the exit err will be
// returned immediately.
func (s *Session) Wait() error {
	<-s.exited
	return s.exitErr
}

// Signal will send a signal to the running process.
func (s *Session) Signal(sig os.Signal) {
	s.command.Process.Signal(sig)
}

func (s *Session) Pid() int {
	return s.command.Process.Pid
}
