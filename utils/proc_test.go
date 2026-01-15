package utils

import (
	"testing"
)

func TestGetProcessInfoByProcName(t *testing.T) {

	procNames := []string{"cmd.exe", "notepad.exe", "sublime_text"}
	for _, name := range procNames {
		pi, err := GetProcessInfoByProcName(name)
		if err != nil {
			t.Log(err.Error())
		} else {
			t.Log(pi)
		}
	}

}

func TestKillProcessByPath(t *testing.T) {
	procNames := []string{"cometbft.exe", "abci-v2-Identity-app.exe"}
	for _, name := range procNames {
		err := KillProcessByProcNameAndCmdline(name, "8d7f11f96878634cdf167958e6764ed71195121e")
		if err != nil {
			t.Log(err.Error())
		}

	}
}
