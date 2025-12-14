package lib

import (
	"testing"
)

func TestLauncher_RunCommand(t *testing.T) {
	l := Launcher{
		Name:        "Run-NodeCLI",
		AppFilePath: "./bin/NodeCLI.exe",
		Args: map[string]string{
			"--home":          " ./app01",
			"--socket-addr":   "tcp://127.0.0.1:26658",
			"--http-addr":     "127.0.0.1:8080",
			"--abci-rpc-addr": "http://127.0.0.1:26657",
		},
		Envs:   nil,
		Logger: nil,
	}

	cmd := l.NewCommandFromMap(l.AppFilePath, l.Args)
	err := l.RunCommand(cmd)
	if err != nil {
		t.Error(err)
	}

}
