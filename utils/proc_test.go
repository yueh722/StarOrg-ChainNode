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
	procNames := []string{"c:\\windows\\system32\\cmd.exe", "notepad.exe", "C:\\Program Files\\Sublime Text\\sublime_text.exe"}
	for _, name := range procNames {
		err := KillProcessByPath(name)
		if err != nil {
			t.Log(err.Error())
		}
	}
}
