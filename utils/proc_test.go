package utils

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestGetProcessInfoByProcName(t *testing.T) {

	procNames := []string{"cmd.exe", "notepad.exe"}
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

func TestJson(t *testing.T) {
	ss := `{
	"data": {
		"index": 1,
		"height": 12,
	},
	"result": {
		"a.b": 1,
		"c.d": [
					"hash1"
				],
	},
}`

	v1 := gjson.Get(ss, "result.a\\.b").Int()
	v2 := gjson.Get(ss, `result.c\.d.0`).String()

	t.Log(v1, v2)
}
